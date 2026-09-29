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

func TestCanonicalProjectionInputRejectsInvalidFormatsAndEncoding(t *testing.T) {
	valid := thresholdReportJSON(t, []finding{thresholdFinding("open", "high")})
	invalidGeneratedAt := bytes.Replace(
		valid,
		[]byte(`"generatedAt":"2026-07-25T00:00:00Z"`),
		[]byte(`"generatedAt":"not-a-date"`),
		1,
	)
	if bytes.Equal(invalidGeneratedAt, valid) {
		t.Fatal("valid report did not contain generatedAt fixture")
	}

	withException := thresholdFinding("excepted", "high")
	withException.Exception = &exceptionEvidence{
		ID:        "EXC-1",
		ExpiresAt: "not-a-date",
	}
	invalidExceptionDate := thresholdReportJSON(t, []finding{withException})

	invalidUTF8 := bytes.Replace(
		valid,
		[]byte(`"clusterContext":"fixture"`),
		append([]byte(`"clusterContext":"fi`), append([]byte{0xff}, []byte(`xture"`)...)...),
		1,
	)
	if bytes.Equal(invalidUTF8, valid) {
		t.Fatal("valid report did not contain clusterContext fixture")
	}

	tests := []struct {
		name      string
		data      []byte
		wantError string
	}{
		{name: "generatedAt format", data: invalidGeneratedAt, wantError: "validate canonical report"},
		{name: "exception expiresAt format", data: invalidExceptionDate, wantError: "validate canonical report"},
		{name: "invalid UTF-8", data: invalidUTF8, wantError: "valid UTF-8"},
	}
	for _, tt := range tests {
		for name, project := range map[string]func(*bytes.Buffer) error{
			"html": func(output *bytes.Buffer) error {
				return WriteHTML(output, bytes.NewReader(tt.data))
			},
			"sarif": func(output *bytes.Buffer) error {
				return WriteSARIF(output, bytes.NewReader(tt.data))
			},
			"score": func(output *bytes.Buffer) error {
				return WriteScore(output, bytes.NewReader(tt.data), "")
			},
			"threshold": func(_ *bytes.Buffer) error {
				_, err := FailsThreshold(bytes.NewReader(tt.data), "info", false)
				return err
			},
		} {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				var output bytes.Buffer
				err := project(&output)
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("projection error = %v, want %q", err, tt.wantError)
				}
			})
		}
	}
}

func TestCanonicalProjectionRejectsNegativeCountersAcrossAllConsumers(t *testing.T) {
	valid := thresholdReportJSON(t, []finding{thresholdFinding("open", "high")})
	for _, tt := range negativeCanonicalCounterMutations() {
		t.Run(tt.name, func(t *testing.T) {
			data := mutateCanonicalDocument(t, valid, tt.mutate)
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
						t.Fatalf("projection error = %v, want negative-counter schema failure", err)
					}
					if output.Len() != 0 {
						t.Fatalf("projection emitted %d bytes before rejecting negative counter", output.Len())
					}
				})
			}
		})
	}
}

type canonicalMutation struct {
	name   string
	mutate func(map[string]any)
}

func negativeCanonicalCounterMutations() []canonicalMutation {
	return []canonicalMutation{
		{
			name: "inventory resource count",
			mutate: func(report map[string]any) {
				report["inventory"].(map[string]any)["counts"].(map[string]any)["pods"] = float64(-1)
			},
		},
		{
			name: "ztunnel nodes covered",
			mutate: func(report map[string]any) {
				report["inventory"].(map[string]any)["dataPlane"].(map[string]any)["ztunnel"] =
					map[string]any{"nodesCovered": float64(-1)}
			},
		},
		{
			name: "ztunnel nodes total",
			mutate: func(report map[string]any) {
				report["inventory"].(map[string]any)["dataPlane"].(map[string]any)["ztunnel"] =
					map[string]any{"nodesTotal": float64(-1)}
			},
		},
		{
			name: "waypoint count",
			mutate: func(report map[string]any) {
				report["inventory"].(map[string]any)["dataPlane"].(map[string]any)["waypoints"] = float64(-1)
			},
		},
		{
			name: "classified namespace count",
			mutate: func(report map[string]any) {
				report["inventory"].(map[string]any)["classification"] =
					map[string]any{"namespacesClassified": float64(-1)}
			},
		},
		{
			name: "unclassified namespace count",
			mutate: func(report map[string]any) {
				report["inventory"].(map[string]any)["classification"] =
					map[string]any{"namespacesUnclassified": float64(-1)}
			},
		},
		{
			name: "environment namespace count",
			mutate: func(report map[string]any) {
				report["inventory"].(map[string]any)["classification"] =
					map[string]any{"byEnvironment": map[string]any{"production": float64(-1)}}
			},
		},
		{
			name: "evaluated score count",
			mutate: func(report map[string]any) {
				report["scores"].(map[string]any)["categories"] = []any{
					map[string]any{
						"category":  "mtls",
						"grade":     "unknown",
						"passRate":  nil,
						"evaluated": float64(-1),
					},
				}
			},
		},
		{
			name: "unknown score count",
			mutate: func(report map[string]any) {
				report["scores"].(map[string]any)["categories"] = []any{
					map[string]any{
						"category": "mtls",
						"grade":    "unknown",
						"passRate": nil,
						"unknown":  float64(-1),
					},
				}
			},
		},
	}
}

func mutateCanonicalDocument(
	t *testing.T,
	valid []byte,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(valid, &document); err != nil {
		t.Fatalf("decode valid canonical report: %v", err)
	}
	mutate(document)
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode mutated canonical report: %v", err)
	}
	return data
}
