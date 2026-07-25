package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	governance "github.com/openmeshguard/openmeshguard/internal/context"
	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/normalize"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	appsv1 "k8s.io/api/apps/v1"
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

func TestProjectionCommandsReadCanonicalJSON(t *testing.T) {
	root := filepath.Join("..", "..")
	golden := filepath.Join(
		root,
		"test",
		"fixtures",
		"governance-context",
		"golden",
		"governance-active-exception.json",
	)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "report", args: []string{"report", "--input", golden}, want: "<!doctype html>"},
		{name: "export", args: []string{"export", "--input", golden}, want: `"version": "2.1.0"`},
		{name: "score", args: []string{"score", "--input", golden}, want: "OpenMeshGuard score:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeForTest(t, defaultVersionInfo(), tt.args...)
			if err != nil {
				t.Fatalf("%s returned error: %v", tt.name, err)
			}
			if stderr != "" {
				t.Fatalf("%s wrote stderr %q", tt.name, stderr)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Fatalf("%s output missing %q: %q", tt.name, tt.want, stdout)
			}
		})
	}
}

func TestProjectionCommandWritesRequestedOutput(t *testing.T) {
	golden := filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"sidecar-basic",
		"golden",
		"namespace-role-degraded.json",
	)
	outputPath := filepath.Join(t.TempDir(), "report.html")
	stdout, _, err := executeForTest(
		t,
		defaultVersionInfo(),
		"report",
		"--input",
		golden,
		"--output",
		outputPath,
	)
	if err != nil {
		t.Fatalf("report command returned error: %v", err)
	}
	if stdout != "" {
		t.Fatalf("report command wrote stdout with --output: %q", stdout)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read report output: %v", err)
	}
	if !bytes.Contains(data, []byte("Declared / Verified / Unknown")) {
		t.Fatalf("report output missing summary: %s", data)
	}
}

