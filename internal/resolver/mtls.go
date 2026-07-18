package resolver

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	mtlsVersion  = "mtls/v2"
	authzVersion = "authz/v1"

	dataPlaneUnknownReason                = "data plane membership unavailable"
	peerAuthenticationUnavailableReason   = "PeerAuthentication resources unavailable"
	ztunnelUnavailableReason              = "ztunnel availability unavailable"
	workloadPortsUnavailableReason        = "workload ports unavailable for port-level PeerAuthentication"
	destinationRulePortsUnavailableReason = "workload ports unavailable for DestinationRule port-level TLS precedence"
	rootSelectorAmbiguousReason           = "root-namespace selector PeerAuthentication semantics vary by Istio version"
	ambientDisableUnsupportedReason       = "ambient PeerAuthentication DISABLE mode is unsupported by Istio"
)

// ResolverV2 implements the current composite resolver semantics.
type ResolverV2 struct{}

// New returns the current resolver implementation.
func New() ResolverV2 {
	return ResolverV2{}
}

func (ResolverV2) Version() string {
	return mtlsVersion + "," + authzVersion
}

func (ResolverV2) ResolveMTLS(in WorkloadInput) MTLSResult {
	if !in.MeshDefaults.Known {
		return unknownMTLS(peerAuthenticationUnavailableReason)
	}

	if result, done := dataPlaneMTLS(in); done {
		return result
	}

	selection, err := selectPeerAuthentications(in)
	if err != nil {
		return unknownMTLS(err.Error())
	}

	effective := MTLSPermissive
	chain := []Step{{
		Kind:   "MeshConfigDefault",
		Field:  "defaultPeerAuthenticationMode",
		Effect: "defaults mesh mTLS mode to PERMISSIVE when no PeerAuthentication mode overrides it",
	}}

	if selection.mesh != nil {
		next, step, err := applyPeerAuthenticationMode(*selection.mesh, "mesh-wide", "mesh default", effective)
		if err != nil {
			return unknownMTLS(err.Error())
		}
		if next != "" {
			effective = next
		}
		chain = append(chain, step)
	}
	if selection.namespace != nil {
		next, step, err := applyPeerAuthenticationMode(*selection.namespace, "namespace", "mesh-wide", effective)
		if err != nil {
			return unknownMTLS(err.Error())
		}
		if next != "" {
			effective = next
		}
		chain = append(chain, step)
	}
	if selection.workload != nil {
		next, step, err := applyPeerAuthenticationMode(*selection.workload, "workload", "namespace", effective)
		if err != nil {
			return unknownMTLS(err.Error())
		}
		if next != "" {
			effective = next
		}
		chain = append(chain, step)
	}

	effective, byPort, portSteps, err := applyPortOverrides(effective, selection.workload, in.Ports)
	if err != nil {
		return unknownMTLS(err.Error())
	}
	chain = append(chain, portSteps...)
	// Ambient uses HBONE mTLS and Istio does not support DISABLE there. Until
	// ambient policy reporting is first-class in M6, do not report a sidecar-only
	// DISABLE conclusion as effective ambient posture.
	// https://istio.io/latest/docs/reference/config/security/peer_authentication/
	if in.DataPlaneMode == ModeAmbient && containsDisabledMode(effective, byPort) {
		return unknownMTLS(ambientDisableUnsupportedReason)
	}

	var contradiction *bool
	if in.DestinationRulesKnown {
		resolvedChain, resolvedContradiction, err := applyDestinationRules(chain, effective, byPort, in.Ports, in.DestRules)
		if err != nil {
			return unknownMTLS(err.Error())
		}
		chain = resolvedChain
		contradiction = &resolvedContradiction
	}

	return MTLSResult{
		Effective:              effective,
		ByPort:                 byPort,
		ClientTLSContradiction: contradiction,
		Chain:                  orderChain(chain),
	}
}

