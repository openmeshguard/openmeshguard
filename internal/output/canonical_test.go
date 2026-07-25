package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCanonicalProjectionInputRequiresCompleteFrozenSchema(t *testing.T) {
	valid := thresholdReportJSON(t, []finding{thresholdFinding("open", "high")})
	var document map[string]any
	if err := json.Unmarshal(valid, &document); err != nil {
		t.Fatalf("decode valid report: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "invalid finding status",
			mutate: func(report map[string]any) {
				report["findings"].([]any)[0].(map[string]any)["status"] = "bogus"
			},
		},
		{
			name: "invalid finding severity",
			mutate: func(report map[string]any) {
				report["findings"].([]any)[0].(map[string]any)["severity"] = "urgent"
			},
		},
		{
			name: "missing score categories",
			mutate: func(report map[string]any) {
				delete(report["scores"].(map[string]any), "categories")
			},
		},
		{
			name: "invalid resolution order",
			mutate: func(report map[string]any) {
				report["findings"].([]any)[0].(map[string]any)["resolutionChain"] = []any{
					map[string]any{"order": float64(0), "kind": "MeshConfigDefault", "effect": "invalid"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyBytes, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("copy valid report: %v", err)
			}
			var mutated map[string]any
			if err := json.Unmarshal(copyBytes, &mutated); err != nil {
				t.Fatalf("decode copied report: %v", err)
			}
			tt.mutate(mutated)
			data, err := json.Marshal(mutated)
			if err != nil {
				t.Fatalf("encode invalid report: %v", err)
			}

			for name, project := range map[string]func(*bytes.Buffer) error{
				"html": func(output *bytes.Buffer) error {
					return WriteHTML(output, bytes.NewReader(data))
				},
				"sarif": func(output *bytes.Buffer) error {
					return WriteSARIF(output, bytes.NewReader(data))
				},
				"score": func(output *bytes.Buffer) error {
					return WriteScore(output, bytes.NewReader(data), "")
				},
				"threshold": func(_ *bytes.Buffer) error {
					_, err := FailsThreshold(bytes.NewReader(data), "info", false)
					return err
				},
			} {
				t.Run(name, func(t *testing.T) {
					var output bytes.Buffer
					err := project(&output)
					if err == nil || !strings.Contains(err.Error(), "validate canonical report") {
						t.Fatalf("projection error = %v, want complete schema validation failure", err)
					}
				})
			}
		})
	}
}
