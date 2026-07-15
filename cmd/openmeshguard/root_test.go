package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVersionCommandPrintsScannerAndResolverVersions(t *testing.T) {
	info := defaultVersionInfo()
	stdout, stderr, err := executeForTest(t, versionInfo{
		Version:         "test-version",
		ResolverVersion: info.ResolverVersion,
	}, "version")

	if err != nil {
		t.Fatalf("version command returned error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("version command wrote stderr %q", stderr)
	}
	if !strings.Contains(stdout, "version=test-version") {
		t.Fatalf("version output missing scanner version: %q", stdout)
	}
	if !strings.Contains(stdout, "resolverVersion="+info.ResolverVersion) {
		t.Fatalf("version output missing resolver version: %q", stdout)
	}
}

func TestStubCommandsReturnNotImplementedExitCode(t *testing.T) {
	for _, name := range []string{"report", "export", "score"} {
		t.Run(name, func(t *testing.T) {
			_, _, err := executeForTest(t, defaultVersionInfo(), name)
			if !errors.Is(err, errNotImplemented) {
				t.Fatalf("%s returned %v, want errNotImplemented", name, err)
			}
			if got := exitCode(err); got != 2 {
				t.Fatalf("%s exit code = %d, want 2", name, got)
			}
		})
	}
}

func TestScanRequiresExplicitScope(t *testing.T) {
	_, _, err := executeForTest(t, defaultVersionInfo(), "scan")
	if err == nil {
		t.Fatal("scan without scope returned nil error")
	}
	if !strings.Contains(err.Error(), "scan scope required") {
		t.Fatalf("scan error = %v, want scope validation", err)
	}
}

func TestScanRejectsEmptyNamespace(t *testing.T) {
	_, _, err := executeForTest(t, defaultVersionInfo(), "scan", "--namespace", "")
	if err == nil {
		t.Fatal("scan with empty namespace returned nil error")
	}
	if !strings.Contains(err.Error(), "namespace must not be empty") {
		t.Fatalf("scan error = %v, want empty namespace validation", err)
	}
}

func TestScanRootNamespaceFlag(t *testing.T) {
	cmd := newScanCommand(defaultVersionInfo())
	flag := cmd.Flags().Lookup("root-namespace")
	if flag == nil {
		t.Fatal("scan command missing root-namespace flag")
	}
	if flag.DefValue != collect.DefaultRootNamespace {
		t.Fatalf("root-namespace default = %q, want %q", flag.DefValue, collect.DefaultRootNamespace)
	}

	opts := scanOptions{AllNamespaces: true, RootNamespace: "  "}
	if err := opts.normalizeAndValidate(); err == nil || !strings.Contains(err.Error(), "root namespace must not be empty") {
		t.Fatalf("empty root namespace validation error = %v, want root namespace error", err)
	}
}

func TestScanControlPackFlagIsRepeatable(t *testing.T) {
	cmd := newScanCommand(defaultVersionInfo())
	flag := cmd.Flags().Lookup("control-pack")
	if flag == nil {
		t.Fatal("scan command missing control-pack flag")
	}
	if flag.Value.Type() != "stringArray" {
		t.Fatalf("control-pack flag type = %q, want stringArray", flag.Value.Type())
	}

	opts := scanOptions{AllNamespaces: true, RootNamespace: collect.DefaultRootNamespace, ControlPacks: []string{"  "}}
	if err := opts.normalizeAndValidate(); err == nil || !strings.Contains(err.Error(), "control pack path must not be empty") {
		t.Fatalf("empty control pack validation error = %v, want control pack path error", err)
	}
}

func TestControlsValidateCommand(t *testing.T) {
	validPath := "../../internal/engine/testdata/valid.yaml"
	stdout, stderr, err := executeForTest(t, defaultVersionInfo(), "controls", "validate", validPath)
	if err != nil {
		t.Fatalf("controls validate returned error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("controls validate wrote stderr %q", stderr)
	}
	if !strings.Contains(stdout, "valid control pack: "+validPath) {
		t.Fatalf("controls validate output = %q", stdout)
	}

	malformedPath := "../../internal/engine/testdata/malformed.yaml"
	_, _, err = executeForTest(t, defaultVersionInfo(), "controls", "validate", malformedPath)
	if err == nil {
		t.Fatal("controls validate accepted malformed pack")
	}
	for _, expected := range []string{"malformed.yaml:", "control ACME-MTLS-001", "CEL compile error at 1:"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("malformed controls validate error %q does not contain %q", err, expected)
		}
	}

	duplicatePath := "../../internal/engine/testdata/duplicate-builtin.yaml"
	_, _, err = executeForTest(t, defaultVersionInfo(), "controls", "validate", duplicatePath)
	if err == nil || !strings.Contains(err.Error(), "duplicate control ID") {
		t.Fatalf("controls validate duplicate error = %v, want collision with built-ins", err)
	}
}

func TestNamespaceInputsIncludeNamespacesWithoutWorkloads(t *testing.T) {
	snapshot := collect.Snapshot{Namespaces: []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "empty", Labels: map[string]string{"team": "platform", "istio-injection": "enabled"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "payments", Labels: map[string]string{"istio-injection": "enabled"}}},
	}}
	workloads := []resolver.WorkloadInput{{
		Ref:       resolver.WorkloadRef{Namespace: "payments", Name: "api"},
		Namespace: resolver.NamespaceInput{Name: "payments", Labels: map[string]string{"istio-injection": "enabled"}},
	}}

	got := namespaceInputs(snapshot, workloads, nil)
	if len(got) != 2 || got[0].Name != "empty" || got[1].Name != "payments" {
		t.Fatalf("namespace inputs = %#v, want empty and workload namespaces", got)
	}
	if got[0].Labels["team"] != "platform" || got[1].MeshEnrollment != "enrolled" {
		t.Fatalf("namespace inputs lost labels or resolver enrollment: %#v", got)
	}
}