func TestScoreCommandExitCodeContract(t *testing.T) {
	root := filepath.Join("..", "..", "test", "fixtures")
	active := filepath.Join(root, "governance-context", "golden", "governance-active-exception.json")
	degraded := filepath.Join(root, "sidecar-basic", "golden", "namespace-role-degraded.json")
	tests := []struct {
		name     string
		input    string
		flags    []string
		wantCode int
	}{
		{name: "excepted critical does not fail critical", input: active, flags: []string{"--fail-on", "critical"}, wantCode: 0},
		{name: "open high fails high", input: active, flags: []string{"--fail-on", "high"}, wantCode: 1},
		{name: "unknown critical excluded by default", input: degraded, flags: []string{"--fail-on", "critical"}, wantCode: 0},
		{name: "unknown opt in", input: degraded, flags: []string{"--fail-on", "critical", "--fail-on-unknown"}, wantCode: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"score", "--input", tt.input}
			args = append(args, tt.flags...)
			stdout, _, err := executeForTest(t, defaultVersionInfo(), args...)
			if !strings.Contains(stdout, "OpenMeshGuard score:") {
				t.Fatalf("score output missing despite exit contract evaluation: %q", stdout)
			}
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("score returned error: %v", err)
				}
				return
			}
			if !errors.Is(err, errFindingsThreshold) {
				t.Fatalf("score error = %v, want threshold sentinel", err)
			}
			if got := exitCode(err); got != tt.wantCode {
				t.Fatalf("score exit code = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestExitCodeUsesTwoForScanOrProjectionErrors(t *testing.T) {
	if got := exitCode(fmt.Errorf("scan failed")); got != 2 {
		t.Fatalf("scan error exit code = %d, want 2", got)
	}
	if got := exitCode(fmt.Errorf("wrapped: %w", errFindingsThreshold)); got != 1 {
		t.Fatalf("threshold error exit code = %d, want 1", got)
	}
}

func TestProjectionBrokenPipeUsesExitTwo(t *testing.T) {
	command := exec.Command(
		os.Args[0],
		"-test.run=^TestProjectionBrokenPipeHelper$",
	)
	command.Env = append(os.Environ(), "OPENMESHGUARD_BROKEN_PIPE_HELPER=1")
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("create helper stdout pipe: %v", err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	if err := stdout.Close(); err != nil {
		t.Fatalf("close helper stdout reader: %v", err)
	}
	err = command.Wait()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("helper error = %v, want exit status 2; stderr=%q", err, stderr.String())
	}
	if exitError.ExitCode() != 2 {
		t.Fatalf("broken-pipe exit code = %d, want 2; stderr=%q", exitError.ExitCode(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "broken pipe") {
		t.Fatalf("broken-pipe stderr = %q, want operational error", stderr.String())
	}
}

func TestProjectionBrokenPipeHelper(t *testing.T) {
	if os.Getenv("OPENMESHGUARD_BROKEN_PIPE_HELPER") != "1" {
		return
	}
	ignoreBrokenPipeSignal()
	golden := filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"sidecar-basic",
		"golden",
		"namespace-role-degraded.json",
	)
	command := newRootCommand(defaultVersionInfo())
	command.SetArgs([]string{"report", "--input", golden})
	command.SetOut(os.Stdout)
	command.SetErr(os.Stderr)
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
	os.Exit(0)
}

func TestScoreCommandUsesExitTwoForSchemaInvalidCanonicalInput(t *testing.T) {
	golden := filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"governance-context",
		"golden",
		"governance-expired-exception.json",
	)
	data, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	invalid := bytes.Replace(data, []byte(`"status": "open"`), []byte(`"status": "bogus"`), 1)
	if bytes.Equal(invalid, data) {
		t.Fatal("golden had no open finding to corrupt")
	}
	input := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(input, invalid, 0o600); err != nil {
		t.Fatalf("write invalid canonical input: %v", err)
	}

	_, _, err = executeForTest(
		t,
		defaultVersionInfo(),
		"score",
		"--input",
		input,
		"--fail-on",
		"info",
	)
	if err == nil || !strings.Contains(err.Error(), "validate canonical report") {
		t.Fatalf("score error = %v, want canonical validation failure", err)
	}
	if got := exitCode(err); got != 2 {
		t.Fatalf("invalid canonical input exit code = %d, want 2", got)
	}
}

func TestProjectionCommandsUseExitTwoForFormatAndEncodingErrors(t *testing.T) {
	golden := filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"sidecar-basic",
		"golden",
		"strict.json",
	)
	valid, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	invalidDate := bytes.Replace(
		valid,
		[]byte(`"generatedAt": "2000-01-01T00:00:00Z"`),
		[]byte(`"generatedAt": "not-a-date"`),
		1,
	)
	if bytes.Equal(invalidDate, valid) {
		t.Fatal("golden had no generatedAt fixture to corrupt")
	}
	invalidUTF8 := bytes.Replace(
		valid,
		[]byte(`"clusterContext": "openmeshguard-e2e"`),
		append(
			[]byte(`"clusterContext": "openmeshguard-`),
			append([]byte{0xff}, []byte(`e2e"`)...)...,
		),
		1,
	)
	if bytes.Equal(invalidUTF8, valid) {
		t.Fatal("golden had no clusterContext fixture to corrupt")
	}

	inputs := []struct {
		name      string
		data      []byte
		wantError string
	}{
		{name: "date format", data: invalidDate, wantError: "validate canonical report"},
		{name: "UTF-8", data: invalidUTF8, wantError: "valid UTF-8"},
	}
	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid.json")
			if err := os.WriteFile(path, input.data, 0o600); err != nil {
				t.Fatalf("write invalid canonical input: %v", err)
			}
			for _, projection := range []string{"report", "export", "score"} {
				t.Run(projection, func(t *testing.T) {
					stdout, _, err := executeForTest(
						t,
						defaultVersionInfo(),
						projection,
						"--input",
						path,
					)
					if err == nil || !strings.Contains(err.Error(), input.wantError) {
						t.Fatalf("%s error = %v, want %q", projection, err, input.wantError)
					}
					if stdout != "" {
						t.Fatalf("%s emitted output before rejecting canonical input: %q", projection, stdout)
					}
					if got := exitCode(err); got != 2 {
						t.Fatalf("%s exit code = %d, want 2", projection, got)
					}
				})
			}
		})
	}
}

