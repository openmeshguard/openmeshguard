package engine

import (
	"io/fs"
	"os"
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
			name:     "nested dynamic CEL result is not a boolean",
			fixture:  "dynamic-nested-non-bool.yaml",
			contains: []string{"dynamic-nested-non-bool.yaml:", "control ACME-MTLS-001", "expression CEL expression must return bool, got dyn"},
		},
		{
			name:     "CEL lexer errors are rejected",
			fixture:  "cel-lexer-error.yaml",
			contains: []string{"cel-lexer-error.yaml:", "control ACME-MTLS-001", "expression CEL compile error at 1:25", "token recognition error"},
		},
		{
			name:     "message selector must exist",
			fixture:  "template-unknown-field.yaml",
			contains: []string{"template-unknown-field.yaml:", "control ACME-GOV-001", "message template is invalid", "field Worklod does not exist"},
		},
		{
			name:     "message variable selector must exist",
			fixture:  "template-variable-unknown-field.yaml",
			contains: []string{"template-variable-unknown-field.yaml:", "control ACME-GOV-001", "message template is invalid", "field Mtlls does not exist"},
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
			name:     "requires bracket key must be a literal",
			fixture:  "requires-dynamic-key.yaml",
			contains: []string{"requires-dynamic-key.yaml:", "control ACME-GOV-001", "must use dotted fields and optional literal bracket keys"},
		},
		{
			name:     "resource match requires API groups",
			fixture:  "resource-missing-api-groups.yaml",
			contains: []string{"resource-missing-api-groups.yaml:", "control ACME-GW-001", `missing required field "apiGroups"`},
		},
		{
			name:     "resource match API groups must not be empty",
			fixture:  "resource-empty-api-groups.yaml",
			contains: []string{"resource-empty-api-groups.yaml:", "control ACME-GW-001", "match.apiGroups must contain at least one value"},
		},
		{
			name:     "resource match API groups exclude versions",
			fixture:  "resource-version-in-api-groups.yaml",
			contains: []string{"resource-version-in-api-groups.yaml:", "control ACME-GW-001", `match.apiGroups value "gateway.networking.k8s.io/v1" must be a Kubernetes API group without a version`},
		},
		{
			name:     "match is resource scope only",
			fixture:  "match-on-workload.yaml",
			contains: []string{"match-on-workload.yaml:", "control ACME-GOV-001", "match is only valid for resource scope"},
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

func TestResourceMatchAllowsExplicitCoreAPIGroup(t *testing.T) {
	_, err := decodeAndValidate("core-resource.yaml", []byte(`
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: core-resource, version: 1.0.0}
controls:
  - id: ACME-CORE-001
    title: ConfigMaps must be named
    category: governance
    severity: low
    evidenceType: config
    scope: resource
    match:
      apiGroups: [""]
      kinds: [ConfigMap]
    requires: [resource.name]
    applicability: 'true'
    expression: 'resource.name != ""'
    message: ConfigMap must be named.
    remediation: {guidance: Set metadata.name.}
`), SourceUser)
	if err != nil {
		t.Fatalf("decode core API resource control: %v", err)
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
	rawExpression := "R\"\"\"a\" namespace\"\"\" == R\"\"\"a\" namespace\"\"\" && // namespace in comment\nnamespace.name == \"payments\""
	rawWant := "R\"\"\"a\" namespace\"\"\" == R\"\"\"a\" namespace\"\"\" && // namespace in comment\nomg_nsctx.name == \"payments\""
	if got := rewriteNamespaceVariable(rawExpression); got != rawWant {
		t.Fatalf("raw/comment rewrite = %q, want %q", got, rawWant)
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
			name: "native label key preserves bracket boundary", scope: "workload",
			expression: `namespace.labels["app.kubernetes.io/name"] == "api"`,
			wantPaths:  []string{`namespace.labels["app.kubernetes.io/name"]`},
		},
		{
			name: "independently read parent is retained", scope: "workload",
			expression: `inventory.counts.exists(key, inventory.counts[key] == 0) && inventory.counts.pods > 0`,
			wantPaths:  []string{"inventory.counts", "inventory.counts.pods"},
		},
		{
			name: "comprehension variable may shadow contract root", scope: "workload",
			expression: `workload.mtls.chain.exists(workload, workload.kind == "PeerAuthentication")`,
			wantPaths:  []string{"workload.mtls.chain"},
		},
		{
			name: "unbounded dynamic index is rejected", scope: "workload",
			expression: `workload[params.field] == true`,
			wantError:  "dynamic index into workload cannot be represented by a dotted requires path",
		},
		{
			name: "root map macro is rejected", scope: "workload",
			expression: `workload.exists(key, key == "owner")`,
			wantError:  "dynamic access to workload cannot be represented by an exact dotted requires path",
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

func TestEvidencePathRoundTrip(t *testing.T) {
	tests := []struct {
		path     string
		segments []string
		want     string
	}{
		{path: "namespace.labels.team", segments: []string{"namespace", "labels", "team"}, want: "namespace.labels.team"},
		{path: `namespace.labels["app.kubernetes.io/name"]`, segments: []string{"namespace", "labels", "app.kubernetes.io/name"}, want: `namespace.labels["app.kubernetes.io/name"]`},
		{path: `params["owner.team"]`, segments: []string{"params", "owner.team"}, want: `params["owner.team"]`},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			segments, err := parseEvidencePath(tt.path)
			if err != nil {
				t.Fatalf("parseEvidencePath returned error: %v", err)
			}
			if !reflect.DeepEqual(segments, tt.segments) {
				t.Fatalf("segments = %#v, want %#v", segments, tt.segments)
			}
			if got := formatEvidencePath(segments); got != tt.want {
				t.Fatalf("formatEvidencePath = %q, want %q", got, tt.want)
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

func TestBuiltinMTLSCatalogMetadata(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("LoadBuiltins returned error: %v", err)
	}

	wantFrameworks := []string{
		"nist-csf-2.0/PR.DS-02",
		"owasp-k8s-2025/K05",
	}
	tests := []struct {
		id    string
		title string
	}{
		{id: "MG-MTLS-001", title: "Mesh-managed workloads must resolve to strict mTLS"},
		{id: "MG-MTLS-002", title: "Declared workload ports must resolve to strict mTLS"},
		{id: "MG-MTLS-003", title: "Workloads must never resolve to globally disabled mTLS"},
		{id: "MG-MTLS-005", title: "Ambient workloads must have validated L4 mTLS posture"},
		{id: "MG-MTLS-006", title: "Ambient workloads must have healthy ztunnel node coverage"},
		{id: "MG-MTLS-007", title: "Client TLS must agree with resolved server mTLS"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			control := packWithControl(t, packs, tt.id).Controls[0]
			if control.Title != tt.title {
				t.Fatalf("title = %q, want %q", control.Title, tt.title)
			}
			if !reflect.DeepEqual(control.Frameworks, wantFrameworks) {
				t.Fatalf("frameworks = %#v, want %#v", control.Frameworks, wantFrameworks)
			}
		})
	}

}

func TestBuiltinAuthorizationCatalogMetadata(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("LoadBuiltins returned error: %v", err)
	}
	accessEnforcement := []string{
		"nist-csf-2.0/PR.AA-05",
		"nist-sp-800-53-r5/AC-3",
		"owasp-k8s-2025/K05",
	}
	leastPrivilege := []string{
		"nist-csf-2.0/PR.AA-05",
		"nist-sp-800-53-r5/AC-3",
		"nist-sp-800-53-r5/AC-6",
		"owasp-k8s-2025/K05",
	}
	tests := []struct {
		id         string
		title      string
		frameworks []string
	}{
		{id: "MG-AUTHZ-001", title: "Mesh-managed workloads must have resolved authorization coverage", frameworks: accessEnforcement},
		{id: "MG-AUTHZ-002", title: "Workloads should resolve to default deny with explicit allow", frameworks: leastPrivilege},
		{id: "MG-AUTHZ-003", title: "Authorization policies must not grant structurally broad access", frameworks: leastPrivilege},
		{id: "MG-AUTHZ-004", title: "Authorization access must be scoped to explicit identities", frameworks: leastPrivilege},
		{id: "MG-AUTHZ-005", title: "Authorization coverage must resolve at workload level", frameworks: accessEnforcement},
		{id: "MG-AUTHZ-006", title: "Waypoint-attached authorization must have a ready enforcement path", frameworks: accessEnforcement},
		{id: "MG-AUTHZ-007", title: "Unenforced waypoint authorization must be reported", frameworks: accessEnforcement},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			control := packWithControl(t, packs, tt.id).Controls[0]
			if control.Title != tt.title || control.Category != "authz" || control.EvidenceType != "config" || control.Scope != "workload" {
				t.Fatalf("control = %#v, want authz config workload control titled %q", control, tt.title)
			}
			if !reflect.DeepEqual(control.Frameworks, tt.frameworks) {
				t.Fatalf("frameworks = %#v, want %#v", control.Frameworks, tt.frameworks)
			}
		})
	}

	gateway := packWithControl(t, packs, "MG-GW-005").Controls[0]
	if gateway.Title != "Ambient waypoint authorization must have explicit ready enrollment" ||
		gateway.Category != "exposure" ||
		gateway.EvidenceType != "config" ||
		gateway.Scope != "workload" {
		t.Fatalf("MG-GW-005 metadata = %#v", gateway)
	}
	if !reflect.DeepEqual(gateway.Frameworks, accessEnforcement) {
		t.Fatalf("MG-GW-005 frameworks = %#v, want %#v", gateway.Frameworks, accessEnforcement)
	}
}

func TestUserSuggestedYAMLTemplateValidation(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		symlinkEscape bool
		wantError     string
	}{
		{name: "static selector typo", template: "namespace: {{ .Namespce }}", wantError: "field Namespce does not exist"},
		{name: "symlink escape", symlinkEscape: true, wantError: "open template within control pack directory"},
		{name: "static selector through variable", template: "{{$p := .Posture}}mode: {{$p.Mtls.Effective}}"},
		{name: "dynamic params selector", template: "owner: {{ .Params.owner }}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			templatePath := filepath.Join(directory, "remediation.tmpl")
			if tt.symlinkEscape {
				outside := filepath.Join(t.TempDir(), "outside.tmpl")
				if err := os.WriteFile(outside, []byte("secret material"), 0o600); err != nil {
					t.Fatalf("write outside template: %v", err)
				}
				if err := os.Symlink(outside, templatePath); err != nil {
					t.Skipf("create template symlink: %v", err)
				}
			} else if err := os.WriteFile(templatePath, []byte(tt.template), 0o600); err != nil {
				t.Fatalf("write template: %v", err)
			}

			packPath := filepath.Join(directory, "pack.yaml")
			pack := `apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: template-validation, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Template validation
    category: governance
    severity: medium
    evidenceType: context
    scope: namespace
    requires: [namespace.name]
    applicability: 'true'
    expression: 'namespace.name != ""'
    message: 'Namespace {{ .Namespace }} failed.'
    remediation:
      guidance: Correct the namespace.
      suggestedYAMLTemplate: remediation.tmpl
`
			if err := os.WriteFile(packPath, []byte(pack), 0o600); err != nil {
				t.Fatalf("write pack: %v", err)
			}
			err := ValidateFile(packPath)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("ValidateFile returned error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ValidateFile error = %v, want %q", err, tt.wantError)
			}
		})
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
