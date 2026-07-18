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
			wantCount: 2,
			wantStatuses: map[string]string{
				"MG-MTLS-001": "open",
				"MG-MTLS-002": "open",
			},
		},
		{
			name: "not in mesh produces only not-applicable findings",
			workload: engine.WorkloadInput{
				Posture:   workloadPosture(resolver.ModeNotApplicable, resolver.MTLSNotInMesh, nil),
				Namespace: engine.NamespaceInput{Name: "payments"},
			},
			wantCount: 11,
			wantStatuses: map[string]string{
				"MG-AUTHZ-001": "not-applicable",
				"MG-AUTHZ-002": "not-applicable",
				"MG-AUTHZ-003": "not-applicable",
				"MG-AUTHZ-004": "not-applicable",
				"MG-AUTHZ-005": "not-applicable",
				"MG-AUTHZ-006": "not-applicable",
				"MG-AUTHZ-007": "not-applicable",
				"MG-MTLS-001":  "not-applicable",
				"MG-MTLS-002":  "not-applicable",
				"MG-MTLS-003":  "not-applicable",
				"MG-MTLS-007":  "not-applicable",
			},
		},
		{
			name: "confirmed disabled posture retains critical finding without unwired producers",
			workload: engine.WorkloadInput{
				Posture:   workloadPosture(resolver.ModeSidecar, resolver.MTLSDisabled, nil),
				Namespace: engine.NamespaceInput{Name: "payments"},
			},
			wantCount: 3,
			wantStatuses: map[string]string{
				"MG-MTLS-001": "open",
				"MG-MTLS-002": "unknown",
				"MG-MTLS-003": "open",
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
				if finding.Status == "unknown" && tt.wantStatuses[finding.ControlID] != "unknown" {
					t.Fatalf("unexpected contradictory unknown finding: %#v", finding)
				}
				if finding.Status == "unknown" && finding.UnknownReason == "" {
					t.Fatalf("unknown finding missing reason: %#v", finding)
				}
				if finding.ControlID == "MG-MTLS-001" && finding.Status == "open" {
					if finding.Remediation == nil || !strings.Contains(finding.Remediation.SuggestedYAML, "namespace: payments") {
						t.Fatalf("MG-MTLS-001 remediation = %#v, want rendered suggested YAML", finding.Remediation)
					}
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
	if statuses["MG-MTLS-002"] != "unknown" || unknownReasons["MG-MTLS-002"] == "" {
		t.Fatalf("MG-MTLS-002 status/reason = %q/%q, want unknown with reason", statuses["MG-MTLS-002"], unknownReasons["MG-MTLS-002"])
	}
	if _, exists := statuses["MG-MTLS-003"]; exists {
		t.Fatalf("MG-MTLS-003 status = %q, want no finding for known non-disabled posture", statuses["MG-MTLS-003"])
	}
}

func workloadPosture(mode resolver.DataPlaneMode, effective resolver.MTLSEffective, byPort map[int32]resolver.MTLSEffective) resolver.WorkloadResult {
	knownFalse := false
	return resolver.WorkloadResult{
		Ref:  resolver.WorkloadRef{Cluster: "cluster-a", Namespace: "payments", Name: "api", Kind: "Deployment"},
		Mode: mode,
		MTLS: resolver.MTLSResult{
			Effective:              effective,
			ByPort:                 byPort,
			ClientTLSContradiction: &knownFalse,
			Chain:                  []resolver.Step{{Order: 1, Kind: "PeerAuthentication", Namespace: "payments", Name: "default", Effect: "sets effective mTLS"}},
		},
		Authz: resolver.AuthzResult{
			Effective:       resolver.AuthzDefaultDenyExplicitAllow,
			BroadAllow:      &knownFalse,
			PoliciesInScope: []string{"payments/default-deny", "payments/api"},
			Chain:           []resolver.Step{{Order: 1, Kind: "AuthorizationPolicy", Namespace: "payments", Name: "default-deny", Effect: "sets effective authorization"}},
		},
	}
}
