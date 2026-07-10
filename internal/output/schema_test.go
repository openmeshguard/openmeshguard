package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/normalize"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
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

func TestGeneratedScanOutputMatchesSchema(t *testing.T) {
	schema := compileSchemaForTest(t)
	resolved := resolver.New()
	workload := resolver.WorkloadInput{
		Ref: resolver.WorkloadRef{
			Cluster:   "cluster-a",
			Namespace: "payments",
			Name:      "api",
			Kind:      "Deployment",
		},
		DataPlaneMode: resolver.ModeSidecar,
		MeshDefaults: resolver.MeshDefaults{
			RootNamespace: "istio-system",
			Known:         true,
		},
		PeerAuthN: []resolver.PeerAuthenticationView{{
			Name:      "default",
			Namespace: "payments",
			Mode:      "PERMISSIVE",
		}},
	}

	var output bytes.Buffer
	err := WriteScanJSON(&output, ScanInput{
		GeneratedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ScannerVersion:  "dev",
		ResolverVersion: resolved.Version(),
		ClusterContext:  "fixture",
		Scope: ScanScope{
			AllNamespaces: true,
		},
		PermissionSummary: []collect.Permission{{
			APIGroup: "security.istio.io",
			Resource: "peerauthentications",
			Verbs:    []string{"list"},
			Granted:  true,
		}},
		Inventory: normalize.Inventory{
			Counts:        map[string]int{"deployments": 1},
			DataPlaneMode: resolver.ModeSidecar,
			MultiCluster:  normalize.MultiCluster{},
		},
		WorkloadPostures: []resolver.WorkloadResult{{
			Ref:   workload.Ref,
			Mode:  workload.DataPlaneMode,
			MTLS:  resolved.ResolveMTLS(workload),
			Authz: resolved.ResolveAuthz(workload),
		}},
	})
	if err != nil {
		t.Fatalf("write generated scan output: %v", err)
	}

	report, err := jsonschema.UnmarshalJSON(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("decode generated report: %v", err)
	}
	if err := schema.Validate(report); err != nil {
		t.Fatalf("generated report should match canonical schema: %v\n%s", err, output.String())
	}
}

func TestExternalScanOutputMatchesSchema(t *testing.T) {
	reportPath := os.Getenv("OPENMESHGUARD_SCHEMA_REPORT")
	if reportPath == "" {
		t.Skip("set OPENMESHGUARD_SCHEMA_REPORT to validate a captured scan output")
	}

	schema := compileSchemaForTest(t)
	file, err := os.Open(reportPath)
	if err != nil {
		t.Fatalf("open external report: %v", err)
	}
	defer file.Close()

	report, err := jsonschema.UnmarshalJSON(file)
	if err != nil {
		t.Fatalf("decode external report: %v", err)
	}
	if err := schema.Validate(report); err != nil {
		t.Fatalf("external report should match canonical schema: %v", err)
	}
}

func compileSchemaForTest(t *testing.T) *jsonschema.Schema {
	t.Helper()

	schemaPath := filepath.Join("..", "..", "docs", "contracts", "canonical-json-schema.json")
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("compile canonical schema: %v", err)
	}
	return schema
}