func TestMeshNamespaceInputsExcludeOnlyKnownNonMeshNamespaces(t *testing.T) {
	got := meshNamespaceInputs([]engine.NamespaceInput{
		{Name: "mesh", MeshEnrollment: "enrolled"},
		{Name: "outside", MeshEnrollment: "not-enrolled"},
		{Name: "uncertain", MeshEnrollment: "unknown"},
		{Name: "unobserved"},
	})
	if len(got) != 3 || got[0].Name != "mesh" || got[1].Name != "uncertain" || got[2].Name != "unobserved" {
		t.Fatalf("mesh namespace inputs = %#v, want enrolled and unknown namespaces only", got)
	}
}

func TestNamespaceInputsAggregateMeshEnrollment(t *testing.T) {
	type workloadObservation struct {
		ambient resolver.Tristate
		mode    resolver.DataPlaneMode
	}
	tests := []struct {
		name      string
		labels    map[string]string
		workloads []workloadObservation
		want      string
	}{
		{name: "sidecar label survives unobserved workload", labels: map[string]string{"istio-injection": "enabled"}, workloads: []workloadObservation{{ambient: resolver.Unobserved, mode: resolver.ModeUnknown}}, want: "enrolled"},
		{name: "ambient label survives unobserved workload", labels: map[string]string{"istio.io/dataplane-mode": "ambient"}, workloads: []workloadObservation{{ambient: resolver.Unobserved, mode: resolver.ModeUnknown}}, want: "enrolled"},
		{name: "ambient observation refines unlabeled namespace", workloads: []workloadObservation{{ambient: resolver.True, mode: resolver.ModeUnknown}}, want: "enrolled"},
		{name: "observed sidecar refines unlabeled namespace", workloads: []workloadObservation{{mode: resolver.ModeSidecar}}, want: "enrolled"},
		{name: "positive observation wins regardless of workload order", workloads: []workloadObservation{{ambient: resolver.True, mode: resolver.ModeUnknown}, {ambient: resolver.False, mode: resolver.ModeUnknown}}, want: "enrolled"},
		{name: "not in mesh observation refines unlabeled namespace", workloads: []workloadObservation{{mode: resolver.ModeNotApplicable}}, want: "not-enrolled"},
		{name: "unobserved workload-only namespace stays unknown", workloads: []workloadObservation{{ambient: resolver.Unobserved, mode: resolver.ModeUnknown}}, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := collect.Snapshot{Namespaces: []corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "payments", Labels: tt.labels}}}}
			if tt.labels == nil {
				snapshot.Namespaces = nil
			}
			workloads := make([]resolver.WorkloadInput, 0, len(tt.workloads))
			for index, observation := range tt.workloads {
				workloads = append(workloads, resolver.WorkloadInput{
					Ref:           resolver.WorkloadRef{Namespace: "payments", Name: fmt.Sprintf("workload-%d", index)},
					DataPlaneMode: observation.mode,
					Namespace:     resolver.NamespaceInput{Name: "payments", AmbientEnrolled: observation.ambient},
				})
			}
			got := namespaceInputs(snapshot, workloads, nil)
			if len(got) != 1 || got[0].MeshEnrollment != tt.want {
				t.Fatalf("namespace inputs = %#v, want meshEnrollment %q", got, tt.want)
			}
		})
	}
}

