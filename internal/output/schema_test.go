package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestReportSchemaFixtures(t *testing.T) {
	root := filepath.Join("..", "..")
	schemaPath := filepath.Join(root, "docs", "contracts", "canonical-json-schema.json")

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile canonical schema: %v", err)
	}

	tests := []struct {
		name      string
		fixture   string
		wantValid bool
	}{
		{
			name:      "minimal report",
			fixture:   "minimal-report.json",
			wantValid: true,
		},
		{
			name:      "workload unknown report",
			fixture:   "workload-unknown-report.json",
			wantValid: true,
		},
		{
			name:      "unknown finding without reason",
			fixture:   "invalid-unknown-finding-missing-reason.json",
			wantValid: false,
		},
		{
			name:      "workload posture extra field",
			fixture:   "invalid-workload-posture-extra-field.json",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixturePath := filepath.Join(root, "test", "fixtures", "reports", tt.fixture)
			file, err := os.Open(fixturePath)
			if err != nil {
				t.Fatalf("open report fixture: %v", err)
			}
			defer file.Close()

			report, err := jsonschema.UnmarshalJSON(file)
			if err != nil {
				t.Fatalf("decode report fixture: %v", err)
			}

			err = schema.Validate(report)
			if tt.wantValid && err != nil {
				t.Fatalf("report fixture should match canonical schema: %v", err)
			}
			if !tt.wantValid && err == nil {
				t.Fatal("report fixture unexpectedly matched canonical schema")
			}
		})
	}
}
