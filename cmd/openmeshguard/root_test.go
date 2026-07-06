package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/collect"
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
