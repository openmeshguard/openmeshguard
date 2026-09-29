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
	evaluated.Context = defaultReportContext(input)
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
				ContextFiles: contextFiles{
					ScanConfig:      evaluated.Context.ScanConfig,
					OwnershipImport: evaluated.Context.OwnershipImport,
					Exceptions:      evaluated.Context.Exceptions,
				},
			},
			EnvironmentInference: evaluated.Context.EnvironmentInference,
		},
		PermissionSummary: permissionSummary(input.PermissionSummary),
		Inventory:         inventory(input.Inventory, evaluated.Context.Classification),
		WorkloadPostures:  workloadPostures(input.WorkloadPostures, evaluated.Context.Workloads),
		Findings:          findings(evaluated.Findings),
		Scores:            reportScores(evaluated),
	}
}

func defaultEngineInput(input ScanInput) engine.Input {
	workloads := make([]engine.WorkloadInput, 0, len(input.WorkloadPostures))
	namespaces := map[string]engine.NamespaceInput{}
	for _, posture := range input.WorkloadPostures {
		namespace := engine.NamespaceInput{
			Name:             posture.Ref.Namespace,
			Environment:      "unclassified",
			EnvironmentKnown: true,
			MeshEnrollment:   defaultMeshEnrollment(posture.Mode),
		}
		namespaces[namespace.Name] = namespace
		workloads = append(workloads, engine.WorkloadInput{
			Posture:          posture,
			Namespace:        namespace,
			Environment:      "unclassified",
			EnvironmentKnown: true,
			OwnerKnown:       true,
			AppIDKnown:       true,
		})
	}
	namespaceInputs := make([]engine.NamespaceInput, 0, len(namespaces))
	for _, namespace := range namespaces {
		namespaceInputs = append(namespaceInputs, namespace)
	}
	return engine.Input{
		Workloads:                workloads,
		Namespaces:               namespaceInputs,
		NamespaceTargetsComplete: true,
		Inventory: map[string]any{
			"counts":    input.Inventory.Counts,
			"dataPlane": inventoryDataPlaneValue(input.Inventory),
			"multiCluster": map[string]any{
				"participationDetected": input.Inventory.MultiCluster.ParticipationDetected,
				"evaluated":             false,
				"signals":               input.Inventory.MultiCluster.Signals,
				"meshNetworks":          input.Inventory.MultiCluster.MeshNetworks,
			},
		},
	}
}

func defaultReportContext(input ScanInput) engine.ReportContext {
	seenNamespaces := map[string]struct{}{}
	workloads := make([]engine.WorkloadContext, 0, len(input.WorkloadPostures))
	for _, posture := range input.WorkloadPostures {
		seenNamespaces[posture.Ref.Namespace] = struct{}{}
		workloads = append(workloads, engine.WorkloadContext{
			Ref:                   posture.Ref,
			Environment:           "unclassified",
			EnvironmentConfidence: "resolved",
			EnvironmentKnown:      true,
			OwnerKnown:            true,
			AppIDKnown:            true,
		})
	}
	return engine.ReportContext{
		Classification: engine.ClassificationSummary{
			NamespacesUnclassified: len(seenNamespaces),
			ByEnvironment:          map[string]int{"unclassified": len(seenNamespaces)},
		},
		Workloads: workloads,
	}
}

