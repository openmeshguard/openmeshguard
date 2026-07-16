package engine

import (
	"github.com/google/cel-go/cel"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

const (
	APIVersion = "openmeshguard.io/v1alpha1"
	Kind       = "ControlPack"
)

type Source string

const (
	SourceBuiltin Source = "builtin"
	SourceUser    Source = "user"
)

// Pack is a validated, compiled control pack.
type Pack struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   Metadata       `yaml:"metadata"`
	Params     map[string]any `yaml:"params,omitempty"`
	Controls   []Control      `yaml:"controls"`

	File   string `yaml:"-"`
	Source Source `yaml:"-"`
}

type Metadata struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type Control struct {
	ID            string      `yaml:"id"`
	Title         string      `yaml:"title"`
	Category      string      `yaml:"category"`
	Severity      string      `yaml:"severity"`
	EvidenceType  string      `yaml:"evidenceType"`
	Scope         string      `yaml:"scope"`
	Environments  []string    `yaml:"environments,omitempty"`
	Requires      []string    `yaml:"requires"`
	Applicability string      `yaml:"applicability"`
	Expression    string      `yaml:"expression"`
	Message       string      `yaml:"message"`
	Remediation   Remediation `yaml:"remediation"`
	Frameworks    []string    `yaml:"frameworks,omitempty"`
	Match         Match       `yaml:"match,omitempty"`

	applicabilityProgram cel.Program
	expressionProgram    cel.Program
	requiredPaths        []string
	applicabilityPaths   []string
	expressionPaths      []string
}

type Remediation struct {
	Guidance              string `yaml:"guidance"`
	SuggestedYAMLTemplate string `yaml:"suggestedYAMLTemplate,omitempty"`
	SuggestedYAML         string `yaml:"-"`
}

type Match struct {
	APIGroups []string `yaml:"apiGroups,omitempty"`
	Kinds     []string `yaml:"kinds,omitempty"`
}

// Availability overrides the engine's default availability inference for a
// dotted path. It is how producers distinguish a known false/empty value from
// a field whose evidence has not been collected yet.
type Availability struct {
	Available bool
	Reason    string
}

// NamespaceInput is the normalized namespace view exposed to namespace and
// workload CEL environments.
type NamespaceInput struct {
	Name           string
	Labels         map[string]string
	Environment    string
	MeshEnrollment string
	Availability   map[string]Availability
}

// WorkloadInput joins one resolver output to its namespace context and any
// producer-supplied availability facts.
type WorkloadInput struct {
	Posture      resolver.WorkloadResult
	Namespace    NamespaceInput
	Environment  string
	Owner        string
	AppID        string
	Verified     map[string]any
	Availability map[string]Availability
}

// ResourceInput is the normalized resource view for resource-scoped controls.
type ResourceInput struct {
	APIVersion      string
	Kind            string
	Namespace       string
	Name            string
	Environment     string
	Fields          map[string]any
	EvidenceSources []string
	Availability    map[string]Availability
}

// Input is the complete evaluation input. Inventory and Params are dynamic
// maps because their contract-backed shapes expand in later milestones.
type Input struct {
	Workloads  []WorkloadInput
	Namespaces []NamespaceInput
	// NamespaceTargetsComplete prevents namespace-scope evaluation from
	// deriving additional targets from workload context. The scan path sets it
	// after selecting mesh and unknown-enrollment namespaces.
	NamespaceTargetsComplete bool
	Resources                []ResourceInput
	Inventory                map[string]any
	InventoryAvailability    map[string]Availability
	Params                   map[string]any
}

type ResourceRef struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

type Finding struct {
	ID              string
	ControlID       string
	Title           string
	Severity        string
	EvidenceType    string
	Status          string
	Confidence      string
	DataPlaneMode   string
	EvidenceSources []string
	Resources       []ResourceRef
	ResolutionChain []resolver.Step
	Reasoning       string
	Remediation     Remediation
	UnknownReason   string
}

type CategoryScore struct {
	Category  string
	Grade     string
	PassRate  *float64
	Evaluated int
	Unknown   int
}

type Result struct {
	Findings []Finding
	Scores   []CategoryScore
}

type Provenance struct {
	Name    string
	Version string
	Source  string
}

func ProvenanceFor(packs []Pack) []Provenance {
	out := make([]Provenance, 0, len(packs))
	for _, pack := range packs {
		out = append(out, Provenance{
			Name:    pack.Metadata.Name,
			Version: pack.Metadata.Version,
			Source:  string(pack.Source),
		})
	}
	return out
}
