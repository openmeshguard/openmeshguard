package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadScanConfig(t *testing.T) {
	path := writeTestFile(t, "scan.yaml", `apiVersion: openmeshguard.io/v1alpha1
kind: ScanConfig
metadata: {name: platform, version: 0.1.0}
inputs:
  ownershipImport: ownership.yaml
  exceptions: [exceptions]
classification:
  sources:
    - type: namespace-label
      keys: [platform.example.com/environment]
    - type: namespace-name
      rules:
        - {pattern: '-prod$', environment: production}
    - type: cluster
      environment: staging
ownership:
  labels:
    appId: [platform.example.com/application-id]
    owner: [platform.example.com/team]
  applications:
    - {appId: checkout, owner: checkout-team}
controls:
  parameters:
    defaults: {approvedIstioVersions: ["1.30"]}
    environments:
      production: {approvedIstioVersions: ["1.30"]}
  overrides:
    - controlId: MG-MTLS-001
      environments: [production]
      severityByEnvironment: {production: critical}
`)
	config, err := LoadScanConfig(path)
	if err != nil {
		t.Fatalf("LoadScanConfig returned error: %v", err)
	}
	if config.Metadata.Name != "platform" || len(config.Classification.Sources) != 3 {
		t.Fatalf("scan config = %#v", config)
	}
	if config.Inputs.OwnershipImport != filepath.Join(filepath.Dir(path), "ownership.yaml") {
		t.Fatalf("ownership import = %q, want path relative to config", config.Inputs.OwnershipImport)
	}
	if len(config.Inputs.Exceptions) != 1 || config.Inputs.Exceptions[0] != filepath.Join(filepath.Dir(path), "exceptions") {
		t.Fatalf("exceptions = %#v, want path relative to config", config.Inputs.Exceptions)
	}
}

func TestLoadScanConfigRejectsInvalidSourcesAndOverrides(t *testing.T) {
	tests := []struct {
		name     string
		fragment string
		want     string
	}{
		{name: "unknown source", fragment: "    - type: magic\n", want: `type "magic" is unsupported`},
		{name: "label source without keys", fragment: "    - type: namespace-label\n", want: "requires only non-empty keys"},
		{name: "invalid name regex", fragment: "    - type: namespace-name\n      rules: [{pattern: '[', environment: production}]\n", want: "missing closing ]"},
		{name: "duplicate control override", fragment: "controls:\n  overrides:\n    - {controlId: MG-MTLS-001}\n    - {controlId: MG-MTLS-001}\n", want: "duplicate controlId"},
		{name: "invalid severity", fragment: "controls:\n  overrides:\n    - controlId: MG-MTLS-001\n      severityByEnvironment: {production: emergency}\n", want: `"emergency" is invalid`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestFile(t, "scan.yaml", `apiVersion: openmeshguard.io/v1alpha1
kind: ScanConfig
metadata: {name: platform, version: 0.1.0}
classification:
  sources:
`+tt.fragment)
			_, err := LoadScanConfig(path)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadScanConfig error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestLoadOwnershipImportYAMLAndCSV(t *testing.T) {
	yamlPath := writeTestFile(t, "ownership.yaml", `apiVersion: openmeshguard.io/v1alpha1
kind: OwnershipImport
metadata: {name: catalog, version: 0.1.0}
applications:
  - {appId: checkout, owner: checkout-team}
`)
	csvPath := writeTestFile(t, "ownership.csv", "appId,owner\npayments,payments-team\n")
	for _, path := range []string{yamlPath, csvPath} {
		ownership, err := LoadOwnershipImport(path)
		if err != nil {
			t.Fatalf("LoadOwnershipImport(%s) returned error: %v", path, err)
		}
		if len(ownership.Applications) != 1 || ownership.Applications[0].Owner == "" {
			t.Fatalf("ownership import = %#v", ownership)
		}
	}
}

func TestLoadExceptionsPreservesSemanticErrorsAndRejectsDuplicateIDs(t *testing.T) {
	directory := t.TempDir()
	valid := `apiVersion: openmeshguard.io/v1alpha1
kind: Exception
metadata: {name: EXC-42}
spec:
  controlIds: [MG-MTLS-001]
  owner: payments
  approver: security@example.com
  justification: Migration window
  ticket: https://tickets.example.com/SEC-42
  expiresAt: 2026-09-01T00:00:00Z
`
	if err := os.WriteFile(filepath.Join(directory, "valid.yaml"), []byte(valid), 0o600); err != nil {
		t.Fatalf("write valid exception: %v", err)
	}
	invalid := strings.ReplaceAll(valid, "EXC-42", "EXC-43")
	invalid = strings.ReplaceAll(invalid, "https://tickets.example.com/SEC-42", "")
	if err := os.WriteFile(filepath.Join(directory, "invalid.yml"), []byte(invalid), 0o600); err != nil {
		t.Fatalf("write invalid exception: %v", err)
	}
	records, err := LoadExceptions([]string{directory})
	if err != nil {
		t.Fatalf("LoadExceptions returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %d, want 2", len(records))
	}
	byID := map[string]ExceptionRecord{}
	for _, record := range records {
		byID[record.Metadata.Name] = record
	}
	if len(byID["EXC-42"].ValidationErrors) != 0 {
		t.Fatalf("valid exception errors = %#v", byID["EXC-42"].ValidationErrors)
	}
	if !containsText(byID["EXC-43"].ValidationErrors, "spec.ticket") {
		t.Fatalf("invalid exception errors = %#v, want ticket error", byID["EXC-43"].ValidationErrors)
	}

	duplicatePath := filepath.Join(directory, "duplicate.yaml")
	if err := os.WriteFile(duplicatePath, []byte(valid), 0o600); err != nil {
		t.Fatalf("write duplicate exception: %v", err)
	}
	if _, err := LoadExceptions([]string{directory}); err == nil || !strings.Contains(err.Error(), "duplicate exception ID") {
		t.Fatalf("duplicate exception error = %v", err)
	}
}

func writeTestFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func containsText(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}