func dataPlaneMTLS(in WorkloadInput) (MTLSResult, bool) {
	switch in.DataPlaneMode {
	case ModeSidecar:
		return MTLSResult{}, false
	case ModeAmbient:
		switch in.ZtunnelOnNode {
		case True:
			return MTLSResult{}, false
		case False:
			return MTLSResult{
				Effective: MTLSNotInMesh,
				Chain: orderChain([]Step{{
					Kind:   "DataPlane",
					Field:  "ztunnelOnNode",
					Effect: "ambient workload has no available ztunnel, so mesh mTLS is not enforced",
				}}),
			}, true
		default:
			return unknownMTLS(ztunnelUnavailableReason), true
		}
	case ModeNotApplicable:
		return MTLSResult{
			Effective: MTLSNotInMesh,
			Chain: orderChain([]Step{{
				Kind:   "DataPlane",
				Field:  "dataPlaneMode",
				Effect: "workload is not enrolled in an Istio data plane, so mesh mTLS is not enforced",
			}}),
		}, true
	case ModeUnknown, ModeMixed, "":
		return unknownMTLS(dataPlaneUnknownReason), true
	default:
		return unknownMTLS(fmt.Sprintf("unsupported data plane mode %q", in.DataPlaneMode)), true
	}
}

type peerAuthenticationSelection struct {
	mesh      *PeerAuthenticationView
	namespace *PeerAuthenticationView
	workload  *PeerAuthenticationView
}

func selectPeerAuthentications(in WorkloadInput) (peerAuthenticationSelection, error) {
	// Istio documents PeerAuthentication precedence as mesh -> namespace ->
	// workload -> port, with UNSET inheriting from the parent scope. It also
	// documents oldest-policy behavior for duplicate same-scope policies:
	// https://istio.io/latest/docs/concepts/security/#peer-authentication
	rootNamespace := rootNamespace(in.MeshDefaults.RootNamespace)
	var mesh []PeerAuthenticationView
	var namespace []PeerAuthenticationView
	var workload []PeerAuthenticationView

	for _, peerAuthentication := range in.PeerAuthN {
		switch {
		case peerAuthentication.SelectorMatch:
			// Current Istio root-namespace guidance says selector policies are
			// ignored, while the generated selector field still says they match
			// across namespaces. Return unknown until the supported minor version
			// resolves that upstream conflict.
			// https://istio.io/latest/docs/reference/config/security/peer_authentication/
			if peerAuthentication.Namespace == rootNamespace {
				return peerAuthenticationSelection{}, fmt.Errorf(
					"%s for %s/%s",
					rootSelectorAmbiguousReason,
					peerAuthentication.Namespace,
					peerAuthentication.Name,
				)
			}
			workload = append(workload, peerAuthentication)
		case peerAuthentication.Namespace == rootNamespace:
			mesh = append(mesh, peerAuthentication)
		case peerAuthentication.Namespace == in.Ref.Namespace:
			namespace = append(namespace, peerAuthentication)
		}
	}

	selectedMesh, err := oldestPeerAuthentication(mesh, "mesh-wide PeerAuthentication")
	if err != nil {
		return peerAuthenticationSelection{}, err
	}
	selectedNamespace, err := oldestPeerAuthentication(namespace, "namespace PeerAuthentication")
	if err != nil {
		return peerAuthenticationSelection{}, err
	}
	selectedWorkload, err := oldestPeerAuthentication(workload, "workload PeerAuthentication")
	if err != nil {
		return peerAuthenticationSelection{}, err
	}

	return peerAuthenticationSelection{
		mesh:      selectedMesh,
		namespace: selectedNamespace,
		workload:  selectedWorkload,
	}, nil
}

