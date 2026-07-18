package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWriteKindConfigQuotesHostPaths(t *testing.T) {
	t.Parallel()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	temporaryDirectory := t.TempDir()
	configPath := filepath.Join(temporaryDirectory, "kind config.yaml")
	auditPolicyPath := filepath.Join(temporaryDirectory, "review # policy's.yaml")
	auditDirectory := filepath.Join(temporaryDirectory, "audit # state", "reviewer's\nlogs")
	libPath := filepath.Join(workingDirectory, "lib.sh")

	command := exec.Command(
		"sh",
		"-c",
		`. "$1"; write_kind_config "$2" "$3" "$4"`,
		filepath.Join(workingDirectory, "kind-config-test"),
		libPath,
		configPath,
		auditPolicyPath,
		auditDirectory,
	)
	if output, commandErr := command.CombinedOutput(); commandErr != nil {
		t.Fatalf("write Kind config: %v\n%s", commandErr, output)
	}

	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read Kind config: %v", err)
	}
	var config struct {
		Nodes []struct {
			ExtraMounts []struct {
				HostPath string `yaml:"hostPath"`
			} `yaml:"extraMounts"`
		} `yaml:"nodes"`
	}
	if err := yaml.Unmarshal(contents, &config); err != nil {
		t.Fatalf("parse Kind config: %v\n%s", err, contents)
	}
	if len(config.Nodes) != 1 {
		t.Fatalf("got %d Kind nodes, want 1", len(config.Nodes))
	}

	got := make([]string, 0, len(config.Nodes[0].ExtraMounts))
	for _, mount := range config.Nodes[0].ExtraMounts {
		got = append(got, mount.HostPath)
	}
	want := []string{auditPolicyPath, auditDirectory}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("host paths = %#v, want %#v\n%s", got, want, contents)
	}
}