func TestProjectionCommandsUseExitTwoForNegativeCanonicalCounters(t *testing.T) {
	golden := filepath.Join(
		"..",
		"..",
		"test",
		"fixtures",
		"sidecar-basic",
		"golden",
		"strict.json",
	)
	valid, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	tests := []struct {
		name        string
		old         []byte
		replacement []byte
	}{
		{
			name:        "score evaluated",
			old:         []byte(`"evaluated": 5`),
			replacement: []byte(`"evaluated": -1`),
		},
		{
			name:        "waypoints",
			old:         []byte(`"waypoints": 0`),
			replacement: []byte(`"waypoints": -1`),
		},
		{
			name:        "classified namespaces",
			old:         []byte(`"namespacesClassified": 0`),
			replacement: []byte(`"namespacesClassified": -1`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalid := bytes.Replace(valid, tt.old, tt.replacement, 1)
			if bytes.Equal(invalid, valid) {
				t.Fatalf("golden had no %q counter to corrupt", tt.old)
			}
			path := filepath.Join(t.TempDir(), "invalid.json")
			if err := os.WriteFile(path, invalid, 0o600); err != nil {
				t.Fatalf("write invalid canonical input: %v", err)
			}
			for _, projection := range []string{"report", "export", "score"} {
				t.Run(projection, func(t *testing.T) {
					stdout, _, err := executeForTest(
						t,
						defaultVersionInfo(),
						projection,
						"--input",
						path,
					)
					if err == nil || !strings.Contains(err.Error(), "validate canonical report") {
						t.Fatalf("%s error = %v, want negative-counter schema failure", projection, err)
					}
					if stdout != "" {
						t.Fatalf("%s emitted output before rejecting negative counter: %q", projection, stdout)
					}
					if got := exitCode(err); got != 2 {
						t.Fatalf("%s exit code = %d, want 2", projection, got)
					}
				})
			}
		})
	}
}

