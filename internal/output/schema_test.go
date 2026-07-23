package output

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/engine"
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

	goldens, err := filepath.Glob(filepath.Join(root, "test", "fixtures", "sidecar-basic", "golden", "*.json"))
	if err != nil {
		t.Fatalf("glob M4 goldens: %v", err)
	}
	if len(goldens) == 0 {
		t.Fatal("no M4 canonical JSON goldens discovered")
	}
	for _, golden := range goldens {
		golden := golden
		t.Run("M4 golden "+filepath.Base(golden), func(t *testing.T) {
			file, err := os.Open(golden)
			if err != nil {
				t.Fatalf("open M4 golden: %v", err)
			}
			defer file.Close()

			report, err := jsonschema.UnmarshalJSON(file)
			if err != nil {
				t.Fatalf("decode M4 golden: %v", err)
			}
			if err := schema.Validate(report); err != nil {
				t.Fatalf("M4 golden should match canonical schema: %v", err)
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
		}, {
			Ref: resolver.WorkloadRef{
				Cluster: "cluster-a", Namespace: "legacy", Name: "outside-mesh", Kind: "Deployment",
			},
			Mode: resolver.ModeNotApplicable,
			MTLS: resolver.MTLSResult{
				Effective: resolver.MTLSNotInMesh,
				Chain:     []resolver.Step{{Order: 1, Kind: "DataPlane", Effect: "workload is not enrolled"}},
			},
			Authz: resolved.ResolveAuthz(resolver.WorkloadInput{}),
		}},
	})
	if err != nil {
		t.Fatalf("write generated scan output: %v", err)
	}

	rawReport, err := jsonschema.UnmarshalJSON(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatalf("decode generated report: %v", err)
	}
	if err := schema.Validate(rawReport); err != nil {
		t.Fatalf("generated report should match canonical schema: %v\n%s", err, output.String())
	}

	var generated report
	if err := json.Unmarshal(output.Bytes(), &generated); err != nil {
		t.Fatalf("decode generated report into output model: %v", err)
	}
	if len(generated.Findings) == 0 {
		t.Fatal("generated report has no engine findings")
	}
	seenUnknown := false
	seenNotApplicable := false
	seenSuggestedYAML := false
	for _, finding := range generated.Findings {
		if !strings.HasPrefix(finding.ID, finding.ControlID+"-") {
			t.Fatalf("finding ID %q does not use engine control prefix %q", finding.ID, finding.ControlID)
		}
		switch finding.Status {
		case "unknown":
			seenUnknown = true
			if finding.UnknownReason == "" {
				t.Fatalf("unknown finding %s missing unknownReason", finding.ID)
			}
		case "not-applicable":
			seenNotApplicable = true
		}
		if finding.ControlID == "MG-MTLS-001" && finding.Remediation != nil && strings.Contains(finding.Remediation.SuggestedYAML, "kind: PeerAuthentication") {
			seenSuggestedYAML = true
		}
	}
	if !seenUnknown || !seenNotApplicable {
		t.Fatalf("generated findings missing required shapes: unknown=%t not-applicable=%t", seenUnknown, seenNotApplicable)
	}
	if !seenSuggestedYAML {
		t.Fatal("generated findings missing rendered suggestedYAML remediation")
	}
	if len(generated.Scores.Categories) != 3 {
		t.Fatalf("score categories = %#v, want authorization, exposure, and mTLS categories", generated.Scores.Categories)
	}
	authzCategory := generated.Scores.Categories[0]
	if authzCategory.Category != "authz" || authzCategory.Grade != "unknown" || authzCategory.PassRate != nil || authzCategory.Unknown != 7 {
		t.Fatalf("generated authorization category = %#v, want seven unknown evaluations", authzCategory)
	}
	exposureCategory := generated.Scores.Categories[1]
	if exposureCategory.Category != "exposure" || exposureCategory.Grade != "unknown" || exposureCategory.PassRate != nil {
		t.Fatalf("generated exposure category = %#v, want no applicable evaluations", exposureCategory)
	}
	mtlsCategory := generated.Scores.Categories[2]
	if mtlsCategory.Category != "mtls" || mtlsCategory.Grade != "F" || mtlsCategory.PassRate == nil || *mtlsCategory.PassRate != 0.5 {
		t.Fatalf("generated mTLS category = %#v, want F grade at 50%% pass rate", mtlsCategory)
	}
}

func TestClientTLSContradictionOutputTracksDestinationRuleAvailability(t *testing.T) {
	knownFalse := false
	workloads := []resolver.WorkloadResult{
		{
			Ref:   resolver.WorkloadRef{Namespace: "payments", Name: "unavailable", Kind: "Deployment"},
			Mode:  resolver.ModeSidecar,
			MTLS:  resolver.MTLSResult{Effective: resolver.MTLSStrict, Chain: []resolver.Step{}},
			Authz: resolver.AuthzResult{Effective: resolver.AuthzUnknown, Chain: []resolver.Step{}, UnknownReason: "not implemented"},
		},
		{
			Ref:   resolver.WorkloadRef{Namespace: "payments", Name: "collected", Kind: "Deployment"},
			Mode:  resolver.ModeSidecar,
			MTLS:  resolver.MTLSResult{Effective: resolver.MTLSStrict, ClientTLSContradiction: &knownFalse, Chain: []resolver.Step{}},
			Authz: resolver.AuthzResult{Effective: resolver.AuthzUnknown, Chain: []resolver.Step{}, UnknownReason: "not implemented"},
		},
	}
	reportJSON, err := json.Marshal(buildReport(ScanInput{
		ScannerVersion: "dev", ResolverVersion: resolver.New().Version(), ClusterContext: "fixture",
		Scope: ScanScope{AllNamespaces: true}, Inventory: normalize.Inventory{Counts: map[string]int{}}, WorkloadPostures: workloads,
	}, nil, engine.Result{}))
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(reportJSON, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	postures := decoded["workloadPostures"].([]any)
	unavailableMTLS := postures[0].(map[string]any)["mtls"].(map[string]any)
	if _, exists := unavailableMTLS["clientTLSContradiction"]; exists {
		t.Fatalf("unavailable DestinationRule evidence emitted clientTLSContradiction: %#v", unavailableMTLS)
	}
	collectedMTLS := postures[1].(map[string]any)["mtls"].(map[string]any)
	if value, exists := collectedMTLS["clientTLSContradiction"]; !exists || value != false {
		t.Fatalf("collected DestinationRule evidence = %#v, want explicit false contradiction", collectedMTLS)
	}
	if err := compileSchemaForTest(t).Validate(decoded); err != nil {
		t.Fatalf("availability-shaped report should match canonical schema: %v", err)
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
