package output

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
)

const sarifSchemaURI = "https://docs.oasis-open.org/sarif/sarif/v2.1.0/errata01/os/schemas/sarif-schema-2.1.0.json"

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool       sarifTool      `json:"tool"`
	Results    []sarifResult  `json:"results"`
	Properties map[string]any `json:"properties,omitempty"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string               `json:"id"`
	ShortDescription     sarifMessage         `json:"shortDescription"`
	FullDescription      *sarifMessage        `json:"fullDescription,omitempty"`
	Help                 *sarifMessage        `json:"help,omitempty"`
	DefaultConfiguration sarifReportingConfig `json:"defaultConfiguration"`
	Properties           map[string]any       `json:"properties,omitempty"`
}

type sarifReportingConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID       string             `json:"ruleId"`
	RuleIndex    int                `json:"ruleIndex"`
	Kind         string             `json:"kind"`
	Level        string             `json:"level"`
	Message      sarifMessage       `json:"message"`
	Locations    []sarifLocation    `json:"locations"`
	Fingerprints map[string]string  `json:"fingerprints"`
	Suppressions []sarifSuppression `json:"suppressions,omitempty"`
	Properties   map[string]any     `json:"properties"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation  `json:"physicalLocation"`
	LogicalLocations []sarifLogicalLocation `json:"logicalLocations,omitempty"`
	Properties       map[string]any         `json:"properties,omitempty"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifLogicalLocation struct {
	Name               string `json:"name,omitempty"`
	FullyQualifiedName string `json:"fullyQualifiedName,omitempty"`
	Kind               string `json:"kind,omitempty"`
}

type sarifSuppression struct {
	Kind          string `json:"kind"`
	Status        string `json:"status,omitempty"`
	Justification string `json:"justification,omitempty"`
}

// WriteSARIF renders a SARIF 2.1.0 compatibility projection from canonical JSON.
func WriteSARIF(writer io.Writer, reader io.Reader) error {
	canonical, err := readCanonicalReport(reader)
	if err != nil {
		return err
	}
	projected, err := projectSARIF(canonical)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(projected); err != nil {
		return fmt.Errorf("encode SARIF projection: %w", err)
	}
	return nil
}

func projectSARIF(canonical report) (sarifLog, error) {
	rules, ruleIndexes, err := sarifRules(canonical.Findings)
	if err != nil {
		return sarifLog{}, err
	}
	results := make([]sarifResult, 0, len(canonical.Findings))
	for _, item := range canonical.Findings {
		index, exists := ruleIndexes[item.ControlID]
		if !exists {
			return sarifLog{}, fmt.Errorf("project SARIF: no rule for finding %s", item.ID)
		}
		results = append(results, sarifResultFor(canonical, item, index))
	}
	return sarifLog{
		Schema:  sarifSchemaURI,
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "OpenMeshGuard",
				Version:        canonical.Scanner.Version,
				InformationURI: "https://openmeshguard.io",
				Rules:          rules,
			}},
			Results: results,
			Properties: map[string]any{
				"openmeshguard.schemaVersion":  canonical.SchemaVersion,
				"openmeshguard.generatedAt":    canonical.GeneratedAt,
				"openmeshguard.clusterContext": canonical.Scan.ClusterContext,
			},
		}},
	}, nil
}

func sarifRules(findings []finding) ([]sarifRule, map[string]int, error) {
	byControl := map[string][]finding{}
	for _, item := range findings {
		if item.ControlID == "" {
			return nil, nil, fmt.Errorf("project SARIF: finding %s has no controlId", item.ID)
		}
		byControl[item.ControlID] = append(byControl[item.ControlID], item)
	}
	controlIDs := make([]string, 0, len(byControl))
	for controlID := range byControl {
		controlIDs = append(controlIDs, controlID)
	}
	sort.Strings(controlIDs)

	rules := make([]sarifRule, 0, len(controlIDs))
	indexes := make(map[string]int, len(controlIDs))
	for _, controlID := range controlIDs {
		controlFindings := byControl[controlID]
		title := controlFindings[0].Title
		if title == "" {
			title = controlID
		}
		remediationText := ""
		severity := "info"
		for _, item := range controlFindings {
			if item.Title != "" && item.Title != title {
				return nil, nil, fmt.Errorf(
					"project SARIF: control %s has conflicting titles %q and %q",
					controlID,
					title,
					item.Title,
				)
			}
			if severityRank(item.Severity) > severityRank(severity) {
				severity = item.Severity
			}
			if remediationText == "" && item.Remediation != nil {
				remediationText = item.Remediation.Guidance
			}
		}
		rule := sarifRule{
			ID:                   controlID,
			ShortDescription:     sarifMessage{Text: title},
			DefaultConfiguration: sarifReportingConfig{Level: sarifLevel(severity)},
			Properties:           map[string]any{"openmeshguard.severity": severity},
		}
		if remediationText != "" {
			rule.Help = &sarifMessage{Text: remediationText}
		}
		indexes[controlID] = len(rules)
		rules = append(rules, rule)
	}
	return rules, indexes, nil
}

func sarifResultFor(canonical report, item finding, ruleIndex int) sarifResult {
	kind := "fail"
	level := sarifLevel(item.Severity)
	switch item.Status {
	case "unknown":
		kind = "review"
		level = "warning"
	case "not-applicable":
		kind = "notApplicable"
		level = "none"
	case "excepted":
		kind = "fail"
	}
	result := sarifResult{
		RuleID:       item.ControlID,
		RuleIndex:    ruleIndex,
		Kind:         kind,
		Level:        level,
		Message:      sarifMessage{Text: item.Reasoning},
		Locations:    sarifLocations(canonical.Scan.ClusterContext, item.Resources),
		Fingerprints: map[string]string{"openmeshguardFindingId/v1": item.ID},
		Properties: map[string]any{
			"openmeshguard.findingId":       item.ID,
			"openmeshguard.status":          item.Status,
			"openmeshguard.severity":        item.Severity,
			"openmeshguard.evidenceType":    item.EvidenceType,
			"openmeshguard.confidence":      item.Confidence,
			"openmeshguard.dataPlaneMode":   item.DataPlaneMode,
			"openmeshguard.evidenceSources": item.EvidenceSources,
			"openmeshguard.unknownReason":   item.UnknownReason,
			"openmeshguard.resolutionChain": item.ResolutionChain,
		},
	}
	if item.Exception != nil {
		result.Properties["openmeshguard.exception"] = item.Exception
	}
	if item.Status == "excepted" {
		justification := "Accepted OpenMeshGuard exception"
		if item.Exception != nil {
			justification = "Accepted OpenMeshGuard exception " + item.Exception.ID
			if item.Exception.Ticket != "" {
				justification += " (" + item.Exception.Ticket + ")"
			}
		}
		result.Suppressions = []sarifSuppression{{
			Kind:          "external",
			Status:        "accepted",
			Justification: justification,
		}}
	}
	return result
}

func sarifLocations(clusterContext string, resources []resourceRef) []sarifLocation {
	locations := make([]sarifLocation, 0, len(resources))
	for _, item := range resources {
		pathParts := []string{"clusters", clusterContext}
		if item.Namespace != "" {
			pathParts = append(pathParts, "namespaces", item.Namespace)
		}
		pathParts = append(pathParts, "resources", item.Kind, item.Name)
		artifactURI := (&url.URL{
			Scheme: "kubernetes",
			Path:   "/" + strings.Join(pathParts, "/"),
		}).String()
		fullyQualifiedName := item.Kind + "/" + item.Name
		if item.Namespace != "" {
			fullyQualifiedName = item.Namespace + "/" + fullyQualifiedName
		}
		locations = append(locations, sarifLocation{
			PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: artifactURI},
			},
			LogicalLocations: []sarifLogicalLocation{{
				Name:               item.Name,
				FullyQualifiedName: fullyQualifiedName,
				Kind:               item.Kind,
			}},
			Properties: map[string]any{
				"openmeshguard.apiVersion": item.APIVersion,
				"openmeshguard.namespace":  item.Namespace,
			},
		})
	}
	return locations
}

func sarifLevel(severity string) string {
	switch severity {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	default:
		return "note"
	}
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}
