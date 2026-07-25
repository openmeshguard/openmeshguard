package engine

import (
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

func TestBuiltinEnvironmentControlOutcomes(t *testing.T) {
	pack := builtinPackForTest(t, "MG-ENV-001")
	tests := []struct {
		name       string
		namespace  NamespaceInput
		wantStatus string
		wantCount  int
	}{
		{name: "pass", namespace: NamespaceInput{Name: "payments", Environment: "production", EnvironmentKnown: true, MeshEnrollment: "enrolled"}},
		{name: "fail", namespace: NamespaceInput{Name: "payments", Environment: "unclassified", EnvironmentKnown: true, MeshEnrollment: "enrolled"}, wantStatus: statusOpen, wantCount: 1},
		{name: "unknown", namespace: NamespaceInput{Name: "payments", MeshEnrollment: "enrolled", Availability: map[string]Availability{"environment": {Reason: "namespace labels unavailable"}}}, wantStatus: statusUnknown, wantCount: 1},
		{name: "not applicable", namespace: NamespaceInput{Name: "payments", Environment: "production", EnvironmentKnown: true, MeshEnrollment: "not-enrolled"}, wantStatus: statusNotApplicable, wantCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Evaluate([]Pack{pack}, Input{Namespaces: []NamespaceInput{tt.namespace}, NamespaceTargetsComplete: true})
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			assertSingleStatus(t, result, tt.wantCount, tt.wantStatus)
		})
	}
}

func TestBuiltinOwnershipControlOutcomes(t *testing.T) {
	tests := []struct {
		controlID string
		cases     []struct {
			name       string
			mutate     func(*WorkloadInput)
			wantStatus string
			wantCount  int
		}
	}{
		{
			controlID: "MG-OWN-001",
			cases: []struct {
				name       string
				mutate     func(*WorkloadInput)
				wantStatus string
				wantCount  int
			}{
				{name: "pass"},
				{name: "fail", mutate: func(workload *WorkloadInput) { workload.Owner = "" }, wantStatus: statusOpen, wantCount: 1},
				{name: "unknown", mutate: func(workload *WorkloadInput) { workload.Owner = ""; workload.OwnerKnown = false }, wantStatus: statusUnknown, wantCount: 1},
				{name: "not applicable", mutate: func(workload *WorkloadInput) { workload.Posture.Mode = resolver.ModeNotApplicable }, wantStatus: statusNotApplicable, wantCount: 1},
			},
		},
		{
			controlID: "MG-OWN-002",
			cases: []struct {
				name       string
				mutate     func(*WorkloadInput)
				wantStatus string
				wantCount  int
			}{
				{name: "pass"},
				{name: "fail", mutate: func(workload *WorkloadInput) { workload.AppID = "" }, wantStatus: statusOpen, wantCount: 1},
				{name: "unknown", mutate: func(workload *WorkloadInput) { workload.AppID = ""; workload.AppIDKnown = false }, wantStatus: statusUnknown, wantCount: 1},
				{name: "not applicable", mutate: func(workload *WorkloadInput) { workload.Posture.Mode = resolver.ModeNotApplicable }, wantStatus: statusNotApplicable, wantCount: 1},
				{name: "unclassified is environment filtered", mutate: func(workload *WorkloadInput) {
					workload.Environment = "unclassified"
					workload.Namespace.Environment = "unclassified"
				}},
			},
		},
	}
	for _, control := range tests {
		pack := builtinPackForTest(t, control.controlID)
		for _, tt := range control.cases {
			t.Run(control.controlID+"/"+tt.name, func(t *testing.T) {
				workload := workloadWithMTLS(resolver.MTLSStrict, map[int32]resolver.MTLSEffective{})
				if tt.mutate != nil {
					tt.mutate(&workload)
				}
				result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
				if err != nil {
					t.Fatalf("Evaluate returned error: %v", err)
				}
				assertSingleStatus(t, result, tt.wantCount, tt.wantStatus)
			})
		}
	}
}

func TestBuiltinExceptionControlOutcomes(t *testing.T) {
	tests := []struct {
		controlID  string
		name       string
		resource   ResourceInput
		wantStatus string
		wantCount  int
	}{
		{controlID: "MG-EXC-001", name: "pass", resource: exceptionResource(map[string]any{"validationErrors": []any{}})},
		{controlID: "MG-EXC-001", name: "fail", resource: exceptionResource(map[string]any{"validationErrors": []any{"missing approver"}}), wantStatus: statusOpen, wantCount: 1},
		{controlID: "MG-EXC-001", name: "unknown", resource: exceptionResource(map[string]any{}), wantStatus: statusUnknown, wantCount: 1},
		{controlID: "MG-EXC-002", name: "pass", resource: exceptionResource(map[string]any{"valid": true, "expired": false})},
		{controlID: "MG-EXC-002", name: "fail", resource: exceptionResource(map[string]any{"valid": true, "expired": true}), wantStatus: statusOpen, wantCount: 1},
		{controlID: "MG-EXC-002", name: "unknown", resource: exceptionResource(map[string]any{"valid": true}), wantStatus: statusUnknown, wantCount: 1},
		{controlID: "MG-EXC-002", name: "not applicable", resource: exceptionResource(map[string]any{"valid": false, "expired": false}), wantStatus: statusNotApplicable, wantCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.controlID+"/"+tt.name, func(t *testing.T) {
			result, err := Evaluate([]Pack{builtinPackForTest(t, tt.controlID)}, Input{Resources: []ResourceInput{tt.resource}})
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			assertSingleStatus(t, result, tt.wantCount, tt.wantStatus)
		})
	}
}

func exceptionResource(fields map[string]any) ResourceInput {
	return ResourceInput{
		APIVersion:      "openmeshguard.io/v1alpha1",
		Kind:            "Exception",
		Name:            "EXC-42",
		Fields:          fields,
		EvidenceSources: []string{"exception-record"},
	}
}

func builtinPackForTest(t *testing.T, controlID string) Pack {
	t.Helper()
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	return packWithControl(t, packs, controlID)
}

func assertSingleStatus(t *testing.T, result Result, wantCount int, wantStatus string) {
	t.Helper()
	if len(result.Findings) != wantCount {
		t.Fatalf("findings = %#v, want count %d", result.Findings, wantCount)
	}
	if wantCount == 1 && result.Findings[0].Status != wantStatus {
		t.Fatalf("status = %q, want %q", result.Findings[0].Status, wantStatus)
	}
}
