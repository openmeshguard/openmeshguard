package output

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/normalize"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

const schemaVersion = "v1alpha1"

// ScanScope is the output-facing copy of the scan scope.
type ScanScope struct {
	AllNamespaces bool
	Namespaces    []string
}

// ScanInput contains the generated scan data needed for canonical JSON output.
type ScanInput struct {
	GeneratedAt       time.Time
	ScannerVersion    string
	ResolverVersion   string
	ClusterContext    string
	Scope             ScanScope
	PermissionSummary []collect.Permission
	Inventory         normalize.Inventory
	WorkloadPostures  []resolver.WorkloadResult
}

// WriteScanJSON writes an indented canonical JSON report.
func WriteScanJSON(w io.Writer, input ScanInput) error {
	packs, err := engine.LoadPacks(nil)
	if err != nil {
		return fmt.Errorf("load built-in control packs: %w", err)
	}
	evaluated, err := engine.Evaluate(packs, defaultEngineInput(input))
	if err != nil {
		return fmt.Errorf("evaluate built-in controls: %w", err)
	}
	return WriteScanJSONWithEvaluation(w, input, packs, evaluated)
}

// WriteScanJSONWithEvaluation writes a report using an already-computed rule
// engine result. The scan command uses this path so repeatable user packs and
// producer-specific availability facts are preserved without changing the
// frozen canonical JSON shape.
func WriteScanJSONWithEvaluation(w io.Writer, input ScanInput, packs []engine.Pack, evaluated engine.Result) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(buildReport(input, packs, evaluated)); err != nil {
		return fmt.Errorf("encode canonical report: %w", err)
	}
	return nil
}

func buildReport(input ScanInput, packs []engine.Pack, evaluated engine.Result) report {
	generatedAt := input.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	return report{
		SchemaVersion: schemaVersion,
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339),
		Scanner: scanner{
			Version:         input.ScannerVersion,
			ResolverVersion: input.ResolverVersion,
			ControlPacks:    controlPacks(packs),
		},
		Scan: scan{
			ClusterContext: input.ClusterContext,
			Scope: scope{
				AllNamespaces: input.Scope.AllNamespaces,
				Namespaces:    optionalNamespaces(input.Scope),
			},
			DataSources: dataSources{
				KubernetesAPI: true,
				Prometheus: prometheus{
					Enabled: false,
				},
			},
		},
		PermissionSummary: permissionSummary(input.PermissionSummary),
		Inventory:         inventory(input.Inventory),
		WorkloadPostures:  workloadPostures(input.WorkloadPostures),
		Findings:          findings(evaluated.Findings),
		Scores: scores{
			Overall:    nil,
			Categories: scoreCategories(evaluated.Scores),
		},
	}
}

func defaultEngineInput(input ScanInput) engine.Input {
	workloads := make([]engine.WorkloadInput, 0, len(input.WorkloadPostures))
	for _, posture := range input.WorkloadPostures {
		workloads = append(workloads, engine.WorkloadInput{
			Posture: posture,
			Namespace: engine.NamespaceInput{
				Name: posture.Ref.Namespace,
			},
		})
	}
	return engine.Input{
		Workloads: workloads,
		Inventory: map[string]any{
			"counts": input.Inventory.Counts,
			"dataPlane": map[string]any{
				"mode": string(input.Inventory.DataPlaneMode),
			},
			"multiCluster": map[string]any{
				"participationDetected": input.Inventory.MultiCluster.ParticipationDetected,
				"evaluated":             false,
				"signals":               input.Inventory.MultiCluster.Signals,
				"meshNetworks":          input.Inventory.MultiCluster.MeshNetworks,
			},
		},
	}
}

func optionalNamespaces(scope ScanScope) []string {
	if scope.AllNamespaces || len(scope.Namespaces) == 0 {
		return nil
	}
	return append([]string(nil), scope.Namespaces...)
}

func permissionSummary(permissions []collect.Permission) []permission {
	out := make([]permission, 0, len(permissions))
	for _, item := range permissions {
		out = append(out, permission{
			APIGroup:         item.APIGroup,
			Resource:         item.Resource,
			Verbs:            append([]string(nil), item.Verbs...),
			Granted:          item.Granted,
			Optional:         item.Optional,
			Impact:           item.Impact,
			AffectedControls: append([]string(nil), item.AffectedControls...),
		})
	}
	return out
}

func workloadPostures(workloads []resolver.WorkloadResult) []resolver.WorkloadResult {
	if workloads == nil {
		return []resolver.WorkloadResult{}
	}
	return workloads
}

func inventory(input normalize.Inventory) inventorySummary {
	mode := string(input.DataPlaneMode)
	if mode == string(resolver.ModeNotApplicable) || mode == "" {
		mode = string(resolver.ModeUnknown)
	}
	return inventorySummary{
		Counts: input.Counts,
		DataPlane: dataPlane{
			Mode: mode,
		},
		MultiCluster: multiCluster{
			ParticipationDetected: input.MultiCluster.ParticipationDetected,
			Evaluated:             false,
			Signals:               input.MultiCluster.Signals,
			MeshNetworks:          input.MultiCluster.MeshNetworks,
		},
	}
}

func controlPacks(packs []engine.Pack) []controlPack {
	provenance := engine.ProvenanceFor(packs)
	out := make([]controlPack, 0, len(provenance))
	for _, pack := range provenance {
		out = append(out, controlPack{Name: pack.Name, Version: pack.Version, Source: pack.Source})
	}
	return out
}