func oldestPeerAuthentication(
	candidates []PeerAuthenticationView,
	scope string,
) (*PeerAuthenticationView, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	if len(candidates) == 1 {
		selected := candidates[0]
		return &selected, nil
	}

	for _, candidate := range candidates {
		if candidate.CreationTimestamp.IsZero() {
			return nil, fmt.Errorf("%s tie-break requires creationTimestamp for %s/%s", scope, candidate.Namespace, candidate.Name)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].CreationTimestamp.Equal(candidates[j].CreationTimestamp) {
			return candidates[i].CreationTimestamp.Before(candidates[j].CreationTimestamp)
		}
		if candidates[i].Namespace != candidates[j].Namespace {
			return candidates[i].Namespace < candidates[j].Namespace
		}
		return candidates[i].Name < candidates[j].Name
	})
	if candidates[0].CreationTimestamp.Equal(candidates[1].CreationTimestamp) {
		return nil, fmt.Errorf("%s tie-break has duplicate creationTimestamp %s", scope, candidates[0].CreationTimestamp.Format(time.RFC3339Nano))
	}

	selected := candidates[0]
	return &selected, nil
}

func applyPeerAuthenticationMode(
	peerAuthentication PeerAuthenticationView,
	scope string,
	parentScope string,
	parent MTLSEffective,
) (MTLSEffective, Step, error) {
	next, ok := mtlsModeToEffective(peerAuthentication.Mode)
	if !ok {
		return "", Step{}, fmt.Errorf("unsupported PeerAuthentication mode %q on %s/%s", peerAuthentication.Mode, peerAuthentication.Namespace, peerAuthentication.Name)
	}

	mode := normalizedMode(peerAuthentication.Mode)
	effect := fmt.Sprintf("sets %s mTLS mode to %s", scope, mode)
	if next == "" {
		effect = fmt.Sprintf("inherits %s mTLS mode %s", parentScope, mtlsEffectiveLabel(parent))
	}

	return next, Step{
		Kind:      "PeerAuthentication",
		Name:      peerAuthentication.Name,
		Namespace: peerAuthentication.Namespace,
		Field:     "spec.mtls.mode",
		Effect:    effect,
	}, nil
}

func applyPortOverrides(
	parent MTLSEffective,
	workloadPeerAuthentication *PeerAuthenticationView,
	ports []int32,
) (MTLSEffective, map[int32]MTLSEffective, []Step, error) {
	if ports == nil {
		if workloadPeerAuthentication == nil || len(workloadPeerAuthentication.PortLevelModes) == 0 {
			return parent, nil, nil, nil
		}
		return MTLSUnknown, nil, nil, fmt.Errorf("%s on %s/%s", workloadPortsUnavailableReason, workloadPeerAuthentication.Namespace, workloadPeerAuthentication.Name)
	}

	claimedPorts := int32Set(ports)
	orderedPorts := sortedInt32Keys(claimedPorts)
	byPort := make(map[int32]MTLSEffective, len(orderedPorts))
	for _, port := range orderedPorts {
		byPort[port] = parent
	}
	if workloadPeerAuthentication == nil || len(workloadPeerAuthentication.PortLevelModes) == 0 {
		return parent, byPort, nil, nil
	}
	for port := range workloadPeerAuthentication.PortLevelModes {
		if !claimedPorts[port] {
			return MTLSUnknown, nil, nil, fmt.Errorf("PeerAuthentication %s/%s portLevelMtls references unclaimed workload port %d", workloadPeerAuthentication.Namespace, workloadPeerAuthentication.Name, port)
		}
	}

	var steps []Step
	for _, port := range orderedPorts {
		mode, ok := workloadPeerAuthentication.PortLevelModes[port]
		if !ok {
			continue
		}
		next, valid := mtlsModeToEffective(mode)
		if !valid {
			return MTLSUnknown, nil, nil, fmt.Errorf("unsupported port-level PeerAuthentication mode %q on %s/%s port %d", mode, workloadPeerAuthentication.Namespace, workloadPeerAuthentication.Name, port)
		}

		effect := fmt.Sprintf("sets port %d mTLS mode to %s", port, normalizedMode(mode))
		if next == "" {
			next = parent
			effect = fmt.Sprintf("inherits workload mTLS mode %s for port %d", mtlsEffectiveLabel(parent), port)
		}
		byPort[port] = next
		steps = append(steps, Step{
			Kind:      "PeerAuthentication",
			Name:      workloadPeerAuthentication.Name,
			Namespace: workloadPeerAuthentication.Namespace,
			Field:     fmt.Sprintf("spec.portLevelMtls[%q].mode", fmt.Sprint(port)),
			Effect:    effect,
		})
	}

	effective := commonPortEffective(byPort)
	return effective, byPort, steps, nil
}