func TestInventoryAvailabilityFromPermissionSummary(t *testing.T) {
	tests := []struct {
		name       string
		permission collect.Permission
		wantPaths  []string
		rejectPath string
	}{
		{
			name:       "services affect count and multi-cluster evidence",
			permission: collect.Permission{Resource: "services", Granted: false},
			wantPaths:  []string{"counts.services", "multiCluster.participationDetected", "multiCluster.signals", "multiCluster.meshNetworks"},
			rejectPath: "dataPlane.mode",
		},
		{
			name:       "pods affect count and data-plane evidence",
			permission: collect.Permission{Resource: "pods", Granted: false, DeniedScopes: []string{"payments"}},
			wantPaths:  []string{"counts.pods", "dataPlane.mode"},
			rejectPath: "multiCluster.participationDetected",
		},
		{
			name:       "peer authentication affects only its count",
			permission: collect.Permission{APIGroup: "security.istio.io", Resource: "peerauthentications", Granted: false},
			wantPaths:  []string{"counts.peerAuthentications"},
			rejectPath: "dataPlane.mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inventoryAvailability(collect.Snapshot{PermissionSummary: []collect.Permission{tt.permission}})
			for _, path := range tt.wantPaths {
				availability, exists := got[path]
				if !exists || availability.Available || !strings.Contains(availability.Reason, tt.permission.Resource) {
					t.Fatalf("availability[%q] = %#v, want permission-derived unknown in %#v", path, availability, got)
				}
			}
			if _, exists := got[tt.rejectPath]; exists {
				t.Fatalf("availability unexpectedly contains %q: %#v", tt.rejectPath, got)
			}
		})
	}
}

func TestPermissionSummaryDerivesAffectedControlsFromLoadedPacks(t *testing.T) {
	directory := t.TempDir()
	packPath := filepath.Join(directory, "permission-controls.yaml")
	packData := []byte(`apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: permission-controls, version: 1.0.0}
controls:
  - id: ACME-INV-001
    title: Service inventory must be non-empty
    category: governance
    severity: low
    evidenceType: context
    scope: namespace
    requires: [inventory.counts.services]
    applicability: 'true'
    expression: 'inventory.counts.services > 0'
    message: No services were observed.
    remediation: {guidance: Confirm service collection.}
  - id: ACME-ENV-001
    title: Namespaces must identify their team
    category: governance
    severity: medium
    evidenceType: context
    scope: namespace
    requires: [namespace.labels.team]
    applicability: 'true'
    expression: 'namespace.labels.team != ""'
    message: Namespace team is unavailable.
    remediation: {guidance: Add a team label.}
  - id: ACME-GOV-002
    title: Workload names must be present
    category: governance
    severity: low
    evidenceType: context
    scope: workload
    requires: [workload.workload.name]
    applicability: 'true'
    expression: 'workload.workload.name != ""'
    message: Workload name is unavailable.
    remediation: {guidance: Restore workload collection.}
`)
	if err := os.WriteFile(packPath, packData, 0o600); err != nil {
		t.Fatalf("write permission pack: %v", err)
	}
	packs, err := engine.LoadPacks([]string{packPath})
	if err != nil {
		t.Fatalf("load permission pack: %v", err)
	}
	permissions := []collect.Permission{
		{Resource: "services", Granted: false},
		{Resource: "namespaces", Granted: false},
		{APIGroup: "security.istio.io", Resource: "peerauthentications", Granted: false},
		{Resource: "pods", Granted: false},
	}
	got := permissionSummaryWithControls(permissions, packs)
	want := [][]string{
		{"ACME-INV-001"},
		{"ACME-ENV-001", "ACME-INV-001"},
		{"MG-MTLS-001", "MG-MTLS-002", "MG-MTLS-003"},
		{"ACME-ENV-001", "ACME-GOV-002", "ACME-INV-001", "MG-MTLS-001", "MG-MTLS-002", "MG-MTLS-003"},
	}
	for index := range want {
		if strings.Join(got[index].AffectedControls, ",") != strings.Join(want[index], ",") {
			t.Fatalf("permission %s affected controls = %#v, want %#v", got[index].Resource, got[index].AffectedControls, want[index])
		}
	}
}

func TestNamespaceInputsMarkDeniedLabelsUnavailable(t *testing.T) {
	snapshot := collect.Snapshot{PermissionSummary: []collect.Permission{{
		Resource: "namespaces", Verbs: []string{"list"}, Granted: false,
	}}}
	got := namespaceInputs(snapshot, nil, []string{"payments"})
	if len(got) != 1 || got[0].Name != "payments" {
		t.Fatalf("namespace inputs = %#v, want requested namespace", got)
	}
	availability, exists := got[0].Availability["labels"]
	if !exists || availability.Available || !strings.Contains(availability.Reason, "permission") {
		t.Fatalf("label availability = %#v, want permission-derived unavailable", got[0].Availability)
	}
}

func TestScanRejectsResourceControlsInsteadOfSilentlySkipping(t *testing.T) {
	err := validateScanControlScopes([]engine.Pack{{
		File:     "resource-pack.yaml",
		Controls: []engine.Control{{ID: "ACME-GW-001", Scope: "resource"}},
	}})
	if err == nil || !strings.Contains(err.Error(), "resource-pack.yaml: control ACME-GW-001") || !strings.Contains(err.Error(), "unavailable in scan") {
		t.Fatalf("resource scope error = %v", err)
	}
}

func executeForTest(t *testing.T, info versionInfo, args ...string) (string, string, error) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCommand(info)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()

	return stdout.String(), stderr.String(), err
}
