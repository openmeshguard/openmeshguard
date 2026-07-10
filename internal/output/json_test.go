package output

import (
	"strings"
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

func TestProvisionalFindings(t *testing.T) {
	tests := []struct {
		name              string
		workload          resolver.WorkloadResult
		wantFindings      int
		wantStatus        string
		wantConfidence    string
		wantUnknownReason string
	}{
		{
			name: "permissive mTLS with unknown data plane emits unknown finding",
			workload: resolver.WorkloadResult{
				Ref:  resolver.WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				Mode: resolver.ModeUnknown,
				MTLS: resolver.MTLSResult{
					Effective:              resolver.MTLSPermissive,
					ClientTLSContradiction: false,
					Chain:                  []resolver.Step{{Order: 1, Kind: "MeshConfigDefault", Effect: "default"}},
				},
			},
			wantFindings:      1,
			wantStatus:        "unknown",
			wantConfidence:    "unavailable",
			wantUnknownReason: "data plane membership unavailable",
		},
		{
			name: "unknown mTLS emits unknown finding",
			workload: resolver.WorkloadResult{
				Ref:  resolver.WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				Mode: resolver.ModeSidecar,
				MTLS: resolver.MTLSResult{
					Effective:     resolver.MTLSUnknown,
					UnknownReason: "PeerAuthentication resources unavailable",
					Chain:         []resolver.Step{},
				},
			},
			wantFindings:      1,
			wantStatus:        "unknown",
			wantConfidence:    "unavailable",
			wantUnknownReason: "PeerAuthentication resources unavailable",
		},
		{
			name: "disabled mTLS emits open finding",
			workload: resolver.WorkloadResult{
				Ref:  resolver.WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				Mode: resolver.ModeSidecar,
				MTLS: resolver.MTLSResult{
					Effective: resolver.MTLSDisabled,
					Chain:     []resolver.Step{{Order: 1, Kind: "PeerAuthentication", Effect: "sets DISABLE"}},
				},
			},
			wantFindings:   1,
			wantStatus:     "open",
			wantConfidence: "resolved",
		},
		{
			name: "strict mTLS emits no provisional permissive finding",
			workload: resolver.WorkloadResult{
				Ref:  resolver.WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				Mode: resolver.ModeSidecar,
				MTLS: resolver.MTLSResult{
					Effective: resolver.MTLSStrict,
					Chain:     []resolver.Step{{Order: 1, Kind: "PeerAuthentication", Effect: "sets STRICT"}},
				},
			},
			wantFindings: 0,
		},
		{
			name: "strict mTLS with unknown data plane emits unknown finding",
			workload: resolver.WorkloadResult{
				Ref:  resolver.WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
				Mode: resolver.ModeUnknown,
				MTLS: resolver.MTLSResult{
					Effective: resolver.MTLSStrict,
					Chain:     []resolver.Step{{Order: 1, Kind: "PeerAuthentication", Effect: "sets STRICT"}},
				},
			},
			wantFindings:      1,
			wantStatus:        "unknown",
			wantConfidence:    "unavailable",
			wantUnknownReason: "data plane membership unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := buildReport(ScanInput{
				ScannerVersion:  "dev",
				ResolverVersion: resolver.New().Version(),
				ClusterContext:  "fixture",
				Scope:           ScanScope{AllNamespaces: true},
				WorkloadPostures: []resolver.WorkloadResult{{
					Ref:   tt.workload.Ref,
					Mode:  tt.workload.Mode,
					MTLS:  tt.workload.MTLS,
					Authz: resolver.New().ResolveAuthz(resolver.WorkloadInput{}),
				}},
			})

			if len(report.Findings) != tt.wantFindings {
				t.Fatalf("findings = %d, want %d", len(report.Findings), tt.wantFindings)
			}
			if tt.wantFindings == 0 {
				return
			}

			finding := report.Findings[0]
			if finding.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", finding.Status, tt.wantStatus)
			}
			if finding.Confidence != tt.wantConfidence {
				t.Fatalf("confidence = %q, want %q", finding.Confidence, tt.wantConfidence)
			}
			if tt.wantUnknownReason == "" {
				if finding.UnknownReason != "" {
					t.Fatalf("unknownReason = %q, want empty", finding.UnknownReason)
				}
				return
			}
			if !strings.Contains(finding.UnknownReason, tt.wantUnknownReason) {
				t.Fatalf("unknownReason = %q, want to contain %q", finding.UnknownReason, tt.wantUnknownReason)
			}
		})
	}
}
