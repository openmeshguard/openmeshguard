package context

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

const (
	EnvironmentUnclassified = "unclassified"
	ConfidenceObserved      = "observed"
	ConfidenceResolved      = "resolved"
	ConfidenceInferred      = "inferred"
	ConfidenceUserSupplied  = "user-supplied"
	ConfidenceUnavailable   = "unavailable"
)

var (
	defaultAppIDKeys          = []string{"openmeshguard.io/app-id", "app.kubernetes.io/part-of", "app.kubernetes.io/name"}
	defaultOwnerKeys          = []string{"openmeshguard.io/owner"}
	defaultEnvironmentSources = []ClassificationSource{{
		Type: "namespace-label",
		Keys: []string{"openmeshguard.io/environment", "environment", "env"},
	}}
	inferredEnvironmentRules = []NameRule{
		{Pattern: `-prod$`, Environment: "production"},
		{Pattern: `-production$`, Environment: "production"},
	}
)

type NamespaceInput struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
	LabelsKnown bool
}

type WorkloadInput struct {
	Ref         resolver.WorkloadRef
	Labels      map[string]string
	Annotations map[string]string
	Namespace   NamespaceInput
}

type Classification struct {
	Environment string
	Confidence  string
	Known       bool
	Reason      string
	Source      string
}

type Ownership struct {
	AppID       string
	Owner       string
	AppIDKnown  bool
	OwnerKnown  bool
	AppIDSource string
	OwnerSource string
	AppIDReason string
	OwnerReason string
}

type WorkloadContext struct {
	Ref            resolver.WorkloadRef
	Classification Classification
	Ownership      Ownership
	ExceptionID    string
}

type Result struct {
	Namespaces map[string]Classification
	Workloads  []WorkloadContext
}

type ResolveInput struct {
	ClusterContext    string
	InferEnvironments bool
	Config            ScanConfig
	OwnershipImport   OwnershipImport
	Namespaces        []NamespaceInput
	Workloads         []WorkloadInput
}

func Resolve(input ResolveInput) Result {
	namespaceClassifications := make(map[string]Classification, len(input.Namespaces))
	for _, namespace := range input.Namespaces {
		namespaceClassifications[namespace.Name] = classifyNamespace(namespace, input.ClusterContext, input.Config.Classification, input.InferEnvironments)
	}
	configOwners := applicationsByID(input.Config.Ownership.Applications)
	importOwners := applicationsByID(input.OwnershipImport.Applications)
	appIDKeys := input.Config.Ownership.Labels.AppID
	if len(appIDKeys) == 0 {
		appIDKeys = defaultAppIDKeys
	}
	ownerKeys := input.Config.Ownership.Labels.Owner
	if len(ownerKeys) == 0 {
		ownerKeys = defaultOwnerKeys
	}

	workloads := make([]WorkloadContext, 0, len(input.Workloads))
	for _, workload := range input.Workloads {
		classification, exists := namespaceClassifications[workload.Namespace.Name]
		if !exists {
			classification = classifyNamespace(workload.Namespace, input.ClusterContext, input.Config.Classification, input.InferEnvironments)
			namespaceClassifications[workload.Namespace.Name] = classification
		}
		ownership := resolveOwnership(workload, appIDKeys, ownerKeys, configOwners, importOwners)
		workloads = append(workloads, WorkloadContext{
			Ref:            workload.Ref,
			Classification: classification,
			Ownership:      ownership,
			ExceptionID:    strings.TrimSpace(workload.Annotations["openmeshguard.io/exception"]),
		})
	}
	sort.Slice(workloads, func(i, j int) bool {
		return workloadKey(workloads[i].Ref) < workloadKey(workloads[j].Ref)
	})
	return Result{Namespaces: namespaceClassifications, Workloads: workloads}
}