func TestScanAndScoreRejectInvalidThresholdBeforeWork(t *testing.T) {
	for _, args := range [][]string{
		{"scan", "--all-namespaces", "--fail-on", "urgent"},
		{"score", "--fail-on", "urgent"},
	} {
		_, _, err := executeForTest(t, defaultVersionInfo(), args...)
		if err == nil || !strings.Contains(err.Error(), "invalid --fail-on severity") {
			t.Fatalf("%v error = %v, want threshold validation", args, err)
		}
		if got := exitCode(err); got != 2 {
			t.Fatalf("%v exit code = %d, want 2", args, got)
		}
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

func TestScanGovernanceFlags(t *testing.T) {
	cmd := newScanCommand(defaultVersionInfo())
	for _, name := range []string{
		"scan-config",
		"ownership-import",
		"exceptions",
		"infer-environments",
		"fail-on",
		"fail-on-unknown",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("scan command missing %s flag", name)
		}
	}
	if cmd.Flags().Lookup("infer-environments").DefValue != "false" {
		t.Fatalf("infer-environments default = %q, want false", cmd.Flags().Lookup("infer-environments").DefValue)
	}
	if cmd.Flags().Lookup("exceptions").Value.Type() != "stringArray" {
		t.Fatalf("exceptions flag type = %q, want stringArray", cmd.Flags().Lookup("exceptions").Value.Type())
	}
}

func TestResolveContextPathsRejectsAmbiguousDeclarations(t *testing.T) {
	config := governance.ScanConfig{Inputs: governance.ContextInputs{
		OwnershipImport: "config-ownership.yaml",
		Exceptions:      []string{"config-exceptions"},
	}}
	tests := []struct {
		name string
		opts scanOptions
		want string
	}{
		{name: "ownership", opts: scanOptions{OwnershipImport: "cli-ownership.yaml"}, want: "both --ownership-import"},
		{name: "exceptions", opts: scanOptions{Exceptions: []string{"cli-exceptions"}}, want: "both --exceptions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := resolveContextPaths(tt.opts, config)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("resolveContextPaths error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestEngineExceptionInputsAreAnnotationOnly(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	records := []governance.ExceptionRecord{{
		Metadata: governance.Metadata{Name: "EXC-ACTIVE"},
		Spec: governance.ExceptionSpec{
			ControlIDs: []string{"MG-MTLS-001"}, Owner: "payments-team",
			Approver: "security", Ticket: "https://tickets.example/active",
		},
		ExpiresAt: now.Add(time.Hour),
	}, {
		Metadata: governance.Metadata{Name: "EXC-EXPIRED"},
		Spec: governance.ExceptionSpec{
			ControlIDs: []string{"MG-MTLS-001"}, Owner: "payments-team",
			Approver: "security", Ticket: "https://tickets.example/expired",
		},
		ExpiresAt: now.Add(-time.Hour),
	}}
	workloads := []governance.WorkloadContext{
		{
			Ref:         resolver.WorkloadRef{Kind: "Deployment", Namespace: "payments", Name: "api"},
			Ownership:   governance.Ownership{Owner: "payments-team", OwnerKnown: true},
			ExceptionID: "EXC-ACTIVE",
		},
		{
			Ref:         resolver.WorkloadRef{Kind: "Deployment", Namespace: "payments", Name: "worker"},
			Ownership:   governance.Ownership{Owner: "payments-team", OwnerKnown: true},
			ExceptionID: "MISSING",
		},
	}
	resources, exceptions, bindings := engineExceptionInputs(records, workloads, now)
	if len(exceptions) != 2 || exceptions[0].Expired || !exceptions[1].Expired {
		t.Fatalf("exception inputs = %#v", exceptions)
	}
	if len(bindings) != 2 {
		t.Fatalf("bindings = %#v, want exact annotated workload bindings", bindings)
	}
	dangling := false
	for _, resource := range resources {
		if resource.Kind == "ExceptionReference" {
			dangling = true
		}
	}
	if !dangling {
		t.Fatalf("resources = %#v, want dangling annotation validation target", resources)
	}
}

func TestEngineExceptionInputsRejectOwnerReplayAndUnknownOwner(t *testing.T) {
	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	records := []governance.ExceptionRecord{{
		Metadata: governance.Metadata{Name: "EXC-ACTIVE"},
		Spec: governance.ExceptionSpec{
			ControlIDs: []string{"MG-MTLS-001"}, Owner: "payments-team",
			Approver: "security", Ticket: "https://tickets.example/active",
		},
		ExpiresAt: now.Add(time.Hour),
	}}
	workloads := []governance.WorkloadContext{
		{
			Ref:         resolver.WorkloadRef{Kind: "Deployment", Namespace: "other", Name: "api"},
			Ownership:   governance.Ownership{Owner: "other-team", OwnerKnown: true},
			ExceptionID: "EXC-ACTIVE",
		},
		{
			Ref:         resolver.WorkloadRef{Kind: "Deployment", Namespace: "unknown", Name: "api"},
			Ownership:   governance.Ownership{OwnerKnown: false, OwnerReason: "namespace metadata unavailable"},
			ExceptionID: "EXC-ACTIVE",
		},
	}
	resources, _, bindings := engineExceptionInputs(records, workloads, now)
	if len(bindings) != 2 {
		t.Fatalf("bindings = %#v, want both rejected bindings preserved for engine defense", bindings)
	}
	var mismatch, unknown bool
	for _, resource := range resources {
		if resource.Kind != "ExceptionReference" {
			continue
		}
		switch resource.Namespace {
		case "other":
			errors, _ := resource.Fields["validationErrors"].([]any)
			mismatch = len(errors) == 1 && strings.Contains(fmt.Sprint(errors[0]), "owner")
		case "unknown":
			availability := resource.Availability["validationErrors"]
			unknown = !availability.Available && strings.Contains(availability.Reason, "unavailable")
		}
	}
	if !mismatch || !unknown {
		t.Fatalf("resources = %#v, want owner mismatch and unknown-owner reference targets", resources)
	}
}

func TestValidateExceptionControlIDsUsesLoadedPacks(t *testing.T) {
	records := []governance.ExceptionRecord{
		{
			Metadata: governance.Metadata{Name: "known"},
			Spec:     governance.ExceptionSpec{ControlIDs: []string{"MG-MTLS-001"}},
		},
		{
			Metadata: governance.Metadata{Name: "unknown"},
			Spec:     governance.ExceptionSpec{ControlIDs: []string{"MG-MTLS-099"}},
		},
	}
	got := validateExceptionControlIDs(records, []engine.Pack{{
		Controls: []engine.Control{{ID: "MG-MTLS-001"}},
	}})
	if len(got[0].ValidationErrors) != 0 ||
		len(got[1].ValidationErrors) != 1 ||
		!strings.Contains(got[1].ValidationErrors[0], "unknown control") {
		t.Fatalf("validated records = %#v", got)
	}
}

func TestGovernanceWorkloadInputsUseOwningResourceMetadata(t *testing.T) {
	controller := metav1.OwnerReference{Kind: "Deployment", Name: "api"}
	replicaSet := metav1.OwnerReference{Kind: "ReplicaSet", Name: "api-abc"}
	snapshot := collect.Snapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api", Namespace: "payments",
				Labels:      map[string]string{"app-id": "controller-id"},
				Annotations: map[string]string{"openmeshguard.io/exception": "EXC-42"},
			},
		}},
		ReplicaSets: []appsv1.ReplicaSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "standalone", Namespace: "payments",
					Annotations: map[string]string{"owner": "replicaset-team"},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "api-abc", Namespace: "payments",
					OwnerReferences: []metav1.OwnerReference{controller},
				},
			},
		},
		Pods: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-1", Namespace: "payments",
				Labels:          map[string]string{"app-id": "pod-id", "pod-only": "value"},
				OwnerReferences: []metav1.OwnerReference{replicaSet},
			},
		}},
	}
	workloads := []resolver.WorkloadInput{
		{
			Ref:    resolver.WorkloadRef{Kind: "Deployment", Namespace: "payments", Name: "api"},
			Labels: map[string]string{"app-id": "template-id", "template-only": "value"},
		},
		{
			Ref: resolver.WorkloadRef{Kind: "ReplicaSet", Namespace: "payments", Name: "standalone"},
		},
		{
			Ref:    resolver.WorkloadRef{Kind: "Pod", Namespace: "payments", Name: "api-1"},
			Labels: map[string]string{"pod-only": "value"},
		},
	}
	got := governanceWorkloadInputs(snapshot, workloads)
	if got[0].Labels["app-id"] != "controller-id" ||
		got[0].Labels["template-only"] != "value" ||
		got[0].Annotations["openmeshguard.io/exception"] != "EXC-42" {
		t.Fatalf("deployment metadata = %#v/%#v", got[0].Labels, got[0].Annotations)
	}
	if got[1].Annotations["owner"] != "replicaset-team" {
		t.Fatalf("ReplicaSet annotations = %#v", got[1].Annotations)
	}
	if got[2].Labels["pod-only"] != "value" ||
		got[2].Labels["app-id"] != "controller-id" ||
		got[2].Annotations["openmeshguard.io/exception"] != "EXC-42" {
		t.Fatalf("split Pod metadata = %#v/%#v", got[2].Labels, got[2].Annotations)
	}
}

