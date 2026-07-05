package output

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/collect"
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
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(buildReport(input)); err != nil {
		return fmt.Errorf("encode canonical report: %w", err)
	}
	return nil
}

func buildReport(input ScanInput) report {
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
			ControlPacks: []controlPack{{
				Name:    "builtin-empty",
				Version: "0.0.0",
				Source:  "builtin",
			}},
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
		Findings:          provisionalFindings(input.WorkloadPostures),
		Scores: scores{
			Overall:    nil,
			Categories: []scoreCategory{},
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

// PROVISIONAL: replaced by CEL engine in M3.
func provisionalFindings(workloads []resolver.WorkloadResult) []finding {
	findings := []finding{}
	for _, workload := range workloads {
		dataPlaneUnavailable := workload.Mode == resolver.ModeUnknown ||
			workload.Mode == resolver.ModeMixed ||
			workload.Mode == resolver.ModeNotApplicable
		status := "open"
		confidence := "resolved"
		var unknownReasons []string
		var title, reasoning string
		switch workload.MTLS.Effective {
		case resolver.MTLSPermissive:
			title = "Effective mTLS is permissive"
			reasoning = fmt.Sprintf(
				"%s/%s resolves to PERMISSIVE mTLS, so plaintext may be accepted by the workload.",
				workload.Ref.Namespace,
				workload.Ref.Name,
			)
		case resolver.MTLSDisabled:
			title = "Effective mTLS is disabled"
			reasoning = fmt.Sprintf(
				"%s/%s resolves to DISABLED mTLS, so plaintext is accepted by the workload.",
				workload.Ref.Namespace,
				workload.Ref.Name,
			)
		case resolver.MTLSUnknown:
			status = "unknown"
			confidence = "unavailable"
			title = "Effective mTLS is unknown"
			reasoning = fmt.Sprintf(
				"%s/%s mTLS posture could not be fully resolved.",
				workload.Ref.Namespace,
				workload.Ref.Name,
			)
			if workload.MTLS.UnknownReason != "" {
				unknownReasons = append(unknownReasons, workload.MTLS.UnknownReason)
			}
		default:
			if !dataPlaneUnavailable {
				continue
			}
			status = "unknown"
			confidence = "unavailable"
			title = "Effective mTLS is unknown"
			reasoning = fmt.Sprintf(
				"%s/%s mTLS posture could not be fully resolved.",
				workload.Ref.Namespace,
				workload.Ref.Name,
			)
		}
		if dataPlaneUnavailable {
			status = "unknown"
			confidence = "unavailable"
			unknownReasons = append(unknownReasons, "data plane membership unavailable")
		}
		unknownReason := strings.Join(uniqueStrings(unknownReasons), "; ")
		if status == "unknown" {
			reasoning = fmt.Sprintf("%s/%s mTLS posture could not be fully resolved.", workload.Ref.Namespace, workload.Ref.Name)
		}
		findings = append(findings, finding{
			ID:            findingID("MG-MTLS-001", workload.Ref),
			ControlID:     "MG-MTLS-001",
			Title:         title,
			Severity:      "medium",
			EvidenceType:  "config",
			Status:        status,
			Confidence:    confidence,
			DataPlaneMode: string(workload.Mode),
			EvidenceSources: []string{
				"kubernetes-api",
				"istio-crd",
			},
			Resources: []resourceRef{{
				Kind:      workload.Ref.Kind,
				Namespace: workload.Ref.Namespace,
				Name:      workload.Ref.Name,
			}},
			ResolutionChain: workload.MTLS.Chain,
			Reasoning:       reasoning,
			UnknownReason:   unknownReason,
		})
	}
	return findings
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func findingID(controlID string, workload resolver.WorkloadRef) string {
	hash := sha256.Sum256([]byte(controlID + "|" + workload.Namespace + "|" + workload.Kind + "|" + workload.Name))
	return controlID + "-" + hex.EncodeToString(hash[:])[:12]
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
	UnknownReason   string          `json:"unknownReason,omitempty"`
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
	Category string   `json:"category"`
	Grade    string   `json:"grade"`
	PassRate *float64 `json:"passRate"`
}
