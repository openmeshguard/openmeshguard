package resolver

import "testing"

func TestProvisionalResolverResolveMTLS(t *testing.T) {
	tests := []struct {
		name              string
		in                WorkloadInput
		wantEffective     MTLSEffective
		wantUnknownReason string
		wantChainKinds    []string
	}{
		{
			name: "namespace PeerAuthentication overrides mesh-wide PeerAuthentication",
			in: WorkloadInput{
				Ref: WorkloadRef{
					Namespace: "payments",
					Name:      "api",
					Kind:      "Deployment",
				},
				DataPlaneMode: ModeSidecar,
				MeshDefaults: MeshDefaults{
					RootNamespace: "istio-system",
					Known:         true,
				},
				PeerAuthN: []PeerAuthenticationView{
					{Name: "default", Namespace: "istio-system", Mode: "STRICT"},
					{Name: "default", Namespace: "payments", Mode: "PERMISSIVE"},
				},
			},
			wantEffective:  MTLSPermissive,
			wantChainKinds: []string{"MeshConfigDefault", "PeerAuthentication", "PeerAuthentication"},
		},
		{
			name: "defaults to permissive with chain",
			in: WorkloadInput{
				Ref:           WorkloadRef{Namespace: "default", Name: "api", Kind: "Deployment"},
				DataPlaneMode: ModeSidecar,
				MeshDefaults: MeshDefaults{
					RootNamespace: "istio-system",
					Known:         true,
				},
			},
			wantEffective:  MTLSPermissive,
			wantChainKinds: []string{"MeshConfigDefault"},
		},
		{
			name: "unknown data plane makes mTLS unknown",
			in: WorkloadInput{
				Ref:           WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				DataPlaneMode: ModeUnknown,
				MeshDefaults:  MeshDefaults{RootNamespace: "istio-system", Known: true},
				PeerAuthN:     []PeerAuthenticationView{{Name: "default", Namespace: "istio-system", Mode: "STRICT"}},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: dataPlaneUnknownReason,
		},
		{
			name: "mixed data plane makes mTLS unknown",
			in: WorkloadInput{
				Ref:           WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				DataPlaneMode: ModeMixed,
				MeshDefaults:  MeshDefaults{RootNamespace: "istio-system", Known: true},
				PeerAuthN:     []PeerAuthenticationView{{Name: "default", Namespace: "istio-system", Mode: "STRICT"}},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: dataPlaneUnknownReason,
		},
		{
			name: "selector PeerAuthentication is M2",
			in: WorkloadInput{
				DataPlaneMode: ModeSidecar,
				MeshDefaults:  MeshDefaults{Known: true},
				PeerAuthN:     []PeerAuthenticationView{{Name: "api", Namespace: "payments", Mode: "STRICT", SelectorMatch: true}},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: notImplementedM2Reason,
		},
		{
			name: "port level PeerAuthentication is M2",
			in: WorkloadInput{
				DataPlaneMode: ModeSidecar,
				MeshDefaults:  MeshDefaults{Known: true},
				PeerAuthN:     []PeerAuthenticationView{{Name: "api", Namespace: "payments", Mode: "STRICT", PortLevelModes: map[int32]string{8080: "DISABLE"}}},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: notImplementedM2Reason,
		},
		{
			name: "DestinationRule interplay is M2",
			in: WorkloadInput{
				DataPlaneMode: ModeSidecar,
				MeshDefaults:  MeshDefaults{Known: true},
				DestRules:     []DestinationRuleView{{Name: "api", Namespace: "payments", Host: "api.payments.svc.cluster.local"}},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: notImplementedM2Reason,
		},
		{
			name: "multiple mesh-wide PeerAuthentications are M2",
			in: WorkloadInput{
				Ref:           WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				DataPlaneMode: ModeSidecar,
				MeshDefaults: MeshDefaults{
					RootNamespace: "istio-system",
					Known:         true,
				},
				PeerAuthN: []PeerAuthenticationView{
					{Name: "a", Namespace: "istio-system", Mode: "STRICT"},
					{Name: "b", Namespace: "istio-system", Mode: "PERMISSIVE"},
				},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: notImplementedM2Reason,
		},
		{
			name: "multiple namespace PeerAuthentications are M2",
			in: WorkloadInput{
				Ref:           WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				DataPlaneMode: ModeSidecar,
				MeshDefaults:  MeshDefaults{RootNamespace: "istio-system", Known: true},
				PeerAuthN: []PeerAuthenticationView{
					{Name: "a", Namespace: "payments", Mode: "STRICT"},
					{Name: "b", Namespace: "payments", Mode: "PERMISSIVE"},
				},
			},
			wantEffective:     MTLSUnknown,
			wantUnknownReason: notImplementedM2Reason,
		},
	}

	resolver := NewProvisional()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ResolveMTLS(tt.in)
			if result.Effective != tt.wantEffective {
				t.Fatalf("effective = %q, want %q", result.Effective, tt.wantEffective)
			}
			if result.UnknownReason != tt.wantUnknownReason {
				t.Fatalf("unknown reason = %q, want %q", result.UnknownReason, tt.wantUnknownReason)
			}
			if len(tt.wantChainKinds) > 0 {
				if len(result.Chain) != len(tt.wantChainKinds) {
					t.Fatalf("chain length = %d, want %d: %#v", len(result.Chain), len(tt.wantChainKinds), result.Chain)
				}
				for i, want := range tt.wantChainKinds {
					if result.Chain[i].Kind != want {
						t.Fatalf("chain[%d].Kind = %q, want %q: %#v", i, result.Chain[i].Kind, want, result.Chain)
					}
				}
			}
		})
	}
}

func TestProvisionalResolverResolveAuthz(t *testing.T) {
	tests := []struct {
		name              string
		in                WorkloadInput
		wantEffective     AuthzEffective
		wantUnknownReason string
	}{
		{
			name:              "authz is M2",
			in:                WorkloadInput{},
			wantEffective:     AuthzUnknown,
			wantUnknownReason: notImplementedM2Reason,
		},
	}

	resolver := NewProvisional()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ResolveAuthz(tt.in)
			if result.Effective != tt.wantEffective {
				t.Fatalf("effective = %q, want %q", result.Effective, tt.wantEffective)
			}
			if result.UnknownReason != tt.wantUnknownReason {
				t.Fatalf("unknown reason = %q, want %q", result.UnknownReason, tt.wantUnknownReason)
			}
		})
	}
}
