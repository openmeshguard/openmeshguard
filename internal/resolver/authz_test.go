package resolver

import (
	"reflect"
	"testing"
)

func TestResolverV2ResolveAuthz(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name           string
		in             WorkloadInput
		wantEffective  AuthzEffective
		wantBroadAllow *bool
		wantIdentity   *bool
		wantPolicies   []string
		wantUnenforced []string
		wantUnknown    string
		wantChain      []Step
	}{
		{
			name:           "no policy anywhere preserves implicit allow",
			in:             authzWorkload(ModeSidecar),
			wantEffective:  AuthzNoPolicy,
			wantBroadAllow: &falseValue,
			wantIdentity:   &falseValue,
			wantChain:      []Step{authzDefaultStep(1)},
		},
		{
			name: "root namespace default deny unions with local explicit allow",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "api", "ALLOW", true),
				rootAuthzPolicy("istio-system", "default-deny", "ALLOW", false),
			),
			wantEffective:  AuthzDefaultDenyExplicitAllow,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"istio-system/default-deny", "payments/api"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "istio-system", "default-deny", "adds an ALLOW policy with no rules; activates default deny for unmatched requests"),
				authzPolicyStepWant(3, "payments", "api", "adds explicit rules to the additive ALLOW union"),
			},
		},
		{
			name: "DENY is evaluated before and overrides additive ALLOW",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "allow-api", "ALLOW", true),
				authzPolicy("payments", "deny-admin", "DENY", true),
			),
			wantEffective:  AuthzDenyPresent,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/deny-admin", "payments/allow-api"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "deny-admin", "adds DENY rules; any matching DENY overrides additive ALLOW policies"),
				authzPolicyStepWant(3, "payments", "allow-api", "adds explicit rules to the additive ALLOW union"),
			},
		},
		{
			name: "CUSTOM is evaluated before native ALLOW",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "allow-api", "ALLOW", true),
				authzPolicy("payments", "external-check", "CUSTOM", true),
			),
			wantEffective:  AuthzAllowOnly,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/external-check", "payments/allow-api"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "external-check", "evaluates CUSTOM authorization before additive DENY and ALLOW policies"),
				authzPolicyStepWant(3, "payments", "allow-api", "adds explicit rules to the additive ALLOW union"),
			},
		},
		{
			name: "namespace ALLOW without empty default deny is allow only",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "allow-api", "ALLOW", true),
			),
			wantEffective:  AuthzAllowOnly,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/allow-api"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "allow-api", "adds explicit rules to the additive ALLOW union"),
			},
		},
		{
			name: "selector mismatch is distinct from absent policy",
			in: authzWorkload(ModeSidecar,
				AuthorizationPolicyView{
					Name: "other-app", Namespace: "payments", Action: "ALLOW",
					HasSelector: true, SelectorMatch: false, HasRules: true,
				},
			),
			wantEffective:  AuthzNoPolicy,
			wantBroadAllow: &falseValue,
			wantIdentity:   &falseValue,
			wantChain: []Step{
				authzDefaultStep(1),
				{
					Order: 2, Kind: "AuthorizationPolicy", Namespace: "payments", Name: "other-app",
					Field: "spec.selector", Effect: "selector does not match the workload; excludes policy from authorization evaluation",
				},
			},
		},
		{
			name: "ambient targetRef L7 policy without waypoint is unenforced",
			in: authzWorkload(ModeAmbient,
				waypointAuthzPolicy("payments", "http-get", true),
			),
			wantEffective:  AuthzWaypointUnenforced,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/http-get"},
			wantUnenforced: []string{"payments/http-get"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "http-get", "adds explicit rules to the additive ALLOW union"),
				{Order: 3, Kind: "Waypoint", Field: "istio.io/use-waypoint", Effect: "no waypoint serves the workload; waypoint-attached policy payments/http-get is not enforced"},
			},
		},
		{
			name: "ambient targetRef L4 policy without waypoint is unenforced",
			in: authzWorkload(ModeAmbient,
				waypointL4AuthzPolicy("payments", "tcp-principal", true),
			),
			wantEffective:  AuthzWaypointUnenforced,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/tcp-principal"},
			wantUnenforced: []string{"payments/tcp-principal"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "tcp-principal", "adds explicit rules to the additive ALLOW union"),
				{Order: 3, Kind: "Waypoint", Field: "istio.io/use-waypoint", Effect: "no waypoint serves the workload; waypoint-attached policy payments/tcp-principal is not enforced"},
			},
		},
		{
			name: "ambient targetRef L4 policy with ready waypoint is enforced",
			in: authzWorkloadWithWaypoint(ModeAmbient, &WaypointView{
				Name: "payments", Namespace: "payments", Known: true, Ready: true, Scope: "service",
			}, waypointL4AuthzPolicy("payments", "tcp-principal", true)),
			wantEffective:  AuthzAllowOnly,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/tcp-principal"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "tcp-principal", "adds explicit rules to the additive ALLOW union"),
				{Order: 3, Kind: "Waypoint", Namespace: "payments", Name: "payments", Field: "status.conditions[Programmed]", Effect: "selected service waypoint is ready and enforces waypoint-attached policy payments/tcp-principal"},
			},
		},
		{
			name: "ambient targetRef L4 policy with unready waypoint is unenforced",
			in: authzWorkloadWithWaypoint(ModeAmbient, &WaypointView{
				Name: "payments", Namespace: "payments", Known: true, Ready: false, Scope: "service",
			}, waypointL4AuthzPolicy("payments", "tcp-principal", true)),
			wantEffective:  AuthzWaypointUnenforced,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/tcp-principal"},
			wantUnenforced: []string{"payments/tcp-principal"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "tcp-principal", "adds explicit rules to the additive ALLOW union"),
				{Order: 3, Kind: "Waypoint", Namespace: "payments", Name: "payments", Field: "status.conditions[Programmed]", Effect: "selected service waypoint is not ready; waypoint-attached policy payments/tcp-principal is not enforced"},
			},
		},
		{
			name: "ambient targetRef L4 policy with unavailable waypoint evidence is unknown",
			in: authzWorkloadWithWaypoint(ModeAmbient, &WaypointView{Known: false},
				waypointL4AuthzPolicy("payments", "tcp-principal", true)),
			wantEffective:  AuthzUnknown,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/tcp-principal"},
			wantUnknown:    waypointEvidenceUnavailableReason,
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "tcp-principal", "adds explicit rules to the additive ALLOW union"),
				{Order: 3, Kind: "Waypoint", Field: "istio.io/use-waypoint", Effect: "waypoint evidence is unavailable for waypoint-attached policy payments/tcp-principal"},
			},
		},
		{
			name: "ambient targetRef L7 policy with ready waypoint is enforced",
			in: authzWorkloadWithWaypoint(ModeAmbient, &WaypointView{
				Name: "payments", Namespace: "payments", Known: true, Ready: true, Scope: "service",
			}, waypointAuthzPolicy("payments", "http-get", true)),
			wantEffective:  AuthzAllowOnly,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/http-get"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "http-get", "adds explicit rules to the additive ALLOW union"),
				{Order: 3, Kind: "Waypoint", Namespace: "payments", Name: "payments", Field: "status.conditions[Programmed]", Effect: "selected service waypoint is ready and enforces waypoint-attached policy payments/http-get"},
			},
		},
		{
			name: "mixed L4 and L7 targetRef policy with unready waypoint is unenforced",
			in: authzWorkloadWithWaypoint(ModeAmbient, &WaypointView{
				Name: "payments", Namespace: "payments", Known: true, Ready: false, Scope: "service",
			}, waypointAuthzPolicy("payments", "mixed-rules", true)),
			wantEffective:  AuthzWaypointUnenforced,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/mixed-rules"},
			wantUnenforced: []string{"payments/mixed-rules"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "mixed-rules", "adds explicit rules to the additive ALLOW union"),
				{Order: 3, Kind: "Waypoint", Namespace: "payments", Name: "payments", Field: "status.conditions[Programmed]", Effect: "selected service waypoint is not ready; waypoint-attached policy payments/mixed-rules is not enforced"},
			},
		},
		{
			name: "root and namespace ALLOW policies union without override",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "local", "ALLOW", true),
				AuthorizationPolicyView{
					Name: "mesh-broad", Namespace: "istio-system", Action: "ALLOW",
					RootNamespace: true, HasRules: true, BroadAllow: true, IdentityScoped: false,
				},
			),
			wantEffective:  AuthzAllowOnly,
			wantBroadAllow: &trueValue,
			wantIdentity:   &falseValue,
			wantPolicies:   []string{"istio-system/mesh-broad", "payments/local"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "istio-system", "mesh-broad", "adds a structurally broad rule to the additive ALLOW union"),
				authzPolicyStepWant(3, "payments", "local", "adds explicit rules to the additive ALLOW union"),
			},
		},
		{
			name: "empty ALLOW policy denies all without claiming explicit allow",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "allow-nothing", "ALLOW", false),
			),
			wantEffective:  AuthzDenyPresent,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/allow-nothing"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "allow-nothing", "adds an ALLOW policy with no rules; activates default deny for unmatched requests"),
			},
		},
		{
			name: "rules containing one empty rule allow all and are broad",
			in: authzWorkload(ModeSidecar,
				AuthorizationPolicyView{Name: "allow-all", Namespace: "payments", Action: "ALLOW", HasRules: true, BroadAllow: true},
			),
			wantEffective:  AuthzAllowOnly,
			wantBroadAllow: &trueValue,
			wantIdentity:   &falseValue,
			wantPolicies:   []string{"payments/allow-all"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "allow-all", "adds a structurally broad rule to the additive ALLOW union"),
			},
		},
		{
			name: "default deny does not make a broad explicit allow identity scoped",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "default-deny", "ALLOW", false),
				AuthorizationPolicyView{Name: "allow-all", Namespace: "payments", Action: "ALLOW", HasRules: true, BroadAllow: true},
			),
			wantEffective:  AuthzDefaultDenyExplicitAllow,
			wantBroadAllow: &trueValue,
			wantIdentity:   &falseValue,
			wantPolicies:   []string{"payments/allow-all", "payments/default-deny"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "allow-all", "adds a structurally broad rule to the additive ALLOW union"),
				authzPolicyStepWant(3, "payments", "default-deny", "adds an ALLOW policy with no rules; activates default deny for unmatched requests"),
			},
		},
		{
			name: "CUSTOM only remains explicit unknown",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "external-check", "CUSTOM", true),
			),
			wantEffective:  AuthzUnknown,
			wantBroadAllow: &falseValue,
			wantIdentity:   &falseValue,
			wantPolicies:   []string{"payments/external-check"},
			wantUnknown:    customOnlyUnknownReason,
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "external-check", "evaluates CUSTOM authorization before additive DENY and ALLOW policies"),
			},
		},
		{
			name: "unsupported action remains traceable in unknown posture",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "future-action", "DELEGATE", true),
			),
			wantEffective:  AuthzUnknown,
			wantBroadAllow: &falseValue,
			wantIdentity:   &falseValue,
			wantPolicies:   []string{"payments/future-action"},
			wantUnknown:    `unsupported AuthorizationPolicy action "DELEGATE" on payments/future-action`,
			wantChain: []Step{
				authzDefaultStep(1),
				{
					Order: 2, Kind: "AuthorizationPolicy", Namespace: "payments", Name: "future-action",
					Field: "spec.action", Effect: `unsupported action "DELEGATE" prevents authorization posture resolution`,
				},
			},
		},
		{
			name: "ambient selector L7 policy becomes fail safe deny",
			in: authzWorkload(ModeAmbient,
				AuthorizationPolicyView{
					Name: "legacy-http", Namespace: "payments", Action: "ALLOW",
					HasSelector: true, SelectorMatch: true, HasRules: true, RequiresL7: true, IdentityScoped: true,
				},
			),
			wantEffective:  AuthzDenyPresent,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/legacy-http"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "legacy-http", "adds explicit rules to the additive ALLOW union; ambient ztunnel converts selector-based L7 policy to fail-safe DENY"),
			},
		},
		{
			name: "unavailable waypoint evidence is unknown rather than unenforced",
			in: authzWorkloadWithWaypoint(ModeAmbient, &WaypointView{Known: false},
				waypointAuthzPolicy("payments", "http-get", true)),
			wantEffective:  AuthzUnknown,
			wantBroadAllow: &falseValue,
			wantIdentity:   &trueValue,
			wantPolicies:   []string{"payments/http-get"},
			wantUnknown:    waypointEvidenceUnavailableReason,
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "http-get", "adds explicit rules to the additive ALLOW union"),
				{Order: 3, Kind: "Waypoint", Field: "istio.io/use-waypoint", Effect: "waypoint evidence is unavailable for waypoint-attached policy payments/http-get"},
			},
		},
		{
			name:           "unavailable AuthorizationPolicy collection is unknown",
			in:             WorkloadInput{Ref: workloadRef(), DataPlaneMode: ModeSidecar},
			wantEffective:  AuthzUnknown,
			wantBroadAllow: nil,
			wantIdentity:   nil,
			wantUnknown:    authorizationPoliciesUnavailableReason,
			wantChain:      []Step{},
		},
		{
			name: "service targetRef policy is not attributed to a sidecar",
			in: authzWorkload(ModeSidecar, AuthorizationPolicyView{
				Name: "service-default-deny", Namespace: "payments", Action: "ALLOW",
				TargetsWaypoint: true, TargetRefKind: "Service", TargetRefName: "api",
			}),
			wantEffective:  AuthzNoPolicy,
			wantBroadAllow: &falseValue,
			wantIdentity:   &falseValue,
			wantChain: []Step{
				authzDefaultStep(1),
				{
					Order: 2, Kind: "AuthorizationPolicy", Namespace: "payments", Name: "service-default-deny",
					Field: `spec.targetRefs["Service/api"]`, Effect: "targetRef policy attaches to a waypoint; excludes it from sidecar authorization evaluation",
				},
			},
		},
		{
			name:           "unknown data plane membership makes L4 authorization unknown",
			in:             authzWorkload(ModeUnknown, authzPolicy("payments", "allow-api", "ALLOW", true)),
			wantEffective:  AuthzUnknown,
			wantBroadAllow: nil,
			wantIdentity:   nil,
			wantUnknown:    authzDataPlaneUnknownReason,
			wantChain:      []Step{},
		},
		{
			name:           "mixed data plane membership makes L4 authorization unknown",
			in:             authzWorkload(ModeMixed, authzPolicy("payments", "allow-api", "ALLOW", true)),
			wantEffective:  AuthzUnknown,
			wantBroadAllow: nil,
			wantIdentity:   nil,
			wantUnknown:    authzDataPlaneUnknownReason,
			wantChain:      []Step{},
		},
		{
			name: "ruleless DENY never matches",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "deny-nothing", "DENY", false),
			),
			wantEffective:  AuthzNoPolicy,
			wantBroadAllow: &falseValue,
			wantIdentity:   &falseValue,
			wantPolicies:   []string{"payments/deny-nothing"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "deny-nothing", "DENY policy has no rules and never matches"),
			},
		},
		{
			name: "ruleless CUSTOM never matches",
			in: authzWorkload(ModeSidecar,
				authzPolicy("payments", "custom-nothing", "CUSTOM", false),
			),
			wantEffective:  AuthzNoPolicy,
			wantBroadAllow: &falseValue,
			wantIdentity:   &falseValue,
			wantPolicies:   []string{"payments/custom-nothing"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "custom-nothing", "CUSTOM policy has no rules and never matches"),
			},
		},
		{
			name: "operation-only ALLOW is not identity scoped",
			in: authzWorkload(ModeSidecar, AuthorizationPolicyView{
				Name: "get-any-source", Namespace: "payments", Action: "ALLOW", HasRules: true, BroadAllow: true,
			}),
			wantEffective:  AuthzAllowOnly,
			wantBroadAllow: &trueValue,
			wantIdentity:   &falseValue,
			wantPolicies:   []string{"payments/get-any-source"},
			wantChain: []Step{
				authzDefaultStep(1),
				authzPolicyStepWant(2, "payments", "get-any-source", "adds a structurally broad rule to the additive ALLOW union"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := New().ResolveAuthz(tt.in)
			if result.Effective != tt.wantEffective {
				t.Fatalf("effective = %q, want %q", result.Effective, tt.wantEffective)
			}
			if !equalOptionalBool(result.BroadAllow, tt.wantBroadAllow) {
				t.Fatalf("broadAllow = %v, want %v", optionalBoolValue(result.BroadAllow), optionalBoolValue(tt.wantBroadAllow))
			}
			if !equalOptionalBool(result.IdentityScoped, tt.wantIdentity) {
				t.Fatalf("identityScoped = %v, want %v", optionalBoolValue(result.IdentityScoped), optionalBoolValue(tt.wantIdentity))
			}
			if !reflect.DeepEqual(result.PoliciesInScope, tt.wantPolicies) {
				t.Fatalf("policiesInScope = %#v, want %#v", result.PoliciesInScope, tt.wantPolicies)
			}
			if !reflect.DeepEqual(result.WaypointUnenforced, tt.wantUnenforced) {
				t.Fatalf("waypointUnenforced = %#v, want %#v", result.WaypointUnenforced, tt.wantUnenforced)
			}
			if result.UnknownReason != tt.wantUnknown {
				t.Fatalf("unknownReason = %q, want %q", result.UnknownReason, tt.wantUnknown)
			}
			if !reflect.DeepEqual(result.Chain, tt.wantChain) {
				t.Fatalf("chain = %#v, want %#v", result.Chain, tt.wantChain)
			}
		})
	}
}