func classifyNamespace(namespace NamespaceInput, clusterContext string, config ClassificationConfig, infer bool) Classification {
	sources := config.Sources
	if len(sources) == 0 {
		sources = defaultEnvironmentSources
	}
	for _, source := range sources {
		switch source.Type {
		case "namespace-mapping":
			if environment := strings.TrimSpace(source.Mappings[namespace.Name]); environment != "" {
				return Classification{Environment: environment, Confidence: ConfidenceUserSupplied, Known: true, Source: "scan-config"}
			}
		case "namespace-label":
			if !namespace.LabelsKnown {
				return Classification{
					Confidence: ConfidenceUnavailable,
					Reason:     "namespace label evidence unavailable",
				}
			}
			for _, key := range source.Keys {
				if environment := strings.TrimSpace(namespace.Labels[key]); environment != "" {
					return Classification{Environment: environment, Confidence: ConfidenceObserved, Known: true, Source: "kubernetes-api"}
				}
			}
		case "namespace-name":
			if environment := matchNameRules(namespace.Name, source.Rules); environment != "" {
				return Classification{Environment: environment, Confidence: ConfidenceResolved, Known: true, Source: "scan-config"}
			}
		case "cluster":
			return Classification{Environment: source.Environment, Confidence: ConfidenceUserSupplied, Known: true, Source: "scan-config"}
		case "cluster-context":
			if environment := strings.TrimSpace(source.Mappings[clusterContext]); environment != "" {
				return Classification{Environment: environment, Confidence: ConfidenceResolved, Known: true, Source: "scan-config"}
			}
		}
	}
	if infer {
		if environment := matchNameRules(namespace.Name, inferredEnvironmentRules); environment != "" {
			return Classification{Environment: environment, Confidence: ConfidenceInferred, Known: true, Source: "kubernetes-api"}
		}
	}
	return Classification{Environment: EnvironmentUnclassified, Confidence: ConfidenceResolved, Known: true, Source: "kubernetes-api"}
}

func resolveOwnership(
	workload WorkloadInput,
	appIDKeys, ownerKeys []string,
	configOwners, importOwners map[string]string,
) Ownership {
	appID, appIDSource := firstMetadataValue(
		workload.Labels,
		workload.Annotations,
		appIDKeys,
		"workload",
	)
	appIDKnown := true
	appIDReason := ""
	if appID == "" {
		if !workload.Namespace.LabelsKnown {
			appIDKnown = false
			appIDReason = "namespace label and annotation evidence unavailable"
		} else {
			appID, appIDSource = firstMetadataValue(
				workload.Namespace.Labels,
				workload.Namespace.Annotations,
				appIDKeys,
				"namespace",
			)
		}
	}
	owner, ownerSource := firstMetadataValue(
		workload.Labels,
		workload.Annotations,
		ownerKeys,
		"workload",
	)
	ownerKnown := true
	ownerReason := ""
	if owner == "" {
		if !workload.Namespace.LabelsKnown {
			ownerKnown = false
			ownerReason = "namespace label and annotation evidence unavailable"
		} else {
			owner, ownerSource = firstMetadataValue(
				workload.Namespace.Labels,
				workload.Namespace.Annotations,
				ownerKeys,
				"namespace",
			)
		}
	}

	ownership := Ownership{
		AppID:       appID,
		Owner:       owner,
		AppIDKnown:  appIDKnown,
		OwnerKnown:  ownerKnown,
		AppIDSource: appIDSource,
		OwnerSource: ownerSource,
		AppIDReason: appIDReason,
		OwnerReason: ownerReason,
	}
	if ownerKnown && owner == "" && appIDKnown && appID != "" {
		if configuredOwner := configOwners[appID]; configuredOwner != "" {
			ownership.Owner = configuredOwner
			ownership.OwnerSource = "scan-config"
		} else if importedOwner := importOwners[appID]; importedOwner != "" {
			ownership.Owner = importedOwner
			ownership.OwnerSource = "ownership-import"
		}
	}
	return ownership
}

func firstMetadataValue(labels, annotations map[string]string, keys []string, scope string) (string, string) {
	for _, key := range keys {
		if value := strings.TrimSpace(labels[key]); value != "" {
			return value, fmt.Sprintf("%s label %s", scope, key)
		}
	}
	for _, key := range keys {
		if value := strings.TrimSpace(annotations[key]); value != "" {
			return value, fmt.Sprintf("%s annotation %s", scope, key)
		}
	}
	return "", ""
}

func applicationsByID(applications []Application) map[string]string {
	out := make(map[string]string, len(applications))
	for _, application := range applications {
		out[strings.TrimSpace(application.AppID)] = strings.TrimSpace(application.Owner)
	}
	return out
}

func matchNameRules(name string, rules []NameRule) string {
	for _, rule := range rules {
		if regexp.MustCompile(rule.Pattern).MatchString(name) {
			return rule.Environment
		}
	}
	return ""
}

func workloadKey(ref resolver.WorkloadRef) string {
	return ref.Cluster + "/" + ref.Namespace + "/" + ref.Kind + "/" + ref.Name
}

func ClassificationCounts(classifications map[string]Classification) (classified, unclassified int, byEnvironment map[string]int) {
	byEnvironment = map[string]int{}
	for _, classification := range classifications {
		switch {
		case !classification.Known:
			unclassified++
			byEnvironment[ConfidenceUnavailable]++
		case classification.Environment == EnvironmentUnclassified:
			unclassified++
			byEnvironment[EnvironmentUnclassified]++
		default:
			classified++
			byEnvironment[classification.Environment]++
		}
	}
	return classified, unclassified, byEnvironment
}