func findings(input []engine.Finding) []finding {
	out := make([]finding, 0, len(input))
	for _, item := range input {
		resources := make([]resourceRef, 0, len(item.Resources))
		for _, resource := range item.Resources {
			resources = append(resources, resourceRef{
				APIVersion: resource.APIVersion,
				Kind:       resource.Kind,
				Namespace:  resource.Namespace,
				Name:       resource.Name,
			})
		}
		var findingRemediation *remediation
		if item.Remediation.Guidance != "" || item.Remediation.SuggestedYAML != "" {
			findingRemediation = &remediation{
				Guidance:      item.Remediation.Guidance,
				SuggestedYAML: item.Remediation.SuggestedYAML,
			}
		}
		out = append(out, finding{
			ID:              item.ID,
			ControlID:       item.ControlID,
			Title:           item.Title,
			Severity:        item.Severity,
			EvidenceType:    item.EvidenceType,
			Status:          item.Status,
			Confidence:      item.Confidence,
			DataPlaneMode:   item.DataPlaneMode,
			EvidenceSources: append([]string(nil), item.EvidenceSources...),
			Resources:       resources,
			ResolutionChain: append([]resolver.Step(nil), item.ResolutionChain...),
			Reasoning:       item.Reasoning,
			Remediation:     findingRemediation,
			UnknownReason:   item.UnknownReason,
		})
	}
	return out
}

func scoreCategories(input []engine.CategoryScore) []scoreCategory {
	out := make([]scoreCategory, 0, len(input))
	for _, item := range input {
		out = append(out, scoreCategory{
			Category:  item.Category,
			Grade:     item.Grade,
			PassRate:  item.PassRate,
			Evaluated: item.Evaluated,
			Unknown:   item.Unknown,
		})
	}
	return out
}

type report struct {
	SchemaVersion     string                    `json:"schemaVersion"`
	GeneratedAt       string                    `json:"generatedAt"`
	Scanner           scanner                   `json:"scanner"`
	Scan              scan                      `json:"scan"`
	PermissionSummary []permission              `json:"permissionSummary"`
	Inventory         inventorySummary          `json:"inventory"`
	WorkloadPostures  []resolver.WorkloadResult `json:"workloadPostures"`
	Findings          []finding                 `json:"findings"`
	Scores            scores                    `json:"scores"`
}

type scanner struct {
	Version         string        `json:"version"`
	ResolverVersion string        `json:"resolverVersion"`
	ControlPacks    []controlPack `json:"controlPacks"`
}

type controlPack struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

type scan struct {
	ClusterContext string      `json:"clusterContext"`
	Scope          scope       `json:"scope"`
	DataSources    dataSources `json:"dataSources"`
}

type scope struct {
	AllNamespaces bool     `json:"allNamespaces"`
	Namespaces    []string `json:"namespaces,omitempty"`
}

type dataSources struct {
	KubernetesAPI bool       `json:"kubernetesAPI"`
	Prometheus    prometheus `json:"prometheus"`
}

type prometheus struct {
	Enabled bool `json:"enabled"`
}

type permission struct {
	APIGroup         string   `json:"apiGroup"`
	Resource         string   `json:"resource"`
	Verbs            []string `json:"verbs"`
	Granted          bool     `json:"granted"`
	Optional         bool     `json:"optional,omitempty"`
	Impact           string   `json:"impact,omitempty"`
	AffectedControls []string `json:"affectedControls,omitempty"`
}

type inventorySummary struct {
	Counts       map[string]int `json:"counts"`
	DataPlane    dataPlane      `json:"dataPlane"`
	MultiCluster multiCluster   `json:"multiCluster"`
}

type dataPlane struct {
	Mode string `json:"mode"`
}

type multiCluster struct {
	ParticipationDetected bool     `json:"participationDetected"`
	Evaluated             bool     `json:"evaluated"`
	Signals               []string `json:"signals,omitempty"`
	MeshNetworks          []string `json:"meshNetworks,omitempty"`
}

type finding struct {
	ID              string          `json:"id"`
	ControlID       string          `json:"controlId"`
	Title           string          `json:"title,omitempty"`
	Severity        string          `json:"severity"`
	EvidenceType    string          `json:"evidenceType"`
	Status          string          `json:"status"`
	Confidence      string          `json:"confidence"`
	DataPlaneMode   string          `json:"dataPlaneMode,omitempty"`
	EvidenceSources []string        `json:"evidenceSources,omitempty"`
	Resources       []resourceRef   `json:"resources"`
	ResolutionChain []resolver.Step `json:"resolutionChain,omitempty"`
	Reasoning       string          `json:"reasoning"`
	Remediation     *remediation    `json:"remediation,omitempty"`
	UnknownReason   string          `json:"unknownReason,omitempty"`
}

type remediation struct {
	Guidance      string `json:"guidance,omitempty"`
	SuggestedYAML string `json:"suggestedYAML,omitempty"`
}

type resourceRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

type scores struct {
	Overall    *float64        `json:"overall"`
	Categories []scoreCategory `json:"categories"`
}

type scoreCategory struct {
	Category  string   `json:"category"`
	Grade     string   `json:"grade"`
	PassRate  *float64 `json:"passRate"`
	Evaluated int      `json:"evaluated,omitempty"`
	Unknown   int      `json:"unknown,omitempty"`
}
