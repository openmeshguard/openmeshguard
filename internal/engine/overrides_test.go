package engine

import (
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

func TestEvaluateAppliesEnvironmentParametersFiltersAndSeverity(t *testing.T) {
	pack, err := decodeAndValidate("override.yaml", []byte(`apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: overrides, version: 1.0.0}
params: {requiredOwner: default-team}
controls:
  - id: ACME-OWN-001
    title: Workloads must use the configured owner
    category: governance
    severity: low
    evidenceType: context
    scope: workload
    environments: [staging]
    requires: [workload.owner, params.requiredOwner]
    applicability: 'true'
    expression: 'workload.owner == params.requiredOwner'
    message: Wrong owner.
    remediation: {guidance: Correct the owner.}
`), SourceUser)
	if err != nil {
		t.Fatalf("decode pack: %v", err)
	}
	environments := []string{"production"}
	workload := workloadWithMTLS("strict", map[int32]resolver.MTLSEffective{})
	workload.Namespace = NamespaceInput{Name: "payments", Environment: "production", EnvironmentKnown: true}
	workload.Environment = "production"
	workload.EnvironmentKnown = true
	workload.Owner = "payments-team"
	workload.OwnerKnown = true
	result, err := Evaluate([]Pack{pack}, Input{
		Workloads:         []WorkloadInput{workload},
		Params:            map[string]any{"requiredOwner": "global-team"},
		EnvironmentParams: map[string]map[string]any{"production": {"requiredOwner": "production-team"}},
		ControlOverrides: map[string]ControlOverride{
			"ACME-OWN-001": {
				Environments:          &environments,
				SeverityByEnvironment: map[string]string{"production": "critical"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("findings = %#v, want one environment-enabled failure", result.Findings)
	}
	if result.Findings[0].Severity != "critical" {
		t.Fatalf("severity = %q, want critical", result.Findings[0].Severity)
	}
}