func defaultMeshEnrollment(mode resolver.DataPlaneMode) string {
	switch mode {
	case resolver.ModeNotApplicable:
		return "not-enrolled"
	case resolver.ModeSidecar, resolver.ModeAmbient, resolver.ModeMixed:
		return "enrolled"
	default:
		return "unknown"
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

func workloadPostures(workloads []resolver.WorkloadResult, contexts []engine.WorkloadContext) []canonicalWorkloadPosture {
	byRef := make(map[string]engine.WorkloadContext, len(contexts))
	for _, workloadContext := range contexts {
		byRef[workloadRefKey(workloadContext.Ref)] = workloadContext
	}
	out := make([]canonicalWorkloadPosture, 0, len(workloads))
	for _, workload := range workloads {
		workloadContext := byRef[workloadRefKey(workload.Ref)]
		out = append(out, canonicalWorkloadPosture{
			Workload:              workload.Ref,
			Environment:           optionalContextValue(workloadContext.Environment, workloadContext.EnvironmentKnown),
			EnvironmentConfidence: optionalContextValue(workloadContext.EnvironmentConfidence, workloadContext.EnvironmentConfidence != ""),
			Owner:                 optionalNonEmptyContextValue(workloadContext.Owner, workloadContext.OwnerKnown),
			AppID:                 optionalNonEmptyContextValue(workloadContext.AppID, workloadContext.AppIDKnown),
			DataPlaneMode:         workload.Mode,
			MTLS:                  workload.MTLS,
			Authorization:         workload.Authz,
		})
	}
	return out
}

func inventory(input normalize.Inventory, context engine.ClassificationSummary) inventorySummary {
	mode := string(input.DataPlaneMode)
	if mode == string(resolver.ModeNotApplicable) || mode == "" {
		mode = string(resolver.ModeUnknown)
	}
	return inventorySummary{
		Counts: input.Counts,
		DataPlane: dataPlane{
			Mode: mode,
			Ztunnel: &ztunnel{
				Present:      input.Ztunnel.Present,
				NodesCovered: input.Ztunnel.NodesCovered,
				NodesTotal:   input.Ztunnel.NodesTotal,
			},
			Waypoints: input.Waypoints,
		},
		MultiCluster: multiCluster{
			ParticipationDetected: input.MultiCluster.ParticipationDetected,
			Evaluated:             false,
			Signals:               input.MultiCluster.Signals,
			MeshNetworks:          input.MultiCluster.MeshNetworks,
		},
		Classification: &classification{
			NamespacesClassified:   context.NamespacesClassified,
			NamespacesUnclassified: context.NamespacesUnclassified,
			ByEnvironment:          nonNilStringIntMap(context.ByEnvironment),
		},
	}
}

func inventoryDataPlaneValue(input normalize.Inventory) map[string]any {
	mode := string(input.DataPlaneMode)
	if mode == string(resolver.ModeNotApplicable) || mode == "" {
		mode = string(resolver.ModeUnknown)
	}
	ztunnelValue := map[string]any{"nodesTotal": nil}
	if input.Ztunnel.Present != nil {
		ztunnelValue["present"] = *input.Ztunnel.Present
	}
	if input.Ztunnel.NodesCovered != nil {
		ztunnelValue["nodesCovered"] = *input.Ztunnel.NodesCovered
	}
	if input.Ztunnel.NodesTotal != nil {
		ztunnelValue["nodesTotal"] = *input.Ztunnel.NodesTotal
	}
	value := map[string]any{
		"mode":    mode,
		"ztunnel": ztunnelValue,
	}
	if input.Waypoints != nil {
		value["waypoints"] = *input.Waypoints
	}
	return value
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
			Exception:       findingException(item.Exception),
			UnknownReason:   item.UnknownReason,
		})
	}
	return out
}

func findingException(input *engine.ExceptionEvidence) *exceptionEvidence {
	if input == nil {
		return nil
	}
	return &exceptionEvidence{
		ID:        input.ID,
		Expired:   input.Expired,
		ExpiresAt: input.ExpiresAt.UTC().Format(time.RFC3339),
		Approver:  input.Approver,
		Ticket:    input.Ticket,
	}
}

func workloadRefKey(ref resolver.WorkloadRef) string {
	return ref.Cluster + "/" + ref.Namespace + "/" + ref.Kind + "/" + ref.Name
}

func optionalContextValue(value string, known bool) *string {
	if !known {
		return nil
	}
	copied := value
	return &copied
}

func optionalNonEmptyContextValue(value string, known bool) *string {
	if !known || value == "" {
		return nil
	}
	copied := value
	return &copied
}

