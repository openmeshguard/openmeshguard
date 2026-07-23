package resolver

import (
	"fmt"
	"sort"
	"strings"
)

const (
	authorizationPoliciesUnavailableReason = "AuthorizationPolicy resources unavailable"
	waypointEvidenceUnavailableReason      = "Gateway API waypoint evidence unavailable"
	authzDataPlaneUnknownReason            = "data plane membership unavailable for authorization enforcement"
	customOnlyUnknownReason                = "CUSTOM-only authorization posture depends on an external provider and is not representable by the current effective-posture enum"
)

// ResolveAuthz implements Istio's additive CUSTOM -> DENY -> ALLOW evaluation
// model. Scope filtering remains explicit in the chain so a selector mismatch
// is distinguishable from the complete absence of authorization policy.
//
// Istio policy evaluation and additive merge semantics:
// https://istio.io/latest/docs/concepts/security/#implicit-enablement
// Empty ALLOW policy and rules: [{}] distinction:
// https://istio.io/latest/docs/reference/config/security/authorization-policy/
func (ResolverV2) ResolveAuthz(in WorkloadInput) AuthzResult {
	if in.AuthzPolicies == nil {
		return unknownAuthz(authorizationPoliciesUnavailableReason, nil, nil, nil, nil)
	}
	if in.DataPlaneMode == ModeUnknown || in.DataPlaneMode == ModeMixed || in.DataPlaneMode == "" {
		return unknownAuthz(authzDataPlaneUnknownReason, nil, nil, nil, nil)
	}

	policies := append([]AuthorizationPolicyView(nil), in.AuthzPolicies...)
	sort.SliceStable(policies, func(i, j int) bool {
		leftAction, rightAction := authzActionRank(policies[i].Action), authzActionRank(policies[j].Action)
		if leftAction != rightAction {
			return leftAction < rightAction
		}
		leftScope, rightScope := authzScopeRank(policies[i]), authzScopeRank(policies[j])
		if leftScope != rightScope {
			return leftScope < rightScope
		}
		if policies[i].Namespace != policies[j].Namespace {
			return policies[i].Namespace < policies[j].Namespace
		}
		return policies[i].Name < policies[j].Name
	})

	chain := []Step{{
		Kind:   "AuthorizationDefault",
		Field:  "implicitEnablement",
		Effect: "establishes Istio's implicit allow when no applicable ALLOW policy exists",
	}}
	var policiesInScope []string
	var waypointUnenforced []string
	broadAllow := false
	identityScoped := true
	hasCustom := false
	hasDeny := false
	hasEmptyAllow := false
	hasExplicitAllow := false
	unknownReason := ""

	for _, policy := range policies {
		if !authzNamespaceCandidate(policy, in) {
			continue
		}
		if policy.HasSelector && !policy.SelectorMatch {
			chain = append(chain, Step{
				Kind:      "AuthorizationPolicy",
				Name:      policy.Name,
				Namespace: policy.Namespace,
				Field:     "spec.selector",
				Effect:    "selector does not match the workload; excludes policy from authorization evaluation",
			})
			continue
		}
		if policy.TargetsWaypoint && in.DataPlaneMode == ModeSidecar {
			chain = append(chain, Step{
				Kind:      "AuthorizationPolicy",
				Name:      policy.Name,
				Namespace: policy.Namespace,
				Field:     authzTargetRefField(policy),
				Effect:    "targetRef policy attaches to a waypoint; excludes it from sidecar authorization evaluation",
			})
			continue
		}

		action := normalizedAuthzAction(policy.Action)
		if action == "" {
			if unknownReason == "" {
				unknownReason = fmt.Sprintf("unsupported AuthorizationPolicy action %q on %s/%s", policy.Action, policy.Namespace, policy.Name)
			}
			continue
		}

		policyRef := namespacedPolicyName(policy)
		policiesInScope = append(policiesInScope, policyRef)
		step := authzPolicyStep(policy, action)

		if in.DataPlaneMode == ModeAmbient && policy.TargetsWaypoint {
			waypointStep, enforced, unavailable := resolveWaypointEnforcement(policy.TargetWaypoint, policy)
			chain = append(chain, step, waypointStep)
			if unavailable && unknownReason == "" {
				unknownReason = waypointEvidenceUnavailableReason
			}
			if !enforced && !unavailable {
				waypointUnenforced = append(waypointUnenforced, policyRef)
			}
			updateAuthzActionState(action, policy, &hasCustom, &hasDeny, &hasEmptyAllow, &hasExplicitAllow, &broadAllow, &identityScoped)
			continue
		}

		if policy.RequiresL7 {
			switch in.DataPlaneMode {
			case ModeSidecar:
				step.Effect += "; the sidecar enforces its L7 attributes"
			case ModeAmbient:
				// Istio documents selector-based L7 policy at ztunnel as
				// fail-safe DENY, not as silently unenforced.
				// https://istio.io/latest/docs/ambient/usage/l7-features/
				hasDeny = true
				step.Effect += "; ambient ztunnel converts selector-based L7 policy to fail-safe DENY"
			default:
				step.Effect += "; workload is outside an enforceable Istio data plane"
			}
		}

		chain = append(chain, step)
		updateAuthzActionState(action, policy, &hasCustom, &hasDeny, &hasEmptyAllow, &hasExplicitAllow, &broadAllow, &identityScoped)
	}

	chain = orderChain(chain)
	policiesInScope = uniqueStrings(policiesInScope)
	waypointUnenforced = uniqueStrings(waypointUnenforced)
	knownBroadAllow := broadAllow
	knownIdentityScoped := hasEmptyAllow
	if hasExplicitAllow {
		knownIdentityScoped = identityScoped
	}

	if unknownReason != "" {
		return unknownAuthz(unknownReason, policiesInScope, chain, &knownBroadAllow, &knownIdentityScoped)
	}
	if len(waypointUnenforced) > 0 {
		return AuthzResult{
			Effective:          AuthzWaypointUnenforced,
			BroadAllow:         &knownBroadAllow,
			IdentityScoped:     &knownIdentityScoped,
			PoliciesInScope:    policiesInScope,
			WaypointUnenforced: waypointUnenforced,
			Chain:              chain,
		}
	}
	if hasDeny || (hasEmptyAllow && !hasExplicitAllow) {
		return AuthzResult{
			Effective:       AuthzDenyPresent,
			BroadAllow:      &knownBroadAllow,
			IdentityScoped:  &knownIdentityScoped,
			PoliciesInScope: policiesInScope,
			Chain:           chain,
		}
	}
	if hasEmptyAllow && hasExplicitAllow {
		return AuthzResult{
			Effective:       AuthzDefaultDenyExplicitAllow,
			BroadAllow:      &knownBroadAllow,
			IdentityScoped:  &knownIdentityScoped,
			PoliciesInScope: policiesInScope,
			Chain:           chain,
		}
	}
	if hasExplicitAllow {
		return AuthzResult{
			Effective:       AuthzAllowOnly,
			BroadAllow:      &knownBroadAllow,
			IdentityScoped:  &knownIdentityScoped,
			PoliciesInScope: policiesInScope,
			Chain:           chain,
		}
	}
	if hasCustom {
		return unknownAuthz(customOnlyUnknownReason, policiesInScope, chain, &knownBroadAllow, &knownIdentityScoped)
	}
	return AuthzResult{
		Effective:       AuthzNoPolicy,
		BroadAllow:      &knownBroadAllow,
		IdentityScoped:  &knownIdentityScoped,
		PoliciesInScope: policiesInScope,
		Chain:           chain,
	}
}

