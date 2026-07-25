package output

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"sort"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestWriteSARIFValidatesOfficialSchemaAndPreservesFindingParity(t *testing.T) {
	golden := readOutputFixture(t, filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"governance-context",
		"golden",
		"governance-active-exception.json",
	))
	canonical, err := readCanonicalReport(bytes.NewReader(golden))
	if err != nil {
		t.Fatalf("read canonical golden: %v", err)
	}

	var first bytes.Buffer
	if err := WriteSARIF(&first, bytes.NewReader(golden)); err != nil {
		t.Fatalf("write SARIF: %v", err)
	}
	var second bytes.Buffer
	if err := WriteSARIF(&second, bytes.NewReader(golden)); err != nil {
		t.Fatalf("write SARIF a second time: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("SARIF projection is not deterministic for identical canonical JSON")
	}

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(filepath.Join("testdata", "sarif-schema-2.1.0.json"))
	if err != nil {
		t.Fatalf("compile official SARIF schema: %v", err)
	}
	raw, err := jsonschema.UnmarshalJSON(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatalf("decode SARIF: %v", err)
	}
	if err := schema.Validate(raw); err != nil {
		t.Fatalf("SARIF projection does not match the official schema: %v\n%s", err, first.String())
	}

	var projected sarifLog
	if err := json.Unmarshal(first.Bytes(), &projected); err != nil {
		t.Fatalf("decode typed SARIF: %v", err)
	}
	if len(projected.Runs) != 1 {
		t.Fatalf("SARIF runs = %d, want 1", len(projected.Runs))
	}
	results := projected.Runs[0].Results
	if len(results) != len(canonical.Findings) {
		t.Fatalf("SARIF results = %d, canonical findings = %d", len(results), len(canonical.Findings))
	}

	canonicalIDs := make([]string, 0, len(canonical.Findings))
	for _, item := range canonical.Findings {
		canonicalIDs = append(canonicalIDs, item.ID)
	}
	projectedIDs := make([]string, 0, len(results))
	for _, result := range results {
		projectedIDs = append(projectedIDs, result.Fingerprints["openmeshguardFindingId/v1"])
		if len(result.Locations) == 0 {
			t.Errorf("SARIF result %s has no resource location", result.RuleID)
		}
	}
	sort.Strings(canonicalIDs)
	sort.Strings(projectedIDs)
	if !equalStrings(canonicalIDs, projectedIDs) {
		t.Fatalf("SARIF finding IDs = %#v, want exact canonical parity %#v", projectedIDs, canonicalIDs)
	}

	uniqueControls := map[string]bool{}
	for _, item := range canonical.Findings {
		uniqueControls[item.ControlID] = true
	}
	if got := len(projected.Runs[0].Tool.Driver.Rules); got != len(uniqueControls) {
		t.Fatalf("SARIF rules = %d, unique finding controls = %d", got, len(uniqueControls))
	}
}

func TestSARIFStatusProjectionIsExplicitAndNonMisleading(t *testing.T) {
	tests := []struct {
		status          string
		severity        string
		wantKind        string
		wantLevel       string
		wantSuppression bool
	}{
		{status: "open", severity: "critical", wantKind: "fail", wantLevel: "error"},
		{status: "excepted", severity: "critical", wantKind: "fail", wantLevel: "error", wantSuppression: true},
		{status: "unknown", severity: "critical", wantKind: "review", wantLevel: "warning"},
		{status: "not-applicable", severity: "critical", wantKind: "notApplicable", wantLevel: "none"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			item := finding{
				ID:           "MG-TEST-001-deadbeef0001",
				ControlID:    "MG-TEST-001",
				Title:        "Test control",
				Severity:     tt.severity,
				EvidenceType: "config",
				Status:       tt.status,
				Confidence:   "resolved",
				Resources:    []resourceRef{{Kind: "Deployment", Namespace: "payments", Name: "api"}},
				Reasoning:    "Canonical reasoning.",
			}
			if tt.status == "unknown" {
				item.UnknownReason = "evidence unavailable"
			}
			if tt.status == "excepted" {
				item.Exception = &exceptionEvidence{ID: "EXC-1", Ticket: "https://tickets.example/1"}
			}
			projected, err := projectSARIF(report{
				SchemaVersion: schemaVersion,
				Scanner:       scanner{Version: "dev"},
				Scan:          scan{ClusterContext: "fixture"},
				Findings:      []finding{item},
			})
			if err != nil {
				t.Fatalf("project SARIF: %v", err)
			}
			result := projected.Runs[0].Results[0]
			if result.Kind != tt.wantKind || result.Level != tt.wantLevel {
				t.Fatalf("SARIF status = kind %q level %q, want %q/%q", result.Kind, result.Level, tt.wantKind, tt.wantLevel)
			}
			if (len(result.Suppressions) > 0) != tt.wantSuppression {
				t.Fatalf("SARIF suppressions = %#v, want present=%t", result.Suppressions, tt.wantSuppression)
			}
			if result.Properties["openmeshguard.status"] != tt.status ||
				result.Properties["openmeshguard.severity"] != tt.severity {
				t.Fatalf("SARIF properties lost canonical status/severity: %#v", result.Properties)
			}
		})
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