func commonPortEffective(byPort map[int32]MTLSEffective) MTLSEffective {
	var first MTLSEffective
	for _, port := range sortedInt32Keys(byPort) {
		mode := byPort[port]
		if first == "" {
			first = mode
			continue
		}
		if mode != first {
			return MTLSMixedByPort
		}
	}
	return first
}

func containsDisabledMode(effective MTLSEffective, byPort map[int32]MTLSEffective) bool {
	if effective == MTLSDisabled {
		return true
	}
	for _, mode := range byPort {
		if mode == MTLSDisabled {
			return true
		}
	}
	return false
}

func applyDestinationRules(
	chain []Step,
	effective MTLSEffective,
	byPort map[int32]MTLSEffective,
	ports []int32,
	destinationRules []DestinationRuleView,
) ([]Step, bool, error) {
	// Auto mTLS only applies when DestinationRule TLS is not explicitly set;
	// explicit DISABLE/SIMPLE modes can conflict with strict server mTLS:
	// https://istio.io/latest/docs/ops/configuration/traffic-management/tls-configuration/#auto-mtls
	// https://istio.io/latest/docs/ops/common-problems/network-issues/#503-errors-after-setting-destination-rule
	// Port-level settings replace, rather than inherit, destination-level fields:
	// https://istio.io/latest/docs/reference/config/networking/destination-rule/#TrafficPolicy
	contradiction := false
	for _, destinationRule := range sortedDestinationRules(destinationRules) {
		if strings.TrimSpace(destinationRule.TLSMode) != "" {
			serverStrict, applies, err := destinationLevelServerStrict(
				effective,
				byPort,
				ports,
				destinationRule.PortTLSModes,
			)
			if err != nil {
				return nil, false, err
			}
			conflicts, step, err := destinationRuleStep(
				destinationRule,
				"spec.trafficPolicy.tls.mode",
				0,
				destinationRule.TLSMode,
				serverStrict,
			)
			if err != nil {
				return nil, false, err
			}
			if !applies {
				step.Effect = fmt.Sprintf(
					"sets client TLS mode %s at destination level, overridden for all workload ports",
					normalizedMode(destinationRule.TLSMode),
				)
			}
			contradiction = contradiction || conflicts
			chain = append(chain, step)
		}
		for _, port := range sortedInt32Keys(destinationRule.PortTLSModes) {
			mode := destinationRule.PortTLSModes[port]
			conflicts, step, err := destinationRuleStep(
				destinationRule,
				fmt.Sprintf("spec.trafficPolicy.portLevelSettings[%q].tls.mode", fmt.Sprint(port)),
				port,
				mode,
				serverStrictForPort(effective, byPort, port),
			)
			if err != nil {
				return nil, false, err
			}
			contradiction = contradiction || conflicts
			chain = append(chain, step)
		}
	}
	return chain, contradiction, nil
}