func authzNamespaceCandidate(policy AuthorizationPolicyView, in WorkloadInput) bool {
	root := rootNamespace(in.MeshDefaults.RootNamespace)
	return policy.RootNamespace || policy.Namespace == root || policy.Namespace == in.Ref.Namespace
}

func authzActionRank(action string) int {
	switch normalizedAuthzAction(action) {
	case "CUSTOM":
		return 0
	case "DENY":
		return 1
	case "ALLOW":
		return 2
	case "AUDIT":
		return 3
	default:
		return 4
	}
}

func authzScopeRank(policy AuthorizationPolicyView) int {
	if policy.RootNamespace {
		return 0
	}
	return 1
}

func normalizedAuthzAction(action string) string {
	action = strings.ToUpper(strings.TrimSpace(action))
	if action == "" {
		return "ALLOW"
	}
	switch action {
	case "ALLOW", "DENY", "CUSTOM", "AUDIT":
		return action
	default:
		return ""
	}
}

func authzPolicyStep(policy AuthorizationPolicyView, action string) Step {
	effect := ""
	switch action {
	case "CUSTOM":
		if !policy.HasRules {
			effect = "CUSTOM policy has no rules and never matches"
		} else {
			effect = "evaluates CUSTOM authorization before additive DENY and ALLOW policies"
		}
	case "DENY":
		if !policy.HasRules {
			effect = "DENY policy has no rules and never matches"
		} else {
			effect = "adds DENY rules; any matching DENY overrides additive ALLOW policies"
		}
	case "ALLOW":
		switch {
		case !policy.HasRules:
			effect = "adds an ALLOW policy with no rules; activates default deny for unmatched requests"
		case policy.BroadAllow:
			effect = "adds a structurally broad rule to the additive ALLOW union"
		default:
			effect = "adds explicit rules to the additive ALLOW union"
		}
	case "AUDIT":
		effect = "records matching requests without changing the authorization decision"
	}
	return Step{
		Kind:      "AuthorizationPolicy",
		Name:      policy.Name,
		Namespace: policy.Namespace,
		Field:     authzPolicyField(policy),
		Effect:    effect,
	}
}