func authzWorkload(mode DataPlaneMode, policies ...AuthorizationPolicyView) WorkloadInput {
	return authzWorkloadWithWaypoint(mode, nil, policies...)
}

func authzWorkloadWithWaypoint(mode DataPlaneMode, waypoint *WaypointView, policies ...AuthorizationPolicyView) WorkloadInput {
	for i := range policies {
		if policies[i].TargetsWaypoint {
			policies[i].TargetWaypoint = waypoint
		}
	}
	return WorkloadInput{
		Ref:           workloadRef(),
		DataPlaneMode: mode,
		MeshDefaults:  MeshDefaults{RootNamespace: "istio-system", Known: true},
		AuthzPolicies: append([]AuthorizationPolicyView{}, policies...),
		Waypoint:      waypoint,
	}
}

func authzPolicy(namespace, name, action string, hasRules bool) AuthorizationPolicyView {
	return AuthorizationPolicyView{Name: name, Namespace: namespace, Action: action, HasRules: hasRules, IdentityScoped: hasRules}
}

func rootAuthzPolicy(namespace, name, action string, hasRules bool) AuthorizationPolicyView {
	policy := authzPolicy(namespace, name, action, hasRules)
	policy.RootNamespace = true
	return policy
}

func waypointAuthzPolicy(namespace, name string, hasRules bool) AuthorizationPolicyView {
	return AuthorizationPolicyView{
		Name: name, Namespace: namespace, Action: "ALLOW", HasRules: hasRules,
		TargetsWaypoint: true, RequiresL7: true, IdentityScoped: hasRules,
	}
}

func waypointL4AuthzPolicy(namespace, name string, hasRules bool) AuthorizationPolicyView {
	return AuthorizationPolicyView{
		Name: name, Namespace: namespace, Action: "ALLOW", HasRules: hasRules,
		TargetsWaypoint: true, IdentityScoped: hasRules,
	}
}

func authzDefaultStep(order int) Step {
	return Step{
		Order: order, Kind: "AuthorizationDefault", Field: "implicitEnablement",
		Effect: "establishes Istio's implicit allow when no applicable ALLOW policy exists",
	}
}

func authzPolicyStepWant(order int, namespace, name, effect string) Step {
	return Step{
		Order: order, Kind: "AuthorizationPolicy", Namespace: namespace, Name: name,
		Field: "spec.action", Effect: effect,
	}
}

func equalOptionalBool(left, right *bool) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func optionalBoolValue(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}
