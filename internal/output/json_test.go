package output

import (
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

func TestProvisionalFindingIsUnknownWhenDataPlaneModeUnknown(t *testing.T) {
	report := buildReport(ScanInput{
		ScannerVersion:  "dev",
		ResolverVersion: resolver.ProvisionalVersion(),
		ClusterContext:  "fixture",
		Scope:           ScanScope{AllNamespaces: true},
		WorkloadPostures: []resolver.WorkloadResult{{
			Ref:  resolver.WorkloadRef{Namespace: "payments", Name: "api", Kind: "Deployment"},
			Mode: resolver.ModeUnknown,
			MTLS: resolver.MTLSResult{
				Effective:              resolver.MTLSPermissive,
				ClientTLSContradiction: false,
				Chain:                  []resolver.Step{{Order: 1, Kind: "MeshConfigDefault", Effect: "default"}},
			},
			Authz: resolver.AuthzResult{
				Effective:     resolver.AuthzUnknown,
				Chain:         []resolver.Step{},
				UnknownReason: "not yet implemented (M2)",
			},
		}},
	})

	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(report.Findings))
	}
	finding := report.Findings[0]
	if finding.Status != "unknown" {
		t.Fatalf("status = %q, want unknown", finding.Status)
	}
	if finding.Confidence != "unavailable" {
		t.Fatalf("confidence = %q, want unavailable", finding.Confidence)
	}
	if finding.UnknownReason == "" {
		t.Fatal("unknown finding missing unknownReason")
	}
}