func TestContextEvidenceSourcesIncludeAnnotationOwnership(t *testing.T) {
	sources := contextEvidenceSources(governance.WorkloadContext{
		Classification: governance.Classification{Source: "scan-config"},
		Ownership: governance.Ownership{
			AppIDSource: "workload annotation platform.example.com/application-id",
			OwnerSource: "namespace annotation platform.example.com/team",
		},
	})
	if len(sources) != 2 || sources[0] != "scan-config" || sources[1] != "kubernetes-api" {
		t.Fatalf("context evidence sources = %#v, want scan-config and kubernetes-api", sources)
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
		{name: "unknown dominates known non-mesh observation", workloads: []workloadObservation{{mode: resolver.ModeNotApplicable}, {ambient: resolver.Unobserved, mode: resolver.ModeUnknown}}, want: "unknown"},
		{name: "unknown dominance is order independent", workloads: []workloadObservation{{ambient: resolver.Unobserved, mode: resolver.ModeUnknown}, {mode: resolver.ModeNotApplicable}}, want: "unknown"},
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

func TestDataPlaneInventoryCanonicalizesNotApplicableMode(t *testing.T) {
	got := dataPlaneInventory(normalize.Inventory{DataPlaneMode: resolver.ModeNotApplicable})
	if got["mode"] != string(resolver.ModeUnknown) {
		t.Fatalf("inventory data plane mode = %#v, want canonical unknown", got["mode"])
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
			name:       "EndpointSlices affect their inventory count",
			permission: collect.Permission{APIGroup: "discovery.k8s.io", Resource: "endpointslices", Granted: false},
			wantPaths:  []string{"counts.endpointSlices"},
			rejectPath: "dataPlane.mode",
		},
		{
			name:       "peer authentication affects only its count",
			permission: collect.Permission{APIGroup: "security.istio.io", Resource: "peerauthentications", Granted: false},
			wantPaths:  []string{"counts.peerAuthentications"},
			rejectPath: "dataPlane.mode",
		},
		{
			name:       "DestinationRule affects its count",
			permission: collect.Permission{APIGroup: "networking.istio.io", Resource: "destinationrules", Granted: false},
			wantPaths:  []string{"counts.destinationRules"},
			rejectPath: "dataPlane.mode",
		},
		{
			name:       "Sidecar affects its count",
			permission: collect.Permission{APIGroup: "networking.istio.io", Resource: "sidecars", Granted: false},
			wantPaths:  []string{"counts.sidecars"},
			rejectPath: "dataPlane.mode",
		},
		{
			name:       "AuthorizationPolicy affects its count",
			permission: collect.Permission{APIGroup: "security.istio.io", Resource: "authorizationpolicies", Granted: false},
			wantPaths:  []string{"counts.authorizationPolicies"},
			rejectPath: "dataPlane.mode",
		},
		{
			name:       "Gateway affects its count",
			permission: collect.Permission{APIGroup: "gateway.networking.k8s.io", Resource: "gateways", Granted: false},
			wantPaths:  []string{"counts.gateways"},
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
		{APIGroup: "discovery.k8s.io", Resource: "endpointslices", Granted: false},
		{Resource: "namespaces", Granted: false},
		{APIGroup: "security.istio.io", Resource: "peerauthentications", Granted: false},
		{Resource: "pods", Granted: false},
	}
	got := permissionSummaryWithControls(permissions, packs)
	want := [][]string{
		{"ACME-INV-001", "MG-AUTHZ-001", "MG-AUTHZ-002", "MG-AUTHZ-003", "MG-AUTHZ-004", "MG-AUTHZ-005", "MG-AUTHZ-006", "MG-AUTHZ-007", "MG-GW-005", "MG-MTLS-002", "MG-MTLS-007"},
		{"MG-AUTHZ-001", "MG-AUTHZ-002", "MG-AUTHZ-003", "MG-AUTHZ-004", "MG-AUTHZ-005", "MG-AUTHZ-006", "MG-AUTHZ-007", "MG-GW-005", "MG-MTLS-002", "MG-MTLS-007"},
		{"ACME-ENV-001", "ACME-GOV-002", "ACME-INV-001", "MG-AUTHZ-001", "MG-AUTHZ-002", "MG-AUTHZ-003", "MG-AUTHZ-004", "MG-AUTHZ-005", "MG-AUTHZ-006", "MG-AUTHZ-007", "MG-ENV-001", "MG-GW-005", "MG-MTLS-001", "MG-MTLS-002", "MG-MTLS-003", "MG-MTLS-005", "MG-MTLS-006", "MG-MTLS-007", "MG-OWN-001", "MG-OWN-002"},
		{"MG-MTLS-001", "MG-MTLS-002", "MG-MTLS-003", "MG-MTLS-005", "MG-MTLS-006", "MG-MTLS-007"},
		{"ACME-ENV-001", "ACME-GOV-002", "ACME-INV-001", "MG-AUTHZ-001", "MG-AUTHZ-002", "MG-AUTHZ-003", "MG-AUTHZ-004", "MG-AUTHZ-005", "MG-AUTHZ-006", "MG-AUTHZ-007", "MG-ENV-001", "MG-GW-005", "MG-MTLS-001", "MG-MTLS-002", "MG-MTLS-003", "MG-MTLS-005", "MG-MTLS-006", "MG-MTLS-007", "MG-OWN-001", "MG-OWN-002"},
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

func TestScannerVersionResolutionOrder(t *testing.T) {
	tests := []struct {
		name          string
		ldflagsValue  string
		moduleVersion string
		want          string
	}{
		{name: "ldflags wins over module version", ldflagsValue: "v0.2.0", moduleVersion: "v0.1.0", want: "v0.2.0"},
		{name: "module version from go install", ldflagsValue: "", moduleVersion: "v0.1.0-alpha.1", want: "v0.1.0-alpha.1"},
		{name: "local go build reports placeholder", ldflagsValue: "", moduleVersion: "(devel)", want: versionPlaceholder},
		{name: "missing build info reports placeholder", ldflagsValue: "", moduleVersion: "", want: versionPlaceholder},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scannerVersion(tt.ldflagsValue, func() string { return tt.moduleVersion })
			if got != tt.want {
				t.Fatalf("scannerVersion = %q, want %q", got, tt.want)
			}
		})
	}
}
