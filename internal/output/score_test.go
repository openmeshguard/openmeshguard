package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/engine"
)

func TestFailsThresholdExhaustiveSeverityBoundaries(t *testing.T) {
	severities := []string{"info", "low", "medium", "high", "critical"}
	for _, threshold := range severities {
		for _, findingSeverity := range severities {
			name := threshold + "/" + findingSeverity
			t.Run(name, func(t *testing.T) {
				data := thresholdReportJSON(t, []finding{thresholdFinding("open", findingSeverity)})
				got, err := FailsThreshold(bytes.NewReader(data), threshold, false)
				if err != nil {
					t.Fatalf("FailsThreshold: %v", err)
				}
				want := severityRank(findingSeverity) >= severityRank(threshold)
				if got != want {
					t.Fatalf("FailsThreshold(%s finding, %s threshold) = %t, want %t", findingSeverity, threshold, got, want)
				}
			})
		}
	}
}

func TestFailsThresholdStatusAndUnknownContract(t *testing.T) {
	tests := []struct {
		name          string
		findings      []finding
		threshold     string
		failOnUnknown bool
		want          bool
	}{
		{name: "clean", threshold: "info"},
		{name: "threshold disabled", findings: []finding{thresholdFinding("open", "critical")}},
		{name: "excepted critical excluded", findings: []finding{thresholdFinding("excepted", "critical")}, threshold: "info"},
		{name: "not applicable critical excluded", findings: []finding{thresholdFinding("not-applicable", "critical")}, threshold: "info"},
		{name: "unknown excluded by default", findings: []finding{thresholdFinding("unknown", "critical")}, threshold: "critical"},
		{name: "unknown opt in", findings: []finding{thresholdFinding("unknown", "info")}, threshold: "critical", failOnUnknown: true, want: true},
		{name: "unknown opt in without severity threshold", findings: []finding{thresholdFinding("unknown", "info")}, failOnUnknown: true, want: true},
		{name: "open expired risk still fails", findings: []finding{thresholdFinding("open", "high")}, threshold: "high", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := thresholdReportJSON(t, tt.findings)
			got, err := FailsThreshold(bytes.NewReader(data), tt.threshold, tt.failOnUnknown)
			if err != nil {
				t.Fatalf("FailsThreshold: %v", err)
			}
			if got != tt.want {
				t.Fatalf("FailsThreshold = %t, want %t", got, tt.want)
			}
		})
	}

	data := thresholdReportJSON(t, nil)
	if _, err := FailsThreshold(bytes.NewReader(data), "urgent", false); err == nil {
		t.Fatal("FailsThreshold accepted an invalid severity")
	}
}

func TestReportScoresWeightsKnownCategoriesAndAppliesCriticalCap(t *testing.T) {
	authzRate := 0.5
	mtlsRate := 1.0
	input := engine.Result{
		Scores: []engine.CategoryScore{
			{Category: "authz", Grade: "F", PassRate: &authzRate, Evaluated: 2},
			{Category: "governance", Grade: "unknown"},
			{Category: "mtls", Grade: "A", PassRate: &mtlsRate, Evaluated: 1},
		},
		NamespaceScores: []engine.NamespaceScore{{
			Namespace:   "payments",
			Environment: "production",
			Categories: []engine.CategoryScore{
				{Category: "authz", Grade: "F", PassRate: &authzRate, Evaluated: 2},
				{Category: "mtls", Grade: "A", PassRate: &mtlsRate, Evaluated: 1},
			},
			ScoreWeights: map[string]float64{"authz": 25, "governance": 20, "mtls": 25},
			CriticalCap:  59,
		}},
		ScoreWeights: map[string]float64{"authz": 25, "governance": 20, "mtls": 25},
		CriticalCap:  59,
	}

	uncapped := reportScores(input)
	if uncapped.Overall == nil || *uncapped.Overall != 75 {
		t.Fatalf("uncapped overall = %v, want 75", uncapped.Overall)
	}
	if len(uncapped.Namespaces) != 1 || uncapped.Namespaces[0].Score == nil || *uncapped.Namespaces[0].Score != 75 {
		t.Fatalf("uncapped namespace scores = %#v", uncapped.Namespaces)
	}

	input.Findings = []engine.Finding{{
		ID: "critical", Severity: "critical", Status: "open",
		Resources: []engine.ResourceRef{{Kind: "Deployment", Namespace: "payments", Name: "api"}},
	}}
	capped := reportScores(input)
	if capped.Overall == nil || *capped.Overall != 59 {
		t.Fatalf("capped overall = %v, want 59", capped.Overall)
	}
	if !capped.Namespaces[0].Capped || capped.Namespaces[0].Score == nil || *capped.Namespaces[0].Score != 59 {
		t.Fatalf("capped namespace = %#v, want 59 and capped=true", capped.Namespaces[0])
	}

	input.Findings[0].Status = "excepted"
	excepted := reportScores(input)
	if excepted.Overall == nil || *excepted.Overall != 75 || excepted.Namespaces[0].Capped {
		t.Fatalf("excepted critical changed score: %#v", excepted)
	}
	input.Findings[0].Status = "unknown"
	unknown := reportScores(input)
	if unknown.Overall == nil || *unknown.Overall != 75 || unknown.Namespaces[0].Capped {
		t.Fatalf("unknown critical changed score: %#v", unknown)
	}
}

