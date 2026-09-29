package engine

import "testing"

func TestScoringConfigurationUsesPublishedDataAndEnvironmentOverrides(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	weights, criticalCap, err := scoringConfiguration(packs, nil)
	if err != nil {
		t.Fatalf("read published scoring data: %v", err)
	}
	var total float64
	for _, weight := range weights {
		total += weight
	}
	if total != 100 || weights["mtls"] != 25 || weights["governance"] != 20 || criticalCap != 59 {
		t.Fatalf("published scoring config = weights %#v cap %v", weights, criticalCap)
	}

	categories := map[string]map[string]*categoryAccumulator{
		"payments": {
			"mtls":  {pass: 1},
			"authz": {fail: 1},
		},
	}
	environmentParams := map[string]map[string]any{
		"production": {
			"scoring": map[string]any{
				"weights": map[string]any{
					"mtls": 80, "authz": 20, "exposure": 0, "governance": 0, "lifecycle": 0,
				},
				"criticalCap": 49,
			},
		},
	}
	namespaceScores, err := buildNamespaceScores(
		categories,
		map[string]string{"payments": "production"},
		packs,
		nil,
		environmentParams,
	)
	if err != nil {
		t.Fatalf("build namespace scores: %v", err)
	}
	if len(namespaceScores) != 1 ||
		namespaceScores[0].ScoreWeights["mtls"] != 80 ||
		namespaceScores[0].ScoreWeights["authz"] != 20 ||
		namespaceScores[0].CriticalCap != 49 {
		t.Fatalf("environment namespace scoring = %#v", namespaceScores)
	}
}

func TestWeightedCategoryWithoutControlsRemainsExplicitlyUnknown(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	result, err := Evaluate(packs, Input{})
	if err != nil {
		t.Fatalf("evaluate built-ins: %v", err)
	}
	for _, category := range result.Scores {
		if category.Category != "lifecycle" {
			continue
		}
		if category.Grade != "unknown" || category.PassRate != nil || category.Evaluated != 0 {
			t.Fatalf("lifecycle score = %#v, want explicit unknown", category)
		}
		return
	}
	t.Fatal("published lifecycle score dimension disappeared because it has no current controls")
}

func TestEvaluateRetainsClusterScopedControlsForRollup(t *testing.T) {
	result, err := Evaluate(
		[]Pack{builtinPackForTest(t, "MG-EXC-001")},
		Input{Resources: []ResourceInput{exceptionResource(map[string]any{
			"validationErrors": []any{"invalid exception"},
		})}},
	)
	if err != nil {
		t.Fatalf("evaluate cluster-scoped exception control: %v", err)
	}
	if len(result.NamespaceScores) != 0 {
		t.Fatalf("cluster-scoped control created namespace scores: %#v", result.NamespaceScores)
	}
	for _, category := range result.ClusterScores {
		if category.Category != "governance" {
			continue
		}
		if category.PassRate == nil || *category.PassRate != 0 || category.Evaluated != 1 {
			t.Fatalf("cluster governance score = %#v, want one failed evaluation", category)
		}
		return
	}
	t.Fatalf("cluster-scoped governance score missing: %#v", result.ClusterScores)
}

func TestScoringConfigurationRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name    string
		scoring map[string]any
	}{
		{
			name: "missing weights",
			scoring: map[string]any{
				"criticalCap": 59,
			},
		},
		{
			name: "negative weight",
			scoring: map[string]any{
				"weights": map[string]any{
					"mtls": -1, "authz": 25, "exposure": 25, "governance": 20, "lifecycle": 5,
				},
				"criticalCap": 59,
			},
		},
		{
			name: "cap over one hundred",
			scoring: map[string]any{
				"weights": map[string]any{
					"mtls": 25, "authz": 25, "exposure": 25, "governance": 20, "lifecycle": 5,
				},
				"criticalCap": 101,
			},
		},
		{
			name: "unknown category",
			scoring: map[string]any{
				"weights": map[string]any{
					"mtls": 25, "authz": 25, "exposure": 25, "governance": 20, "lifecycle": 5, "egres": 1,
				},
				"criticalCap": 59,
			},
		},
		{
			name: "zero total",
			scoring: map[string]any{
				"weights": map[string]any{
					"mtls": 0, "authz": 0, "exposure": 0, "governance": 0, "lifecycle": 0,
				},
				"criticalCap": 59,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := scoringConfiguration(nil, map[string]any{"scoring": tt.scoring})
			if err == nil {
				t.Fatalf("scoringConfiguration accepted %#v", tt.scoring)
			}
		})
	}
}
