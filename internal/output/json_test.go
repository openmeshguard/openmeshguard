package output

import (
	"strings"
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

func TestEngineFindingsReplaceProvisionalPathEndToEnd(t *testing.T) {
	packs, err := engine.LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}

	tests := []struct {
		name         string
		workload     engine.WorkloadInput
		wantCount    int
		wantStatuses map[string]string
	}{
		{
			name: "mixed by port produces explicit open findings",
			workload: engine.WorkloadInput{
				Posture: workloadPosture(
					resolver.ModeSidecar,
					resolver.MTLSMixedByPort,
					map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict, 9090: resolver.MTLSDisabled},
				),
				Namespace: engine.NamespaceInput{Name: "payments"},
				Availability: map[string]engine.Availability{
					"mtls.clientTLSContradiction": {Available: true},
				},
			},
			wantCount: 3,
			wantStatuses: map[string]string{
				"MG-MTLS-001": "open",
				"MG-MTLS-002": "open",
				"MG-MTLS-003": "open",
			},
		},
		{
			name: "not in mesh produces only not-applicable findings",
			workload: engine.WorkloadInput{
				Posture:   workloadPosture(resolver.ModeNotApplicable, resolver.MTLSNotInMesh, nil),
				Namespace: engine.NamespaceInput{Name: "payments"},
			},
			wantCount: 3,
			wantStatuses: map[string]string{
				"MG-MTLS-001": "not-applicable",
				"MG-MTLS-002": "not-applicable",
				"MG-MTLS-003": "not-applicable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated, err := engine.Evaluate(packs, engine.Input{Workloads: []engine.WorkloadInput{tt.workload}})
			if err != nil {
				t.Fatalf("evaluate controls: %v", err)
			}
			report := buildReport(ScanInput{
				ScannerVersion: "dev", ResolverVersion: resolver.New().Version(),
				ClusterContext: "fixture", Scope: ScanScope{AllNamespaces: true},
				WorkloadPostures: []resolver.WorkloadResult{tt.workload.Posture},
			}, packs, evaluated)
			if len(report.Findings) != tt.wantCount {
				t.Fatalf("findings = %d, want %d: %#v", len(report.Findings), tt.wantCount, report.Findings)
			}
			for _, finding := range report.Findings {
				if finding.Status != tt.wantStatuses[finding.ControlID] {
					t.Fatalf("%s status = %q, want %q", finding.ControlID, finding.Status, tt.wantStatuses[finding.ControlID])
				}
				if !strings.HasPrefix(finding.ID, finding.ControlID+"-") {
					t.Fatalf("engine finding ID = %q, want control prefix", finding.ID)
				}
				if finding.Status == "unknown" {
					t.Fatalf("unexpected contradictory unknown finding: %#v", finding)
				}
			}
		})
	}
}

func TestDefaultOutputMakesUnwiredEvidenceUnknown(t *testing.T) {
	posture := workloadPosture(resolver.ModeSidecar, resolver.MTLSPermissive, nil)
	packs, err := engine.LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	evaluated, err := engine.Evaluate(packs, defaultEngineInput(ScanInput{WorkloadPostures: []resolver.WorkloadResult{posture}}))
	if err != nil {
		t.Fatalf("evaluate controls: %v", err)
	}
	report := buildReport(ScanInput{WorkloadPostures: []resolver.WorkloadResult{posture}}, packs, evaluated)
	statuses := map[string]string{}
	unknownReasons := map[string]string{}
	for _, finding := range report.Findings {
		statuses[finding.ControlID] = finding.Status
		unknownReasons[finding.ControlID] = finding.UnknownReason
	}
	if statuses["MG-MTLS-001"] != "open" {
		t.Fatalf("MG-MTLS-001 status = %q, want open", statuses["MG-MTLS-001"])
	}
	for _, controlID := range []string{"MG-MTLS-002", "MG-MTLS-003"} {
		if statuses[controlID] != "unknown" {
			t.Fatalf("%s status = %q, want unknown", controlID, statuses[controlID])
		}
		if unknownReasons[controlID] == "" {
			t.Fatalf("%s missing unknownReason", controlID)
		}
	}
}

func workloadPosture(mode resolver.DataPlaneMode, effective resolver.MTLSEffective, byPort map[int32]resolver.MTLSEffective) resolver.WorkloadResult {
	return resolver.WorkloadResult{
		Ref:  resolver.WorkloadRef{Cluster: "cluster-a", Namespace: "payments", Name: "api", Kind: "Deployment"},
		Mode: mode,
		MTLS: resolver.MTLSResult{
			Effective: effective,
			ByPort:    byPort,
			Chain:     []resolver.Step{{Order: 1, Kind: "PeerAuthentication", Namespace: "payments", Name: "default", Effect: "sets effective mTLS"}},
		},
		Authz: resolver.AuthzResult{
			Effective:     resolver.AuthzUnknown,
			Chain:         []resolver.Step{},
			UnknownReason: "authorization resolver not yet implemented (M5)",
		},
	}
}