func TestReportScoresIncludesClusterScopedControlsInOverall(t *testing.T) {
	globalRate := 0.5
	namespaceRate := 1.0
	weights := map[string]float64{
		"authz": 0, "exposure": 0, "governance": 100, "lifecycle": 0, "mtls": 0,
	}
	scored := reportScores(engine.Result{
		Scores: []engine.CategoryScore{{
			Category: "governance", Grade: "F", PassRate: &globalRate, Evaluated: 2,
		}},
		NamespaceScores: []engine.NamespaceScore{{
			Namespace: "payments",
			Categories: []engine.CategoryScore{{
				Category: "governance", Grade: "A", PassRate: &namespaceRate, Evaluated: 1,
			}},
			ScoreWeights: weights,
			CriticalCap:  59,
		}},
		ScoreWeights: weights,
		CriticalCap:  59,
	})
	if scored.Overall == nil || *scored.Overall != 50 {
		t.Fatalf("overall = %v, want 50 from global category aggregate", scored.Overall)
	}
	if scored.Namespaces[0].Score == nil || *scored.Namespaces[0].Score != 100 {
		t.Fatalf("namespace score = %#v, want unaffected namespace score 100", scored.Namespaces[0])
	}
}

func TestWriteScoreReadsCanonicalScoreWithoutRecomputation(t *testing.T) {
	scoreValue := 87.5
	data := thresholdReportJSON(t, []finding{thresholdFinding("unknown", "critical")})
	var canonical report
	if err := json.Unmarshal(data, &canonical); err != nil {
		t.Fatalf("decode test report: %v", err)
	}
	canonical.Scores = scores{
		Overall: &scoreValue,
		Categories: []scoreCategory{{
			Category: "mtls", Grade: "B", PassRate: floatPointer(0.875), Evaluated: 8, Unknown: 3,
		}},
		Namespaces: []namespaceScore{{Namespace: "payments", Score: floatPointer(59), Capped: true}},
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("encode test report: %v", err)
	}

	var rendered bytes.Buffer
	if err := WriteScore(&rendered, bytes.NewReader(data), "payments"); err != nil {
		t.Fatalf("WriteScore: %v", err)
	}
	for _, want := range []string{
		"score for namespace payments: 59.0/100 (critical cap applied)",
		"Cluster category grades (namespace category grades are not present in canonical JSON):",
		"mtls         B  pass=87.5%",
		"Unknown findings: 1 (excluded from score and exit status by default)",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("score output missing %q:\n%s", want, rendered.String())
		}
	}
}

func thresholdFinding(status, severity string) finding {
	item := finding{
		ID:           "MG-TEST-001-deadbeef0001",
		ControlID:    "MG-TEST-001",
		Severity:     severity,
		EvidenceType: "config",
		Status:       status,
		Confidence:   "resolved",
		Resources:    []resourceRef{{Kind: "Deployment", Namespace: "payments", Name: "api"}},
		Reasoning:    "Test reasoning.",
	}
	if status == "unknown" {
		item.UnknownReason = "evidence unavailable"
	}
	return item
}

func thresholdReportJSON(t *testing.T, findings []finding) []byte {
	t.Helper()
	data, err := json.Marshal(report{
		SchemaVersion:     schemaVersion,
		GeneratedAt:       "2026-07-25T00:00:00Z",
		Scanner:           scanner{Version: "dev", ResolverVersion: "mtls/v5,authz/v8", ControlPacks: []controlPack{}},
		Scan:              scan{ClusterContext: "fixture", Scope: scope{AllNamespaces: true}},
		PermissionSummary: []permission{},
		Inventory: inventorySummary{
			Counts:       map[string]int{},
			DataPlane:    dataPlane{Mode: "unknown"},
			MultiCluster: multiCluster{},
		},
		WorkloadPostures: []canonicalWorkloadPosture{},
		Findings:         append([]finding{}, findings...),
		Scores:           scores{Categories: []scoreCategory{}},
	})
	if err != nil {
		t.Fatalf("encode threshold report: %v", err)
	}
	return data
}

func floatPointer(value float64) *float64 {
	return &value
}
