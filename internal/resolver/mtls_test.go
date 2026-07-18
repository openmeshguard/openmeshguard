package resolver

import (
	"reflect"
	"testing"
	"time"
)

func TestResolverV2Version(t *testing.T) {
	if got := New().Version(); got != "mtls/v2,authz/v1" {
		t.Fatalf("Version() = %q, want mtls/v2,authz/v1", got)
	}
}

func TestResolverV2ResolveMTLS(t *testing.T) {
	tests := []struct {
		name                    string
		in                      WorkloadInput
		wantEffective           MTLSEffective
		wantByPort              map[int32]MTLSEffective
		wantClientContradiction bool
		wantUnknownReason       string
		wantChain               []Step
	}{
		{
			name:          "no PeerAuthentication uses mesh permissive default",
			in:            sidecarWorkload(),
			wantEffective: MTLSPermissive,
			wantChain: []Step{
				defaultStep(1),
			},
		},
		{
			name: "namespace PeerAuthentication overrides mesh-wide PeerAuthentication",
			in: sidecarWorkload(
				peerAuthentication("istio-system", "default", "STRICT", false),
				peerAuthentication("payments", "default", "PERMISSIVE", false),
			),
			wantEffective: MTLSPermissive,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "istio-system", "default", "spec.mtls.mode", "sets mesh-wide mTLS mode to STRICT"),
				peerStep(3, "payments", "default", "spec.mtls.mode", "sets namespace mTLS mode to PERMISSIVE"),
			},
		},
		{
			name: "workload PeerAuthentication overrides namespace PeerAuthentication",
			in: sidecarWorkload(
				peerAuthentication("payments", "default", "STRICT", false),
				peerAuthentication("payments", "api", "DISABLE", true),
			),
			wantEffective: MTLSDisabled,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "payments", "default", "spec.mtls.mode", "sets namespace mTLS mode to STRICT"),
				peerStep(3, "payments", "api", "spec.mtls.mode", "sets workload mTLS mode to DISABLE"),
			},
		},
		{
			name: "UNSET inherits through namespace and workload scopes",
			in: sidecarWorkload(
				peerAuthentication("istio-system", "default", "STRICT", false),
				peerAuthentication("payments", "default", "UNSET", false),
				peerAuthentication("payments", "api", "UNSET", true),
			),
			wantEffective: MTLSStrict,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "istio-system", "default", "spec.mtls.mode", "sets mesh-wide mTLS mode to STRICT"),
				peerStep(3, "payments", "default", "spec.mtls.mode", "inherits mesh-wide mTLS mode STRICT"),
				peerStep(4, "payments", "api", "spec.mtls.mode", "inherits namespace mTLS mode STRICT"),
			},
		},
		{
			name: "port-level DISABLE overrides workload STRICT",
			in: sidecarWorkload(
				peerAuthenticationWithPorts("payments", "api", "STRICT", true, map[int32]string{8080: "DISABLE"}),
			),
			wantEffective: MTLSMixedByPort,
			wantByPort: map[int32]MTLSEffective{
				8080: MTLSDisabled,
				9090: MTLSStrict,
			},
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "payments", "api", "spec.mtls.mode", "sets workload mTLS mode to STRICT"),
				peerStep(3, "payments", "api", `spec.portLevelMtls["8080"].mode`, "sets port 8080 mTLS mode to DISABLE"),
			},
		},
		{
			name: "port-level STRICT overrides disabled parent",
			in: sidecarWorkload(
				peerAuthentication("payments", "default", "DISABLE", false),
				peerAuthenticationWithPorts("payments", "api", "UNSET", true, map[int32]string{8080: "STRICT"}),
			),
			wantEffective: MTLSMixedByPort,
			wantByPort: map[int32]MTLSEffective{
				8080: MTLSStrict,
				9090: MTLSDisabled,
			},
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "payments", "default", "spec.mtls.mode", "sets namespace mTLS mode to DISABLE"),
				peerStep(3, "payments", "api", "spec.mtls.mode", "inherits namespace mTLS mode DISABLE"),
				peerStep(4, "payments", "api", `spec.portLevelMtls["8080"].mode`, "sets port 8080 mTLS mode to STRICT"),
			},
		},
		{
			name: "DestinationRule DISABLE contradicts strict server mTLS",
			in: sidecarWorkloadWithDestinationRules(
				[]PeerAuthenticationView{peerAuthentication("istio-system", "default", "STRICT", false)},
				[]DestinationRuleView{{Name: "api", Namespace: "payments", Host: "api.payments.svc.cluster.local", TLSMode: "DISABLE"}},
			),
			wantEffective:           MTLSStrict,
			wantClientContradiction: true,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "istio-system", "default", "spec.mtls.mode", "sets mesh-wide mTLS mode to STRICT"),
				destinationRuleStepForTest(3, "payments", "api", "spec.trafficPolicy.tls.mode", "sets client TLS mode DISABLE, which conflicts with strict server mTLS"),
			},
		},
		{
			name: "DestinationRule SIMPLE contradicts strict server mTLS",
			in: sidecarWorkloadWithDestinationRules(
				[]PeerAuthenticationView{peerAuthentication("istio-system", "default", "STRICT", false)},
				[]DestinationRuleView{{Name: "api", Namespace: "payments", Host: "api.payments.svc.cluster.local", TLSMode: "SIMPLE"}},
			),
			wantEffective:           MTLSStrict,
			wantClientContradiction: true,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "istio-system", "default", "spec.mtls.mode", "sets mesh-wide mTLS mode to STRICT"),
				destinationRuleStepForTest(3, "payments", "api", "spec.trafficPolicy.tls.mode", "sets client TLS mode SIMPLE, which conflicts with strict server mTLS"),
			},
		},
		{
			name: "DestinationRule ISTIO_MUTUAL does not contradict strict server mTLS",
			in: sidecarWorkloadWithDestinationRules(
				[]PeerAuthenticationView{peerAuthentication("istio-system", "default", "STRICT", false)},
				[]DestinationRuleView{{Name: "api", Namespace: "payments", Host: "api.payments.svc.cluster.local", TLSMode: "ISTIO_MUTUAL"}},
			),
			wantEffective: MTLSStrict,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "istio-system", "default", "spec.mtls.mode", "sets mesh-wide mTLS mode to STRICT"),
				destinationRuleStepForTest(3, "payments", "api", "spec.trafficPolicy.tls.mode", "sets client TLS mode ISTIO_MUTUAL, compatible with strict server mTLS"),
			},
		},
		{
			name: "DestinationRule port TLS overrides destination-level contradiction",
			in: sidecarWorkloadWithPortsAndDestinationRules(
				[]int32{8080},
				[]PeerAuthenticationView{peerAuthentication("istio-system", "default", "STRICT", false)},
				[]DestinationRuleView{{
					Name:         "api",
					Namespace:    "payments",
					Host:         "api.payments.svc.cluster.local",
					TLSMode:      "DISABLE",
					PortTLSModes: map[int32]string{8080: "ISTIO_MUTUAL"},
				}},
			),
			wantEffective: MTLSStrict,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "istio-system", "default", "spec.mtls.mode", "sets mesh-wide mTLS mode to STRICT"),
				destinationRuleStepForTest(3, "payments", "api", "spec.trafficPolicy.tls.mode", "sets client TLS mode DISABLE at destination level, overridden for all workload ports"),
				destinationRuleStepForTest(4, "payments", "api", `spec.trafficPolicy.portLevelSettings["8080"].tls.mode`, "sets client TLS mode ISTIO_MUTUAL for port 8080, compatible with strict server mTLS"),
			},
		},
		{
			name: "DestinationRule destination TLS still contradicts an unoverridden strict port",
			in: sidecarWorkloadWithPortsAndDestinationRules(
				[]int32{8080, 9090},
				[]PeerAuthenticationView{peerAuthentication("istio-system", "default", "STRICT", false)},
				[]DestinationRuleView{{
					Name:         "api",
					Namespace:    "payments",
					Host:         "api.payments.svc.cluster.local",
					TLSMode:      "DISABLE",
					PortTLSModes: map[int32]string{8080: "ISTIO_MUTUAL"},
				}},
			),
			wantEffective:           MTLSStrict,
			wantClientContradiction: true,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "istio-system", "default", "spec.mtls.mode", "sets mesh-wide mTLS mode to STRICT"),
				destinationRuleStepForTest(3, "payments", "api", "spec.trafficPolicy.tls.mode", "sets client TLS mode DISABLE, which conflicts with strict server mTLS"),
				destinationRuleStepForTest(4, "payments", "api", `spec.trafficPolicy.portLevelSettings["8080"].tls.mode`, "sets client TLS mode ISTIO_MUTUAL for port 8080, compatible with strict server mTLS"),
			},
		},
		{
			name: "DestinationRule port policy without TLS does not inherit destination TLS",
			in: sidecarWorkloadWithPortsAndDestinationRules(
				[]int32{8080},
				[]PeerAuthenticationView{peerAuthentication("istio-system", "default", "STRICT", false)},
				[]DestinationRuleView{{
					Name:         "api",
					Namespace:    "payments",
					Host:         "api.payments.svc.cluster.local",
					TLSMode:      "DISABLE",
					PortTLSModes: map[int32]string{8080: ""},
				}},
			),
			wantEffective: MTLSStrict,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "istio-system", "default", "spec.mtls.mode", "sets mesh-wide mTLS mode to STRICT"),
				destinationRuleStepForTest(3, "payments", "api", "spec.trafficPolicy.tls.mode", "sets client TLS mode DISABLE at destination level, overridden for all workload ports"),
				destinationRuleStepForTest(4, "payments", "api", `spec.trafficPolicy.portLevelSettings["8080"].tls.mode`, "leaves client TLS mode unset for port 8080, so automatic mTLS applies, compatible with strict server mTLS"),
			},
		},
		{
			name: "DestinationRule port precedence without workload ports is unknown",
			in: sidecarWorkloadWithPortsAndDestinationRules(
				nil,
				[]PeerAuthenticationView{peerAuthentication("istio-system", "default", "STRICT", false)},
				[]DestinationRuleView{{
					Name:         "api",
					Namespace:    "payments",
					Host:         "api.payments.svc.cluster.local",
					TLSMode:      "DISABLE",
					PortTLSModes: map[int32]string{8080: "ISTIO_MUTUAL"},
				}},
			),
			wantEffective:     MTLSUnknown,
			wantUnknownReason: destinationRulePortsUnavailableReason,
			wantChain:         []Step{},
		},
		{
			name: "not-in-mesh workload resolves not-in-mesh",
			in: WorkloadInput{
				Ref:           workloadRef(),
				DataPlaneMode: ModeNotApplicable,
				MeshDefaults:  knownMeshDefaults(),
			},
			wantEffective: MTLSNotInMesh,
			wantChain: []Step{{
				Order:  1,
				Kind:   "DataPlane",
				Field:  "dataPlaneMode",
				Effect: "workload is not enrolled in an Istio data plane, so mesh mTLS is not enforced",
			}},
		},
		{
			name: "unknown mesh config propagates unknown",
			in: WorkloadInput{
				Ref:           workloadRef(),
				DataPlaneMode: ModeSidecar,
				MeshDefaults:  MeshDefaults{RootNamespace: "istio-system", Known: false},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: peerAuthenticationUnavailableReason,
			wantChain:         []Step{},
		},
		{
			name: "ambient workload with unobserved ztunnel propagates unknown",
			in: WorkloadInput{
				Ref:           workloadRef(),
				DataPlaneMode: ModeAmbient,
				MeshDefaults:  knownMeshDefaults(),
				ZtunnelOnNode: Unobserved,
				PeerAuthN:     []PeerAuthenticationView{peerAuthentication("istio-system", "default", "STRICT", false)},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: ztunnelUnavailableReason,
			wantChain:         []Step{},
		},
		{
			name: "ambient workload with DISABLE policy is unknown",
			in: WorkloadInput{
				Ref:           workloadRef(),
				DataPlaneMode: ModeAmbient,
				MeshDefaults:  knownMeshDefaults(),
				ZtunnelOnNode: True,
				PeerAuthN:     []PeerAuthenticationView{peerAuthentication("payments", "default", "DISABLE", false)},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: ambientDisableUnsupportedReason,
			wantChain:         []Step{},
		},
		{
			name: "ambient workload with port-level DISABLE policy is unknown",
			in: WorkloadInput{
				Ref:           workloadRef(),
				Ports:         []int32{8080, 9090},
				DataPlaneMode: ModeAmbient,
				MeshDefaults:  knownMeshDefaults(),
				ZtunnelOnNode: True,
				PeerAuthN: []PeerAuthenticationView{
					peerAuthenticationWithPorts("payments", "api", "STRICT", true, map[int32]string{8080: "DISABLE"}),
				},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: ambientDisableUnsupportedReason,
			wantChain:         []Step{},
		},
		{
			name: "root namespace selector PeerAuthentication is unknown across Istio versions",
			in: sidecarWorkload(
				peerAuthentication("istio-system", "api", "STRICT", true),
			),
			wantEffective:     MTLSUnknown,
			wantUnknownReason: rootSelectorAmbiguousReason + " for istio-system/api",
			wantChain:         []Step{},
		},
		{
			name: "multiple selector PeerAuthentications pick the oldest match",
			in: sidecarWorkload(
				peerAuthentication("payments", "default", "STRICT", false),
				peerAuthenticationCreated("payments", "newer-api", "PERMISSIVE", true, timestamp(2)),
				peerAuthenticationCreated("payments", "older-api", "DISABLE", true, timestamp(1)),
			),
			wantEffective: MTLSDisabled,
			wantChain: []Step{
				defaultStep(1),
				peerStep(2, "payments", "default", "spec.mtls.mode", "sets namespace mTLS mode to STRICT"),
				peerStep(3, "payments", "older-api", "spec.mtls.mode", "sets workload mTLS mode to DISABLE"),
			},
		},
		{
			name: "multiple selector PeerAuthentications without timestamps are unknown",
			in: sidecarWorkload(
				peerAuthentication("payments", "default", "STRICT", false),
				peerAuthentication("payments", "api-a", "DISABLE", true),
				peerAuthentication("payments", "api-b", "PERMISSIVE", true),
			),
			wantEffective:     MTLSUnknown,
			wantUnknownReason: "workload PeerAuthentication tie-break requires creationTimestamp for payments/api-a",
			wantChain:         []Step{},
		},
	}

	resolver := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ResolveMTLS(tt.in)
			if result.Effective != tt.wantEffective {
				t.Fatalf("effective = %q, want %q", result.Effective, tt.wantEffective)
			}
			if !reflect.DeepEqual(result.ByPort, tt.wantByPort) {
				t.Fatalf("byPort = %#v, want %#v", result.ByPort, tt.wantByPort)
			}
			wantContradictionKnown := tt.in.DestinationRulesKnown && tt.wantEffective != MTLSUnknown && tt.wantEffective != MTLSNotInMesh
			if gotKnown := result.ClientTLSContradiction != nil; gotKnown != wantContradictionKnown {
				t.Fatalf("clientTLSContradiction known = %t, want %t", gotKnown, wantContradictionKnown)
			}
			if result.ClientTLSContradiction != nil && *result.ClientTLSContradiction != tt.wantClientContradiction {
				t.Fatalf("clientTLSContradiction = %t, want %t", *result.ClientTLSContradiction, tt.wantClientContradiction)
			}
			if result.UnknownReason != tt.wantUnknownReason {
				t.Fatalf("unknownReason = %q, want %q", result.UnknownReason, tt.wantUnknownReason)
			}
			if !reflect.DeepEqual(result.Chain, tt.wantChain) {
				t.Fatalf("chain = %#v, want %#v", result.Chain, tt.wantChain)
			}
		})
	}
}

