package resolver

import "testing"

func TestProvisionalResolverMeshAndNamespacePeerAuthenticationPrecedence(t *testing.T) {
	resolver := NewProvisional()

	result := resolver.ResolveMTLS(WorkloadInput{
		Ref: WorkloadRef{
			Namespace: "payments",
			Name:      "api",
			Kind:      "Deployment",
		},
		MeshDefaults: MeshDefaults{
			RootNamespace: "istio-system",
			Known:         true,
		},
		PeerAuthN: []PeerAuthenticationView{
			{Name: "default", Namespace: "istio-system", Mode: "STRICT"},
			{Name: "default", Namespace: "payments", Mode: "PERMISSIVE"},
		},
	})

	if result.Effective != MTLSPermissive {
		t.Fatalf("effective = %q, want %q", result.Effective, MTLSPermissive)
	}
	if result.UnknownReason != "" {
		t.Fatalf("unknown reason = %q, want empty", result.UnknownReason)
	}
	if len(result.Chain) != 3 {
		t.Fatalf("chain length = %d, want 3: %#v", len(result.Chain), result.Chain)
	}
	if result.Chain[1].Namespace != "istio-system" || result.Chain[2].Namespace != "payments" {
		t.Fatalf("chain did not record mesh then namespace precedence: %#v", result.Chain)
	}
}

func TestProvisionalResolverDefaultsToPermissiveWithChain(t *testing.T) {
	result := NewProvisional().ResolveMTLS(WorkloadInput{
		Ref: WorkloadRef{Namespace: "default", Name: "api", Kind: "Deployment"},
		MeshDefaults: MeshDefaults{
			RootNamespace: "istio-system",
			Known:         true,
		},
	})

	if result.Effective != MTLSPermissive {
		t.Fatalf("effective = %q, want %q", result.Effective, MTLSPermissive)
	}
	if len(result.Chain) != 1 || result.Chain[0].Kind != "MeshConfigDefault" {
		t.Fatalf("default result chain = %#v, want MeshConfigDefault", result.Chain)
	}
}

func TestProvisionalResolverM2InputsReturnUnknown(t *testing.T) {
	tests := []struct {
		name string
		in   WorkloadInput
	}{
		{
			name: "selector PeerAuthentication",
			in: WorkloadInput{
				MeshDefaults: MeshDefaults{Known: true},
				PeerAuthN:    []PeerAuthenticationView{{Name: "api", Namespace: "payments", Mode: "STRICT", SelectorMatch: true}},
			},
		},
		{
			name: "port level PeerAuthentication",
			in: WorkloadInput{
				MeshDefaults: MeshDefaults{Known: true},
				PeerAuthN:    []PeerAuthenticationView{{Name: "api", Namespace: "payments", Mode: "STRICT", PortLevelModes: map[int32]string{8080: "DISABLE"}}},
			},
		},
		{
			name: "DestinationRule interplay",
			in: WorkloadInput{
				MeshDefaults: MeshDefaults{Known: true},
				DestRules:    []DestinationRuleView{{Name: "api", Namespace: "payments", Host: "api.payments.svc.cluster.local"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewProvisional().ResolveMTLS(tt.in)
			if result.Effective != MTLSUnknown {
				t.Fatalf("effective = %q, want unknown", result.Effective)
			}
			if result.UnknownReason != notImplementedM2Reason {
				t.Fatalf("unknown reason = %q, want %q", result.UnknownReason, notImplementedM2Reason)
			}
		})
	}
}

func TestProvisionalResolverAuthzUnknown(t *testing.T) {
	result := NewProvisional().ResolveAuthz(WorkloadInput{})
	if result.Effective != AuthzUnknown {
		t.Fatalf("effective = %q, want %q", result.Effective, AuthzUnknown)
	}
	if result.UnknownReason != notImplementedM2Reason {
		t.Fatalf("unknown reason = %q, want %q", result.UnknownReason, notImplementedM2Reason)
	}
}