func authzPolicyField(policy AuthorizationPolicyView) string {
	if !policy.TargetsWaypoint || policy.TargetRefKind == "" || policy.TargetRefName == "" {
		return "spec.action"
	}
	return authzTargetRefField(policy)
}

func authzTargetRefField(policy AuthorizationPolicyView) string {
	if policy.TargetRefKind == "" || policy.TargetRefName == "" {
		return "spec.targetRefs"
	}
	return fmt.Sprintf(`spec.targetRefs["%s/%s"]`, policy.TargetRefKind, policy.TargetRefName)
}

func resolveWaypointEnforcement(waypoint *WaypointView, policy AuthorizationPolicyView) (Step, bool, bool) {
	step := Step{
		Kind:  "Waypoint",
		Field: "istio.io/use-waypoint",
	}
	if waypoint == nil {
		step.Effect = fmt.Sprintf("no waypoint serves the workload; waypoint-attached policy %s/%s is not enforced", policy.Namespace, policy.Name)
		return step, false, false
	}
	step.Name = waypoint.Name
	step.Namespace = waypoint.Namespace
	if !waypoint.Known {
		step.Effect = fmt.Sprintf("waypoint evidence is unavailable for waypoint-attached policy %s/%s", policy.Namespace, policy.Name)
		return step, false, true
	}
	step.Field = "status.conditions[Programmed]"
	if !waypoint.Ready {
		step.Effect = fmt.Sprintf("selected %s waypoint is not ready; waypoint-attached policy %s/%s is not enforced", waypoint.Scope, policy.Namespace, policy.Name)
		return step, false, false
	}
	step.Effect = fmt.Sprintf("selected %s waypoint is ready and enforces waypoint-attached policy %s/%s", waypoint.Scope, policy.Namespace, policy.Name)
	return step, true, false
}

func updateAuthzActionState(
	action string,
	policy AuthorizationPolicyView,
	hasCustom *bool,
	hasDeny *bool,
	hasEmptyAllow *bool,
	hasExplicitAllow *bool,
	broadAllow *bool,
	identityScoped *bool,
) {
	switch action {
	case "CUSTOM":
		*hasCustom = *hasCustom || policy.HasRules
	case "DENY":
		*hasDeny = *hasDeny || policy.HasRules
	case "ALLOW":
		if !policy.HasRules {
			*hasEmptyAllow = true
			return
		}
		*hasExplicitAllow = true
		*broadAllow = *broadAllow || policy.BroadAllow
		*identityScoped = *identityScoped && policy.IdentityScoped
	}
}

func namespacedPolicyName(policy AuthorizationPolicyView) string {
	return policy.Namespace + "/" + policy.Name
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func unknownAuthz(reason string, policies []string, chain []Step, broadAllow, identityScoped *bool) AuthzResult {
	if chain == nil {
		chain = []Step{}
	}
	return AuthzResult{
		Effective:       AuthzUnknown,
		BroadAllow:      broadAllow,
		IdentityScoped:  identityScoped,
		PoliciesInScope: policies,
		Chain:           chain,
		UnknownReason:   reason,
	}
}