func TestResolverV2OmitsClientTLSConclusionWhenDestinationRulesUnavailable(t *testing.T) {
	in := sidecarWorkload(peerAuthentication("istio-system", "default", "STRICT", false))
	in.DestinationRulesKnown = false
	in.DestRules = []DestinationRuleView{{
		Name: "untrusted-input", Namespace: "payments", TLSMode: "DISABLE",
	}}
	result := New().ResolveMTLS(in)
	if result.Effective != MTLSStrict {
		t.Fatalf("effective = %q, want strict server posture", result.Effective)
	}
	if result.ClientTLSContradiction != nil {
		t.Fatalf("clientTLSContradiction = %v, want omitted without DestinationRule evidence", *result.ClientTLSContradiction)
	}
	for _, step := range result.Chain {
		if step.Kind == "DestinationRule" {
			t.Fatalf("resolution chain used unavailable DestinationRule evidence: %#v", result.Chain)
		}
	}
}

func sidecarWorkload(peerAuthentications ...PeerAuthenticationView) WorkloadInput {
	return WorkloadInput{
		Ref:                   workloadRef(),
		Ports:                 []int32{8080, 9090},
		DataPlaneMode:         ModeSidecar,
		MeshDefaults:          knownMeshDefaults(),
		PeerAuthN:             peerAuthentications,
		DestinationRulesKnown: true,
	}
}

