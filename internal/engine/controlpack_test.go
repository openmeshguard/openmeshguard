package engine

import (
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	builtincontrols "github.com/openmeshguard/openmeshguard/controls"
)

func TestBuiltinControlPackEmbedMatchesValidationGlob(t *testing.T) {
	// This .yaml-only glob is intentionally identical to controls/embed.go.
	diskPaths, err := filepath.Glob(filepath.Join("..", "..", "controls", "*.yaml"))
	if err != nil {
		t.Fatalf("glob built-in control packs: %v", err)
	}
	var diskNames []string
	for _, path := range diskPaths {
		diskNames = append(diskNames, filepath.Base(path))
	}
	sort.Strings(diskNames)

	entries, err := fs.ReadDir(builtincontrols.BuiltinFS, ".")
	if err != nil {
		t.Fatalf("read embedded control packs: %v", err)
	}
	var embeddedNames []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			embeddedNames = append(embeddedNames, entry.Name())
		}
	}
	sort.Strings(embeddedNames)
	if !reflect.DeepEqual(embeddedNames, diskNames) {
		t.Fatalf("embedded packs = %v, disk packs validated by glob = %v", embeddedNames, diskNames)
	}

	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-in packs: %v", err)
	}
	if len(packs) != len(diskNames) {
		t.Fatalf("loaded packs = %d, want %d", len(packs), len(diskNames))
	}
}

func TestValidateFileRejections(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		contains []string
	}{
		{
			name:     "unknown fields",
			fixture:  "unknown-field.yaml",
			contains: []string{"unknown-field.yaml:", "control ACME-MTLS-001", `unknown field "typo"`},
		},
		{
			name:     "missing required fields",
			fixture:  "missing-required.yaml",
			contains: []string{"missing-required.yaml:", "control ACME-MTLS-001", `missing required field "expression"`},
		},
		{
			name:     "malformed identifier",
			fixture:  "malformed-id.yaml",
			contains: []string{"malformed-id.yaml:", "control bad-id", `id must match ^[A-Z]+-[A-Z]+-[0-9]{3}$`},
		},
		{
			name:     "CEL syntax includes compile position",
			fixture:  "cel-syntax.yaml",
			contains: []string{"cel-syntax.yaml:", "control ACME-MTLS-001", "expression CEL compile error at 1:"},
		},
		{
			name:     "out-of-scope variable",
			fixture:  "out-of-scope.yaml",
			contains: []string{"out-of-scope.yaml:", "control ACME-MTLS-001", "expression CEL compile error at 1:", "undeclared reference to 'resource'"},
		},
		{
			name:     "runtime control requires verified evidence",
			fixture:  "runtime-no-verified.yaml",
			contains: []string{"runtime-no-verified.yaml:", "control ACME-MTLS-101", "runtime evidenceType must require a verified.* field"},
		},
		{
			name:     "expression must return bool",
			fixture:  "non-bool.yaml",
			contains: []string{"non-bool.yaml:", "control ACME-MTLS-001", "expression CEL expression must return bool"},
		},
		{
			name:     "expression cannot bypass requires",
			fixture:  "requires-bypass.yaml",
			contains: []string{"requires-bypass.yaml:", "control ACME-MTLS-001", `expression path "workload.mtls.clientTLSContradiction" must be declared exactly in requires`},
		},
		{
			name:     "bracket expression cannot bypass requires",
			fixture:  "bracket-requires-bypass.yaml",
			contains: []string{"bracket-requires-bypass.yaml:", "control ACME-MTLS-001", `expression path "workload.mtls.clientTLSContradiction" must be declared exactly in requires`},
		},
		{
			name:     "parent requires cannot cover child",
			fixture:  "parent-requires-bypass.yaml",
			contains: []string{"parent-requires-bypass.yaml:", "control ACME-MTLS-001", `expression path "workload.mtls.effective" must be declared exactly in requires`},
		},
		{
			name:     "dynamic CEL result must be known boolean",
			fixture:  "dynamic-non-bool.yaml",
			contains: []string{"dynamic-non-bool.yaml:", "control ACME-GOV-001", "expression CEL expression must return bool, got dyn"},
		},
		{
			name:     "null contract headers",
			fixture:  "null-headers.yaml",
			contains: []string{"null-headers.yaml:", "control <pack>", `apiVersion must be "openmeshguard.io/v1alpha1"`, `kind must be "ControlPack"`},
		},
		{
			name:     "requires path must be in scope",
			fixture:  "scope-invalid-requires.yaml",
			contains: []string{"scope-invalid-requires.yaml:", "control ACME-GOV-001", `requires path "resource.kind" is not available to workload scope`},
		},
		{
			name:     "duplicate within pack",
			fixture:  "duplicate-in-pack.yaml",
			contains: []string{"duplicate-in-pack.yaml:", "control ACME-MTLS-001", "duplicate control ID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFile(filepath.Join("testdata", tt.fixture))
			if err == nil {
				t.Fatal("ValidateFile returned nil error")
			}
			for _, expected := range tt.contains {
				if !strings.Contains(err.Error(), expected) {
					t.Fatalf("error %q does not contain %q", err, expected)
				}
			}
		})
	}
}