func destinationRuleStep(
	destinationRule DestinationRuleView,
	field string,
	port int32,
	mode string,
	serverStrict bool,
) (bool, Step, error) {
	normalized := normalizedMode(mode)
	switch normalized {
	case "":
	case "DISABLE", "SIMPLE":
	case "MUTUAL", "ISTIO_MUTUAL":
	default:
		return false, Step{}, fmt.Errorf("unsupported DestinationRule TLS mode %q on %s/%s", mode, destinationRule.Namespace, destinationRule.Name)
	}

	conflicts := serverStrict && (normalized == "DISABLE" || normalized == "SIMPLE")
	var effect string
	if normalized == "" {
		effect = "leaves client TLS mode unset, so automatic mTLS applies"
		if port != 0 {
			effect = fmt.Sprintf("leaves client TLS mode unset for port %d, so automatic mTLS applies", port)
		}
	} else {
		effect = fmt.Sprintf("sets client TLS mode %s", normalized)
		if port != 0 {
			effect = fmt.Sprintf("sets client TLS mode %s for port %d", normalized, port)
		}
	}
	if conflicts {
		effect += ", which conflicts with strict server mTLS"
	} else if serverStrict {
		effect += ", compatible with strict server mTLS"
	}

	return conflicts, Step{
		Kind:      "DestinationRule",
		Name:      destinationRule.Name,
		Namespace: destinationRule.Namespace,
		Field:     field,
		Effect:    effect,
	}, nil
}

func destinationLevelServerStrict(
	effective MTLSEffective,
	byPort map[int32]MTLSEffective,
	ports []int32,
	portTLSModes map[int32]string,
) (serverStrict bool, applies bool, err error) {
	if len(portTLSModes) == 0 {
		return serverStrictForAnyPort(effective, byPort), true, nil
	}
	if len(ports) == 0 {
		return false, false, fmt.Errorf("%s", destinationRulePortsUnavailableReason)
	}

	for _, port := range sortedInt32Keys(int32Set(ports)) {
		if _, overridden := portTLSModes[port]; overridden {
			continue
		}
		applies = true
		serverStrict = serverStrict || serverStrictForPort(effective, byPort, port)
	}
	return serverStrict, applies, nil
}

func serverStrictForAnyPort(effective MTLSEffective, byPort map[int32]MTLSEffective) bool {
	if len(byPort) == 0 {
		return effective == MTLSStrict
	}
	for _, mode := range byPort {
		if mode == MTLSStrict {
			return true
		}
	}
	return false
}

func serverStrictForPort(effective MTLSEffective, byPort map[int32]MTLSEffective, port int32) bool {
	if len(byPort) == 0 {
		return effective == MTLSStrict
	}
	if mode, ok := byPort[port]; ok {
		return mode == MTLSStrict
	}
	return effective == MTLSStrict
}

func sortedDestinationRules(destinationRules []DestinationRuleView) []DestinationRuleView {
	out := append([]DestinationRuleView(nil), destinationRules...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Host < out[j].Host
	})
	return out
}

func mtlsModeToEffective(mode string) (MTLSEffective, bool) {
	switch normalizedMode(mode) {
	case "", "UNSET":
		return "", true
	case "STRICT":
		return MTLSStrict, true
	case "PERMISSIVE":
		return MTLSPermissive, true
	case "DISABLE":
		return MTLSDisabled, true
	default:
		return MTLSUnknown, false
	}
}

func normalizedMode(mode string) string {
	return strings.ToUpper(strings.TrimSpace(mode))
}

func mtlsEffectiveLabel(effective MTLSEffective) string {
	switch effective {
	case MTLSDisabled:
		return "DISABLE"
	case MTLSPermissive:
		return "PERMISSIVE"
	case MTLSStrict:
		return "STRICT"
	default:
		return strings.ToUpper(string(effective))
	}
}

func rootNamespace(namespace string) string {
	if strings.TrimSpace(namespace) == "" {
		return "istio-system"
	}
	return namespace
}

func int32Set(values []int32) map[int32]bool {
	out := make(map[int32]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func sortedInt32Keys[V any](values map[int32]V) []int32 {
	out := make([]int32, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func orderChain(chain []Step) []Step {
	out := append([]Step(nil), chain...)
	for i := range out {
		out[i].Order = i + 1
	}
	return out
}

func unknownMTLS(reason string) MTLSResult {
	return MTLSResult{
		Effective:     MTLSUnknown,
		Chain:         []Step{},
		UnknownReason: reason,
	}
}