func sidecarWorkloadWithDestinationRules(
	peerAuthentications []PeerAuthenticationView,
	destinationRules []DestinationRuleView,
) WorkloadInput {
	return sidecarWorkloadWithPortsAndDestinationRules(
		[]int32{8080, 9090},
		peerAuthentications,
		destinationRules,
	)
}

func sidecarWorkloadWithPortsAndDestinationRules(
	ports []int32,
	peerAuthentications []PeerAuthenticationView,
	destinationRules []DestinationRuleView,
) WorkloadInput {
	in := sidecarWorkload(peerAuthentications...)
	in.Ports = ports
	in.DestRules = destinationRules
	return in
}

func workloadRef() WorkloadRef {
	return WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"}
}

func knownMeshDefaults() MeshDefaults {
	return MeshDefaults{RootNamespace: "istio-system", Known: true}
}

func peerAuthentication(namespace, name, mode string, selectorMatch bool) PeerAuthenticationView {
	return peerAuthenticationCreated(namespace, name, mode, selectorMatch, time.Time{})
}

func peerAuthenticationCreated(
	namespace string,
	name string,
	mode string,
	selectorMatch bool,
	creationTimestamp time.Time,
) PeerAuthenticationView {
	return PeerAuthenticationView{
		Name:              name,
		Namespace:         namespace,
		SelectorMatch:     selectorMatch,
		CreationTimestamp: creationTimestamp,
		Mode:              mode,
	}
}

func peerAuthenticationWithPorts(
	namespace string,
	name string,
	mode string,
	selectorMatch bool,
	ports map[int32]string,
) PeerAuthenticationView {
	view := peerAuthentication(namespace, name, mode, selectorMatch)
	view.PortLevelModes = ports
	return view
}

func timestamp(offset int) time.Time {
	return time.Date(2026, time.January, 1, 0, 0, offset, 0, time.UTC)
}

func defaultStep(order int) Step {
	return Step{
		Order:  order,
		Kind:   "MeshConfigDefault",
		Field:  "defaultPeerAuthenticationMode",
		Effect: "defaults mesh mTLS mode to PERMISSIVE when no PeerAuthentication mode overrides it",
	}
}

func peerStep(order int, namespace, name, field, effect string) Step {
	return Step{
		Order:     order,
		Kind:      "PeerAuthentication",
		Name:      name,
		Namespace: namespace,
		Field:     field,
		Effect:    effect,
	}
}

func destinationRuleStepForTest(order int, namespace, name, field, effect string) Step {
	return Step{
		Order:     order,
		Kind:      "DestinationRule",
		Name:      name,
		Namespace: namespace,
		Field:     field,
		Effect:    effect,
	}
}