func TestDeliberatelyMalformedPackReportsAllReviewGateDiagnostics(t *testing.T) {
	err := ValidateFile(filepath.Join("testdata", "malformed.yaml"))
	if err == nil {
		t.Fatal("ValidateFile returned nil error")
	}
	for _, expected := range []string{
		"malformed.yaml:",
		"control ACME-MTLS-001",
		`unknown field "unexpectedField"`,
		`missing required field "title"`,
		"runtime evidenceType must require a verified.* field",
		"applicability CEL compile error at 1:",
		"undeclared reference to 'resource'",
		"expression CEL compile error at 1:",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("malformed-pack error %q does not contain %q", err, expected)
		}
	}
}

func TestDuplicateControlIDAcrossBuiltinAndUserPack(t *testing.T) {
	_, err := LoadPacks([]string{filepath.Join("testdata", "duplicate-builtin.yaml")})
	if err == nil {
		t.Fatal("LoadPacks returned nil error")
	}
	for _, expected := range []string{"duplicate-builtin.yaml:", "control MG-MTLS-001", "duplicate control ID", "controls/builtin-mtls.yaml"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("duplicate error %q does not contain %q", err, expected)
		}
	}
}

func TestValidUserPackUsesStringsExtension(t *testing.T) {
	if err := ValidateFile(filepath.Join("testdata", "valid.yaml")); err != nil {
		t.Fatalf("ValidateFile returned error: %v", err)
	}
}

