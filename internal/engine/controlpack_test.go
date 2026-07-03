package engine

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type controlPack struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   controlMetadata  `yaml:"metadata"`
	Controls   []map[string]any `yaml:"controls"`
}

type controlMetadata struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func TestBuiltinControlPacksAreLoadable(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "controls", "*.yaml"))
	if err != nil {
		t.Fatalf("glob control packs: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("expected at least one built-in control pack")
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			file, err := os.Open(path)
			if err != nil {
				t.Fatalf("open control pack: %v", err)
			}
			defer file.Close()

			var pack controlPack
			decoder := yaml.NewDecoder(file)
			decoder.KnownFields(true)
			if err := decoder.Decode(&pack); err != nil {
				t.Fatalf("decode control pack: %v", err)
			}

			if pack.APIVersion != "openmeshguard.io/v1alpha1" {
				t.Fatalf("apiVersion = %q, want openmeshguard.io/v1alpha1", pack.APIVersion)
			}
			if pack.Kind != "ControlPack" {
				t.Fatalf("kind = %q, want ControlPack", pack.Kind)
			}
			if pack.Metadata.Name == "" {
				t.Fatal("metadata.name is required")
			}
			if pack.Metadata.Version == "" {
				t.Fatal("metadata.version is required")
			}
			if pack.Controls == nil {
				t.Fatal("controls must be present, even when empty")
			}
		})
	}
}