func nonNilStringIntMap(input map[string]int) map[string]int {
	if input == nil {
		return map[string]int{}
	}
	return input
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

func reportScores(evaluated engine.Result) scores {
	categories := scoreCategories(evaluated.Scores)
	namespaces := make([]namespaceScore, 0, len(evaluated.NamespaceScores))
	rollupScores := make([]weightedRollup, 0, len(evaluated.NamespaceScores)+1)
	for _, namespace := range evaluated.NamespaceScores {
		score, rollupWeight := weightedScoreAndWeight(
			scoreCategories(namespace.Categories),
			namespace.ScoreWeights,
		)
		hasCritical := hasOpenCriticalFinding(evaluated.Findings, namespace.Namespace)
		capped := false
		if hasCritical {
			score, capped = capScore(score, namespace.CriticalCap)
		}
		namespaces = append(namespaces, namespaceScore{
			Namespace:   namespace.Namespace,
			Environment: optionalScoreEnvironment(namespace.Environment),
			Score:       score,
			Capped:      capped,
		})
		if score != nil {
			rollupScores = append(rollupScores, weightedRollup{
				Score:  score,
				Weight: rollupWeight,
			})
		}
	}

	clusterScore, clusterWeight := weightedScoreAndWeight(
		scoreCategories(evaluated.ClusterScores),
		evaluated.ScoreWeights,
	)
	if hasOpenCriticalClusterFinding(evaluated.Findings) {
		clusterScore, _ = capScore(clusterScore, evaluated.CriticalCap)
	}
	if clusterScore != nil {
		rollupScores = append(rollupScores, weightedRollup{
			Score:  clusterScore,
			Weight: clusterWeight,
		})
	}
	overall := rollupScore(rollupScores)
	return scores{Overall: overall, Categories: categories, Namespaces: namespaces}
}

func weightedScoreAndWeight(
	categories []scoreCategory,
	weights map[string]float64,
) (*float64, float64) {
	var weighted, totalWeight float64
	for _, category := range categories {
		weight, exists := weights[category.Category]
		if !exists || weight <= 0 || category.PassRate == nil {
			continue
		}
		weighted += *category.PassRate * weight
		totalWeight += weight
	}
	if totalWeight == 0 {
		return nil, 0
	}
	score := weighted / totalWeight * 100
	return &score, totalWeight
}

type weightedRollup struct {
	Score  *float64
	Weight float64
}

func rollupScore(values []weightedRollup) *float64 {
	if len(values) == 0 {
		return nil
	}
	var weighted, totalWeight float64
	for _, value := range values {
		weighted += *value.Score * value.Weight
		totalWeight += value.Weight
	}
	if totalWeight == 0 {
		return nil
	}
	score := weighted / totalWeight
	return &score
}

func capScore(score *float64, cap float64) (*float64, bool) {
	if score == nil || *score <= cap {
		return score, false
	}
	capped := cap
	return &capped, true
}

func hasOpenCriticalFinding(findings []engine.Finding, namespace string) bool {
	for _, finding := range findings {
		if finding.Status != "open" || finding.Severity != "critical" {
			continue
		}
		if namespace == "" {
			return true
		}
		for _, resource := range finding.Resources {
			if resource.Namespace == namespace || (resource.Kind == "Namespace" && resource.Name == namespace) {
				return true
			}
		}
	}
	return false
}

func hasOpenCriticalClusterFinding(findings []engine.Finding) bool {
	for _, finding := range findings {
		if finding.Status != "open" || finding.Severity != "critical" {
			continue
		}
		clusterScoped := len(finding.Resources) > 0
		for _, resource := range finding.Resources {
			if resource.Namespace != "" || resource.Kind == "Namespace" {
				clusterScoped = false
				break
			}
		}
		if clusterScoped {
			return true
		}
	}
	return false
}

func optionalScoreEnvironment(environment string) *string {
	if environment == "" {
		return nil
	}
	copied := environment
	return &copied
}

type report struct {
	SchemaVersion     string                     `json:"schemaVersion"`
	GeneratedAt       string                     `json:"generatedAt"`
	Scanner           scanner                    `json:"scanner"`
	Scan              scan                       `json:"scan"`
	PermissionSummary []permission               `json:"permissionSummary"`
	Inventory         inventorySummary           `json:"inventory"`
	WorkloadPostures  []canonicalWorkloadPosture `json:"workloadPostures"`
	Findings          []finding                  `json:"findings"`
	Scores            scores                     `json:"scores"`
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
	ClusterContext       string      `json:"clusterContext"`
	Scope                scope       `json:"scope"`
	DataSources          dataSources `json:"dataSources"`
	EnvironmentInference bool        `json:"environmentInference"`
}

type scope struct {
	AllNamespaces bool     `json:"allNamespaces"`
	Namespaces    []string `json:"namespaces,omitempty"`
}

type dataSources struct {
	KubernetesAPI bool         `json:"kubernetesAPI"`
	Prometheus    prometheus   `json:"prometheus"`
	ContextFiles  contextFiles `json:"contextFiles"`
}

type prometheus struct {
	Enabled    bool    `json:"enabled"`
	URL        string  `json:"url,omitempty"`
	Lookback   string  `json:"lookback,omitempty"`
	DegradedTo *string `json:"degradedTo,omitempty"`
}

type contextFiles struct {
	ScanConfig      bool `json:"scanConfig"`
	OwnershipImport bool `json:"ownershipImport"`
	Exceptions      bool `json:"exceptions"`
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
	Counts         map[string]int  `json:"counts"`
	DataPlane      dataPlane       `json:"dataPlane"`
	MultiCluster   multiCluster    `json:"multiCluster"`
	Classification *classification `json:"classification,omitempty"`
}

type classification struct {
	NamespacesClassified   int            `json:"namespacesClassified"`
	NamespacesUnclassified int            `json:"namespacesUnclassified"`
	ByEnvironment          map[string]int `json:"byEnvironment"`
}

type dataPlane struct {
	Mode      string   `json:"mode"`
	Ztunnel   *ztunnel `json:"ztunnel,omitempty"`
	Waypoints *int     `json:"waypoints,omitempty"`
}

type ztunnel struct {
	Present      *bool `json:"present,omitempty"`
	NodesCovered *int  `json:"nodesCovered,omitempty"`
	NodesTotal   *int  `json:"nodesTotal"`
}

type multiCluster struct {
	ParticipationDetected bool     `json:"participationDetected"`
	Evaluated             bool     `json:"evaluated"`
	Signals               []string `json:"signals,omitempty"`
	MeshNetworks          []string `json:"meshNetworks,omitempty"`
}

type finding struct {
	ID              string             `json:"id"`
	ControlID       string             `json:"controlId"`
	Title           string             `json:"title,omitempty"`
	Severity        string             `json:"severity"`
	EvidenceType    string             `json:"evidenceType"`
	Status          string             `json:"status"`
	Confidence      string             `json:"confidence"`
	DataPlaneMode   string             `json:"dataPlaneMode,omitempty"`
	EvidenceSources []string           `json:"evidenceSources,omitempty"`
	Resources       []resourceRef      `json:"resources"`
	ResolutionChain []resolver.Step    `json:"resolutionChain,omitempty"`
	Reasoning       string             `json:"reasoning"`
	Remediation     *remediation       `json:"remediation,omitempty"`
	Exception       *exceptionEvidence `json:"exception,omitempty"`
	UnknownReason   string             `json:"unknownReason,omitempty"`
}

type exceptionEvidence struct {
	ID        string `json:"id"`
	Expired   bool   `json:"expired"`
	ExpiresAt string `json:"expiresAt"`
	Approver  string `json:"approver"`
	Ticket    string `json:"ticket"`
}

type canonicalWorkloadPosture struct {
	Workload              resolver.WorkloadRef   `json:"workload"`
	Environment           *string                `json:"environment"`
	EnvironmentConfidence *string                `json:"environmentConfidence"`
	Owner                 *string                `json:"owner"`
	AppID                 *string                `json:"appId"`
	DataPlaneMode         resolver.DataPlaneMode `json:"dataPlaneMode"`
	MTLS                  resolver.MTLSResult    `json:"mtls"`
	Authorization         resolver.AuthzResult   `json:"authorization"`
	Verified              *verifiedPosture       `json:"verified,omitempty"`
}

type verifiedPosture struct {
	Status            string   `json:"status"`
	Window            string   `json:"window"`
	MTLSTrafficShare  *float64 `json:"mtlsTrafficShare,omitempty"`
	PlaintextObserved *bool    `json:"plaintextObserved,omitempty"`
	PlaintextSources  []string `json:"plaintextSources,omitempty"`
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
	Overall    *float64         `json:"overall"`
	Categories []scoreCategory  `json:"categories"`
	Namespaces []namespaceScore `json:"namespaces,omitempty"`
}

type scoreCategory struct {
	Category  string   `json:"category"`
	Grade     string   `json:"grade"`
	PassRate  *float64 `json:"passRate"`
	Evaluated int      `json:"evaluated,omitempty"`
	Unknown   int      `json:"unknown,omitempty"`
}

type namespaceScore struct {
	Namespace   string   `json:"namespace"`
	Environment *string  `json:"environment,omitempty"`
	Score       *float64 `json:"score"`
	Capped      bool     `json:"capped,omitempty"`
}