func TestCELVariablesAreScopedExactly(t *testing.T) {
	tests := []struct {
		name       string
		scope      string
		expression string
		wantError  string
	}{
		{
			name:       "workload exposes workload namespace inventory and params",
			scope:      "workload",
			expression: `workload.dataPlaneMode == "sidecar" && namespace.name == "payments" && inventory.ready == true && params.enabled == true`,
		},
		{
			name:  "workload rejects resource",
			scope: "workload", expression: `resource.kind == "Gateway"`, wantError: "undeclared reference to 'resource'",
		},
		{
			name:  "namespace exposes namespace",
			scope: "namespace", expression: `namespace.name == "payments"`,
		},
		{
			name:  "namespace rejects workload",
			scope: "namespace", expression: `workload.dataPlaneMode == "sidecar"`, wantError: "undeclared reference to 'workload'",
		},
		{
			name:  "resource exposes resource",
			scope: "resource", expression: `resource.kind == "Gateway"`,
		},
		{
			name:  "resource rejects namespace",
			scope: "resource", expression: `namespace.name == "payments"`, wantError: "undeclared reference to 'namespace'",
		},
		{
			name:  "internal namespace alias is not part of the contract",
			scope: "namespace", expression: `omg_nsctx.name == "payments"`, wantError: `undeclared reference to "omg_nsctx"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, issues := compileBoolean("scope-test.yaml", "ACME-TEST-001", "expression", nil, tt.scope, tt.expression)
			if tt.wantError == "" {
				if len(issues) > 0 {
					t.Fatalf("compileBoolean returned errors: %v", issues)
				}
				return
			}
			if len(issues) == 0 || !strings.Contains(issues.Error(), tt.wantError) {
				t.Fatalf("compileBoolean error = %v, want %q", issues, tt.wantError)
			}
		})
	}
}

func TestNamespaceCELRewritePreservesContractTextAndPositions(t *testing.T) {
	expression := `namespace.name == "namespace" && workload.workload . namespace == "payments"`
	rewritten := rewriteNamespaceVariable(expression)
	want := `omg_nsctx.name == "namespace" && workload.workload . namespace == "payments"`
	if rewritten != want {
		t.Fatalf("rewrite = %q, want %q", rewritten, want)
	}
	if len(rewritten) != len(expression) {
		t.Fatalf("rewrite length = %d, original = %d; CEL positions would drift", len(rewritten), len(expression))
	}
	if position := rootIdentifierPosition(`params.name == "omg_nsctx"`, namespaceCELVariable); position != -1 {
		t.Fatalf("internal alias inside string reported at %d", position)
	}
}

func TestCELDependencyAnalysisUsesTheCheckedAST(t *testing.T) {
	tests := []struct {
		name       string
		scope      string
		expression string
		wantPaths  []string
		wantError  string
	}{
		{
			name: "dot access", scope: "workload",
			expression: `workload.mtls.effective == "strict"`,
			wantPaths:  []string{"workload.mtls.effective"},
		},
		{
			name: "literal bracket access", scope: "workload",
			expression: `namespace["labels"]["team"] == "platform"`,
			wantPaths:  []string{"namespace.labels.team"},
		},
		{
			name: "path-shaped string is not evidence", scope: "workload",
			expression: `"workload.owner" == "workload.owner"`,
			wantPaths:  nil,
		},
		{
			name: "macro-bound index resolves to collection", scope: "workload",
			expression: `!workload.mtls.byPort.exists(port, workload.mtls.byPort[port] == "disabled")`,
			wantPaths:  []string{"workload.mtls.byPort"},
		},
		{
			name: "unbounded dynamic index is rejected", scope: "workload",
			expression: `workload[params.field] == true`,
			wantError:  "dynamic index into workload cannot be represented by a dotted requires path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, paths, issues := compileBoolean("dependencies.yaml", "ACME-TEST-001", "expression", nil, tt.scope, tt.expression)
			if tt.wantError != "" {
				if len(issues) == 0 || !strings.Contains(issues.Error(), tt.wantError) {
					t.Fatalf("compileBoolean error = %v, want %q", issues, tt.wantError)
				}
				return
			}
			if len(issues) > 0 {
				t.Fatalf("compileBoolean returned errors: %v", issues)
			}
			if !reflect.DeepEqual(paths, tt.wantPaths) {
				t.Fatalf("dependencies = %#v, want %#v", paths, tt.wantPaths)
			}
		})
	}
}

func TestBuiltinSuggestedYAMLTemplateIsEmbedded(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("LoadBuiltins returned error: %v", err)
	}
	control := packWithControl(t, packs, "MG-MTLS-001").Controls[0]
	if !strings.Contains(control.Remediation.SuggestedYAML, "kind: PeerAuthentication") {
		t.Fatalf("embedded suggested YAML = %q", control.Remediation.SuggestedYAML)
	}
}

func TestUnknownWorkloadIdentityFieldsAreNotTreatedAsStructural(t *testing.T) {
	_, err := decodeAndValidate("unknown-identity.yaml", []byte(`
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: unknown-identity, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Unknown identity field
    category: governance
    severity: medium
    evidenceType: context
    scope: workload
    requires: [workload.workload.name]
    applicability: 'true'
    expression: 'has(workload.workload.notAContractField)'
    message: Invalid identity field.
    remediation: {guidance: Use a contract field.}
`), SourceUser)
	if err == nil || !strings.Contains(err.Error(), `expression path "workload.workload.notAContractField" must be declared exactly in requires`) {
		t.Fatalf("decode error = %v, want unknown identity dependency rejection", err)
	}
}
