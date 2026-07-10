package engine

import (
	"strings"
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

func TestBuiltinControlsCoverEveryOutcome(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}

	tests := []struct {
		name          string
		controlID     string
		workload      WorkloadInput
		wantFindings  int
		wantStatus    string
		unknownReason string
		wantGrade     string
	}{
		{
			name:      "MG-MTLS-001 pass",
			controlID: "MG-MTLS-001",
			workload:  workloadWithMTLS(resolver.MTLSStrict, nil),
			wantGrade: "A",
		},
		{
			name:         "MG-MTLS-001 fail without exception matching",
			controlID:    "MG-MTLS-001",
			workload:     workloadWithMTLS(resolver.MTLSPermissive, nil),
			wantFindings: 1, wantStatus: statusOpen, wantGrade: "F",
		},
		{
			name:         "MG-MTLS-001 unknown",
			controlID:    "MG-MTLS-001",
			workload:     unknownWorkload("PeerAuthentication resources unavailable"),
			wantFindings: 1, wantStatus: statusUnknown,
			unknownReason: "PeerAuthentication resources unavailable", wantGrade: "unknown",
		},
		{
			name:         "MG-MTLS-001 not applicable",
			controlID:    "MG-MTLS-001",
			workload:     notInMeshWorkload(),
			wantFindings: 1, wantStatus: statusNotApplicable, wantGrade: "unknown",
		},
		{
			name:      "MG-MTLS-002 pass",
			controlID: "MG-MTLS-002",
			workload:  workloadWithMTLS(resolver.MTLSStrict, map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict}),
			wantGrade: "A",
		},
		{
			name:         "MG-MTLS-002 fail",
			controlID:    "MG-MTLS-002",
			workload:     workloadWithMTLS(resolver.MTLSMixedByPort, map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict, 9090: resolver.MTLSPermissive}),
			wantFindings: 1, wantStatus: statusOpen, wantGrade: "F",
		},
		{
			name:         "MG-MTLS-002 unknown",
			controlID:    "MG-MTLS-002",
			workload:     workloadWithMTLS(resolver.MTLSStrict, nil),
			wantFindings: 1, wantStatus: statusUnknown,
			unknownReason: "workload ports unavailable", wantGrade: "unknown",
		},
		{
			name:         "MG-MTLS-002 not applicable before requires",
			controlID:    "MG-MTLS-002",
			workload:     notInMeshWorkload(),
			wantFindings: 1, wantStatus: statusNotApplicable, wantGrade: "unknown",
		},
		{
			name:      "MG-MTLS-003 pass",
			controlID: "MG-MTLS-003",
			workload:  contradictionAvailable(workloadWithMTLS(resolver.MTLSStrict, map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict}), false),
			wantGrade: "A",
		},
		{
			name:         "MG-MTLS-003 fail",
			controlID:    "MG-MTLS-003",
			workload:     contradictionAvailable(workloadWithMTLS(resolver.MTLSMixedByPort, map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict, 9090: resolver.MTLSDisabled}), false),
			wantFindings: 1, wantStatus: statusOpen, wantGrade: "F",
		},
		{
			name:         "MG-MTLS-003 unknown",
			controlID:    "MG-MTLS-003",
			workload:     workloadWithMTLS(resolver.MTLSStrict, map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict}),
			wantFindings: 1, wantStatus: statusUnknown,
			unknownReason: "DestinationRule collection unavailable", wantGrade: "unknown",
		},
		{
			name:         "MG-MTLS-003 not applicable before requires",
			controlID:    "MG-MTLS-003",
			workload:     notInMeshWorkload(),
			wantFindings: 1, wantStatus: statusNotApplicable, wantGrade: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pack := packWithControl(t, packs, tt.controlID)
			result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{tt.workload}})
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			if len(result.Findings) != tt.wantFindings {
				t.Fatalf("findings = %d, want %d: %#v", len(result.Findings), tt.wantFindings, result.Findings)
			}
			if tt.wantFindings > 0 {
				finding := result.Findings[0]
				if finding.Status != tt.wantStatus {
					t.Fatalf("status = %q, want %q", finding.Status, tt.wantStatus)
				}
				if tt.wantStatus == statusOpen && finding.ControlID != tt.controlID {
					t.Fatalf("control ID = %q, want %q", finding.ControlID, tt.controlID)
				}
				if tt.unknownReason != "" && !strings.Contains(finding.UnknownReason, tt.unknownReason) {
					t.Fatalf("unknownReason = %q, want to contain %q", finding.UnknownReason, tt.unknownReason)
				}
			}
			if len(result.Scores) != 1 {
				t.Fatalf("scores = %#v, want one category", result.Scores)
			}
			if result.Scores[0].Grade != tt.wantGrade {
				t.Fatalf("grade = %q, want %q", result.Scores[0].Grade, tt.wantGrade)
			}
		})
	}
}

func TestRealScanMissingProducersBecomeUnknownThroughRequires(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	result, err := Evaluate(packs, Input{Workloads: []WorkloadInput{
		workloadWithMTLS(resolver.MTLSPermissive, nil),
	}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	statuses := map[string]string{}
	reasons := map[string]string{}
	for _, finding := range result.Findings {
		statuses[finding.ControlID] = finding.Status
		reasons[finding.ControlID] = finding.UnknownReason
	}
	if statuses["MG-MTLS-001"] != statusOpen {
		t.Fatalf("MG-MTLS-001 status = %q, want open", statuses["MG-MTLS-001"])
	}
	for _, controlID := range []string{"MG-MTLS-002", "MG-MTLS-003"} {
		if statuses[controlID] != statusUnknown {
			t.Fatalf("%s status = %q, want unknown", controlID, statuses[controlID])
		}
		if !strings.Contains(reasons[controlID], "workload ports unavailable") {
			t.Fatalf("%s unknownReason = %q, want missing port producer", controlID, reasons[controlID])
		}
	}
	if !strings.Contains(reasons["MG-MTLS-003"], "DestinationRule collection unavailable") {
		t.Fatalf("MG-MTLS-003 unknownReason = %q, want missing DestinationRule producer", reasons["MG-MTLS-003"])
	}
}

func TestEvaluateDeterministicIDsAndCategoryGrades(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	input := Input{Workloads: []WorkloadInput{
		workloadWithMTLS(resolver.MTLSPermissive, nil),
		workloadNamed("worker", resolver.MTLSStrict),
	}}
	first, err := Evaluate(packs, input)
	if err != nil {
		t.Fatalf("first Evaluate: %v", err)
	}
	second, err := Evaluate(packs, input)
	if err != nil {
		t.Fatalf("second Evaluate: %v", err)
	}
	if len(first.Findings) != len(second.Findings) {
		t.Fatalf("finding counts differ: %d and %d", len(first.Findings), len(second.Findings))
	}
	for index := range first.Findings {
		if first.Findings[index].ID != second.Findings[index].ID {
			t.Fatalf("finding ID changed: %q and %q", first.Findings[index].ID, second.Findings[index].ID)
		}
	}
	if len(first.Scores) != 1 || first.Scores[0].PassRate == nil || *first.Scores[0].PassRate != 0.5 || first.Scores[0].Grade != "F" {
		t.Fatalf("score = %#v, want mtls 50%% grade F", first.Scores)
	}
}

func TestEvaluateNamespaceAndResourceScopes(t *testing.T) {
	packYAML := []byte(`
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata:
  name: scoped-controls
  version: 1.0.0
controls:
  - id: ACME-ENV-001
    title: Production namespaces must belong to platform
    category: governance
    severity: medium
    evidenceType: context
    scope: namespace
    environments: [production]
    requires: [namespace.labels.team]
    applicability: 'true'
    expression: 'namespace.labels.team == "platform"'
    message: 'Namespace {{ .Namespace }} is not owned by platform.'
    remediation:
      guidance: Correct the team label.
  - id: ACME-GW-001
    title: Public gateways must not use wildcard hosts
    category: exposure
    severity: high
    evidenceType: config
    scope: resource
    environments: []
    match:
      kinds: [Gateway]
    requires: [resource.servers]
    applicability: 'resource.isPubliclyExposed'
    expression: '!resource.servers.exists(s, s.hosts.exists(h, h == "*"))'
    message: 'Gateway {{ .Resource }} exposes a wildcard host.'
    remediation:
      guidance: Replace wildcard hosts with explicit names.
`)
	pack, err := decodeAndValidate("scoped-controls.yaml", packYAML, SourceUser)
	if err != nil {
		t.Fatalf("decode scoped pack: %v", err)
	}
	result, err := Evaluate([]Pack{pack}, Input{
		Namespaces: []NamespaceInput{
			{Name: "platform-prod", Environment: "production", Labels: map[string]string{"team": "platform"}},
			{Name: "payments-prod", Environment: "production", Labels: map[string]string{"team": "payments"}},
			{Name: "unclassified", Labels: map[string]string{"team": "payments"}},
		},
		Resources: []ResourceInput{
			{
				APIVersion: "networking.istio.io/v1", Kind: "Gateway", Namespace: "istio-system", Name: "public",
				Fields: map[string]any{
					"isPubliclyExposed": true,
					"servers":           []any{map[string]any{"hosts": []any{"*"}}},
				},
			},
			{
				APIVersion: "networking.istio.io/v1", Kind: "Gateway", Namespace: "istio-system", Name: "private",
				Fields: map[string]any{
					"isPubliclyExposed": false,
					"servers":           []any{map[string]any{"hosts": []any{"internal.example"}}},
				},
			},
			{APIVersion: "networking.istio.io/v1", Kind: "ServiceEntry", Namespace: "payments", Name: "ignored"},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	statuses := map[string][]string{}
	resourceStatusCounts := map[string]int{}
	for _, finding := range result.Findings {
		statuses[finding.ControlID] = append(statuses[finding.ControlID], finding.Status)
		if finding.ControlID == "ACME-GW-001" {
			resourceStatusCounts[finding.Status]++
		}
	}
	if len(statuses["ACME-ENV-001"]) != 1 || statuses["ACME-ENV-001"][0] != statusOpen {
		t.Fatalf("namespace statuses = %#v, want one production failure with unclassified namespace filtered", statuses["ACME-ENV-001"])
	}
	if len(statuses["ACME-GW-001"]) != 2 || resourceStatusCounts[statusNotApplicable] != 1 || resourceStatusCounts[statusOpen] != 1 {
		t.Fatalf("resource statuses = %#v, want not-applicable and open Gateway results", statuses["ACME-GW-001"])
	}
}

func packWithControl(t *testing.T, packs []Pack, controlID string) Pack {
	t.Helper()
	for _, pack := range packs {
		for _, control := range pack.Controls {
			if control.ID == controlID {
				pack.Controls = []Control{control}
				return pack
			}
		}
	}
	t.Fatalf("built-in control %s not found", controlID)
	return Pack{}
}

func workloadWithMTLS(effective resolver.MTLSEffective, byPort map[int32]resolver.MTLSEffective) WorkloadInput {
	return WorkloadInput{
		Posture: resolver.WorkloadResult{
			Ref:  resolver.WorkloadRef{Cluster: "cluster-a", Namespace: "payments", Name: "api", Kind: "Deployment"},
			Mode: resolver.ModeSidecar,
			MTLS: resolver.MTLSResult{
				Effective: effective,
				ByPort:    byPort,
				Chain:     []resolver.Step{{Order: 1, Kind: "PeerAuthentication", Namespace: "payments", Name: "default", Effect: "sets effective mTLS"}},
			},
			Authz: resolver.AuthzResult{Effective: resolver.AuthzUnknown, Chain: []resolver.Step{}, UnknownReason: "authorization resolver not yet implemented (M5)"},
		},
		Namespace: NamespaceInput{Name: "payments", Labels: map[string]string{"team": "payments"}},
	}
}

func workloadNamed(name string, effective resolver.MTLSEffective) WorkloadInput {
	workload := workloadWithMTLS(effective, nil)
	workload.Posture.Ref.Name = name
	return workload
}

func contradictionAvailable(workload WorkloadInput, contradiction bool) WorkloadInput {
	workload.Posture.MTLS.ClientTLSContradiction = contradiction
	workload.Availability = map[string]Availability{
		"mtls.clientTLSContradiction": {Available: true},
	}
	return workload
}

func unknownWorkload(reason string) WorkloadInput {
	workload := workloadWithMTLS(resolver.MTLSUnknown, nil)
	workload.Posture.MTLS.UnknownReason = reason
	workload.Posture.MTLS.Chain = []resolver.Step{}
	return workload
}

func notInMeshWorkload() WorkloadInput {
	workload := workloadWithMTLS(resolver.MTLSNotInMesh, nil)
	workload.Posture.Mode = resolver.ModeNotApplicable
	workload.Posture.MTLS.Chain = []resolver.Step{{Order: 1, Kind: "DataPlane", Effect: "not enrolled"}}
	return workload
}
