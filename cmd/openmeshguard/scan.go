package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	governance "github.com/openmeshguard/openmeshguard/internal/context"
	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/normalize"
	"github.com/openmeshguard/openmeshguard/internal/output"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	"github.com/spf13/cobra"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type scanOptions struct {
	Kubeconfig        string
	Context           string
	AllNamespaces     bool
	Namespaces        []string
	RootNamespace     string
	ControlPacks      []string
	ScanConfig        string
	OwnershipImport   string
	Exceptions        []string
	InferEnvironments bool
}

func newScanCommand(info versionInfo) *cobra.Command {
	opts := scanOptions{}
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a cluster and emit canonical JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.normalizeAndValidate(); err != nil {
				return err
			}
			return runScan(cmd.Context(), info, opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.Kubeconfig, "kubeconfig", "", "path to kubeconfig")
	cmd.Flags().StringVar(&opts.Context, "context", "", "kubeconfig context to use")
	cmd.Flags().BoolVar(&opts.AllNamespaces, "all-namespaces", false, "scan all namespaces")
	cmd.Flags().StringArrayVar(&opts.Namespaces, "namespace", nil, "namespace to scan; may be repeated")
	cmd.Flags().StringVar(&opts.RootNamespace, "root-namespace", collect.DefaultRootNamespace, "Istio mesh root namespace")
	cmd.Flags().StringArrayVar(&opts.ControlPacks, "control-pack", nil, "user control pack path; may be repeated")
	cmd.Flags().StringVar(&opts.ScanConfig, "scan-config", "", "governance scan config path")
	cmd.Flags().StringVar(&opts.OwnershipImport, "ownership-import", "", "ownership import YAML or CSV path")
	cmd.Flags().StringArrayVar(&opts.Exceptions, "exceptions", nil, "exception record file or directory; may be repeated")
	cmd.Flags().BoolVar(&opts.InferEnvironments, "infer-environments", false, "infer production from namespace names and disclose inferred confidence")
	return cmd
}

func (o *scanOptions) normalizeAndValidate() error {
	if o.AllNamespaces && len(o.Namespaces) > 0 {
		return fmt.Errorf("choose either --all-namespaces or --namespace, not both")
	}
	namespaces := make([]string, 0, len(o.Namespaces))
	seen := map[string]struct{}{}
	for _, namespace := range o.Namespaces {
		namespace = strings.TrimSpace(namespace)
		if namespace == "" {
			return fmt.Errorf("namespace must not be empty")
		}
		if _, ok := seen[namespace]; ok {
			continue
		}
		seen[namespace] = struct{}{}
		namespaces = append(namespaces, namespace)
	}
	o.Namespaces = namespaces
	o.RootNamespace = strings.TrimSpace(o.RootNamespace)
	if o.RootNamespace == "" {
		return fmt.Errorf("root namespace must not be empty")
	}
	if !o.AllNamespaces && len(o.Namespaces) == 0 {
		return fmt.Errorf("scan scope required: pass --all-namespaces or at least one --namespace")
	}
	controlPacks := make([]string, 0, len(o.ControlPacks))
	for _, path := range o.ControlPacks {
		path = strings.TrimSpace(path)
		if path == "" {
			return fmt.Errorf("control pack path must not be empty")
		}
		controlPacks = append(controlPacks, path)
	}
	o.ControlPacks = controlPacks
	o.ScanConfig = strings.TrimSpace(o.ScanConfig)
	o.OwnershipImport = strings.TrimSpace(o.OwnershipImport)
	exceptions := make([]string, 0, len(o.Exceptions))
	for _, path := range o.Exceptions {
		path = strings.TrimSpace(path)
		if path == "" {
			return fmt.Errorf("exception path must not be empty")
		}
		exceptions = append(exceptions, path)
	}
	o.Exceptions = exceptions
	return nil
}

func runScan(ctx context.Context, info versionInfo, opts scanOptions, stdout io.Writer) error {
	scanConfig, err := governance.LoadScanConfig(opts.ScanConfig)
	if err != nil {
		return fmt.Errorf("load scan config: %w", err)
	}
	ownershipPath, exceptionPaths, err := resolveContextPaths(opts, scanConfig)
	if err != nil {
		return err
	}
	ownershipImport, err := governance.LoadOwnershipImport(ownershipPath)
	if err != nil {
		return fmt.Errorf("load ownership import: %w", err)
	}
	exceptionRecords, err := governance.LoadExceptions(exceptionPaths)
	if err != nil {
		return fmt.Errorf("load exceptions: %w", err)
	}

	packs, err := engine.LoadPacks(opts.ControlPacks)
	if err != nil {
		return fmt.Errorf("load control packs: %w", err)
	}
	exceptionRecords = validateExceptionControlIDs(exceptionRecords, packs)
	controlOverrides, err := engineControlOverrides(scanConfig.Controls.Overrides, packs)
	if err != nil {
		return fmt.Errorf("load scan config: %w", err)
	}
	if opts.ScanConfig != "" {
		packs = append(packs, engine.Pack{
			APIVersion: engine.APIVersion,
			Kind:       engine.Kind,
			Metadata:   engine.Metadata{Name: "scan-config:" + scanConfig.Metadata.Name, Version: scanConfig.Metadata.Version},
			File:       scanConfig.File,
			Source:     engine.SourceUser,
		})
	}
	if err := validateScanControlScopes(packs); err != nil {
		return err
	}
	restConfig, clusterContext, err := clientConfig(opts)
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create Kubernetes client: %w", err)
	}
	istioClient, err := istioclient.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create Istio client: %w", err)
	}
	gatewayClient, err := gatewayclient.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create Gateway API client: %w", err)
	}

	snapshot, err := collect.New(kubeClient, istioClient, gatewayClient).Collect(ctx, collect.Scope{
		AllNamespaces: opts.AllNamespaces,
		Namespaces:    opts.Namespaces,
		RootNamespace: opts.RootNamespace,
	})
	if err != nil {
		return fmt.Errorf("collect cluster resources: %w", err)
	}
	snapshot.PermissionSummary = permissionSummaryWithControls(snapshot.PermissionSummary, packs)

	normalized := normalize.Build(snapshot)
	resolved := resolver.New()
	engineNamespaces := namespaceInputs(snapshot, normalized.Workloads, opts.Namespaces)
	contextResult := governance.Resolve(governance.ResolveInput{
		ClusterContext:    clusterContext,
		InferEnvironments: opts.InferEnvironments,
		Config:            scanConfig,
		OwnershipImport:   ownershipImport,
		Namespaces:        governanceNamespaceInputs(snapshot, engineNamespaces),
		Workloads:         governanceWorkloadInputs(snapshot, normalized.Workloads),
	})
	for index := range engineNamespaces {
		applyNamespaceContext(&engineNamespaces[index], contextResult.Namespaces[engineNamespaces[index].Name])
	}
	namespacesByName := make(map[string]engine.NamespaceInput, len(engineNamespaces))
	for _, namespace := range engineNamespaces {
		namespacesByName[namespace.Name] = namespace
	}
	workloadPostures := make([]resolver.WorkloadResult, 0, len(normalized.Workloads))
	engineWorkloads := make([]engine.WorkloadInput, 0, len(normalized.Workloads))
	workloadContexts := make(map[string]governance.WorkloadContext, len(contextResult.Workloads))
	for _, workloadContext := range contextResult.Workloads {
		workloadContexts[workloadContextKey(workloadContext.Ref)] = workloadContext
	}
	for _, workload := range normalized.Workloads {
		posture := resolver.WorkloadResult{
			Ref:   workload.Ref,
			Mode:  workload.DataPlaneMode,
			MTLS:  resolved.ResolveMTLS(workload),
			Authz: resolved.ResolveAuthz(workload),
		}
		workloadPostures = append(workloadPostures, posture)
		namespaceName := workload.Namespace.Name
		if namespaceName == "" {
			namespaceName = workload.Ref.Namespace
		}
		engineWorkload := engine.WorkloadInput{Posture: posture, Namespace: namespacesByName[namespaceName]}
		applyWorkloadContext(&engineWorkload, workloadContexts[workloadContextKey(workload.Ref)])
		engineWorkloads = append(engineWorkloads, engineWorkload)
	}
	evaluationTime := time.Now().UTC()
	exceptionResources, exceptionInputs, exceptionBindings := engineExceptionInputs(
		exceptionRecords,
		contextResult.Workloads,
		evaluationTime,
	)
	evaluated, err := engine.Evaluate(packs, engine.Input{
		Workloads:                engineWorkloads,
		Namespaces:               meshNamespaceInputs(engineNamespaces),
		NamespaceTargetsComplete: true,
		Resources:                exceptionResources,
		InventoryAvailability:    inventoryAvailability(snapshot),
		Inventory: map[string]any{
			"counts":    normalized.Inventory.Counts,
			"dataPlane": dataPlaneInventory(normalized.Inventory),
			"multiCluster": map[string]any{
				"participationDetected": normalized.Inventory.MultiCluster.ParticipationDetected,
				"evaluated":             false,
				"signals":               normalized.Inventory.MultiCluster.Signals,
				"meshNetworks":          normalized.Inventory.MultiCluster.MeshNetworks,
			},
		},
		Params:            scanConfig.Controls.Parameters.Defaults,
		EnvironmentParams: scanConfig.Controls.Parameters.Environments,
		ControlOverrides:  controlOverrides,
	})
	if err != nil {
		return fmt.Errorf("evaluate controls: %w", err)
	}
	evaluated = engine.ApplyExceptions(evaluated, exceptionInputs, exceptionBindings)
	classified, unclassified, byEnvironment := governance.ClassificationCounts(contextResult.Namespaces)
	evaluated.Context = reportContext(
		contextResult,
		classified,
		unclassified,
		byEnvironment,
		opts.InferEnvironments,
		opts.ScanConfig != "",
		ownershipPath != "",
		len(exceptionPaths) > 0,
	)

	return output.WriteScanJSONWithEvaluation(stdout, output.ScanInput{
		GeneratedAt:       evaluationTime,
		ScannerVersion:    info.Version,
		ResolverVersion:   resolved.Version(),
		ClusterContext:    clusterContext,
		Scope:             outputScope(opts),
		PermissionSummary: snapshot.PermissionSummary,
		Inventory:         normalized.Inventory,
		WorkloadPostures:  workloadPostures,
	}, packs, evaluated)
}

func validateScanControlScopes(packs []engine.Pack) error {
	for _, pack := range packs {
		for _, control := range pack.Controls {
			if control.Scope == "resource" && !contextResourceControl(control) {
				return fmt.Errorf("%s: control %s: resource scope is unavailable in scan until normalized resource collection is implemented", pack.File, control.ID)
			}
		}
	}
	return nil
}

func contextResourceControl(control engine.Control) bool {
	if len(control.Match.APIGroups) != 1 || control.Match.APIGroups[0] != "openmeshguard.io" {
		return false
	}
	for _, kind := range control.Match.Kinds {
		if kind != governance.ExceptionKind && kind != "ExceptionReference" {
			return false
		}
	}
	return len(control.Match.Kinds) > 0
}

func resolveContextPaths(opts scanOptions, config governance.ScanConfig) (string, []string, error) {
	ownershipPath := opts.OwnershipImport
	if ownershipPath != "" && config.Inputs.OwnershipImport != "" {
		return "", nil, fmt.Errorf("ownership import is set by both --ownership-import and scan config inputs.ownershipImport")
	}
	if ownershipPath == "" {
		ownershipPath = config.Inputs.OwnershipImport
	}
	exceptionPaths := append([]string(nil), opts.Exceptions...)
	if len(exceptionPaths) > 0 && len(config.Inputs.Exceptions) > 0 {
		return "", nil, fmt.Errorf("exceptions are set by both --exceptions and scan config inputs.exceptions")
	}
	if len(exceptionPaths) == 0 {
		exceptionPaths = append(exceptionPaths, config.Inputs.Exceptions...)
	}
	return ownershipPath, exceptionPaths, nil
}

func engineControlOverrides(overrides []governance.ControlOverride, packs []engine.Pack) (map[string]engine.ControlOverride, error) {
	known := map[string]struct{}{}
	for _, pack := range packs {
		for _, control := range pack.Controls {
			known[control.ID] = struct{}{}
		}
	}
	out := make(map[string]engine.ControlOverride, len(overrides))
	for _, override := range overrides {
		if _, exists := known[override.ControlID]; !exists {
			return nil, fmt.Errorf("controls.overrides references unknown control %s", override.ControlID)
		}
		var environments *[]string
		if override.Environments != nil {
			copied := append([]string(nil), (*override.Environments)...)
			environments = &copied
		}
		severities := make(map[string]string, len(override.SeverityByEnvironment))
		for environment, severity := range override.SeverityByEnvironment {
			severities[environment] = severity
		}
		out[override.ControlID] = engine.ControlOverride{
			Environments:          environments,
			SeverityByEnvironment: severities,
		}
	}
	return out, nil
}

func governanceNamespaceInputs(snapshot collect.Snapshot, namespaces []engine.NamespaceInput) []governance.NamespaceInput {
	out := make([]governance.NamespaceInput, 0, len(namespaces))
	for _, namespace := range namespaces {
		out = append(out, governance.NamespaceInput{
			Name:        namespace.Name,
			Labels:      namespace.Labels,
			Annotations: namespaceAnnotations(snapshot, namespace.Name),
			LabelsKnown: namespaceLabelsKnown(namespace),
		})
	}
	return out
}

func governanceWorkloadInputs(snapshot collect.Snapshot, workloads []resolver.WorkloadInput) []governance.WorkloadInput {
	labelsKnown := namespacesKnownForContext(snapshot)
	out := make([]governance.WorkloadInput, 0, len(workloads))
	for _, workload := range workloads {
		namespace := workload.Namespace.Name
		if namespace == "" {
			namespace = workload.Ref.Namespace
		}
		metadata := collectedWorkloadMetadata(snapshot, workload.Ref, 0)
		out = append(out, governance.WorkloadInput{
			Ref:         workload.Ref,
			Labels:      mergeStringValues(workload.Labels, metadata.Labels),
			Annotations: metadata.Annotations,
			Namespace: governance.NamespaceInput{
				Name:        namespace,
				Labels:      workload.Namespace.Labels,
				Annotations: namespaceAnnotations(snapshot, namespace),
				LabelsKnown: labelsKnown,
			},
		})
	}
	return out
}

func namespacesKnownForContext(snapshot collect.Snapshot) bool {
	for _, permission := range snapshot.PermissionSummary {
		if permission.APIGroup == "" && permission.Resource == "namespaces" && !permission.Granted {
			return false
		}
	}
	return true
}

func namespaceLabelsKnown(namespace engine.NamespaceInput) bool {
	availability, exists := namespace.Availability["labels"]
	return !exists || availability.Available
}

type workloadMetadata struct {
	Labels      map[string]string
	Annotations map[string]string
}

func collectedWorkloadMetadata(snapshot collect.Snapshot, ref resolver.WorkloadRef, depth int) workloadMetadata {
	if depth > 2 {
		return workloadMetadata{}
	}
	switch ref.Kind {
	case "Deployment":
		for _, resource := range snapshot.Deployments {
			if resource.Namespace == ref.Namespace && resource.Name == ref.Name {
				return mergeWorkloadMetadata(
					workloadMetadata{
						Labels:      copyStringValues(resource.Spec.Template.Labels),
						Annotations: copyStringValues(resource.Spec.Template.Annotations),
					},
					workloadMetadata{
						Labels:      copyStringValues(resource.Labels),
						Annotations: copyStringValues(resource.Annotations),
					},
				)
			}
		}
	case "ReplicaSet":
		for _, resource := range snapshot.ReplicaSets {
			if resource.Namespace == ref.Namespace && resource.Name == ref.Name {
				parent := ownerWorkloadMetadata(snapshot, resource.Namespace, resource.OwnerReferences, depth+1)
				current := mergeWorkloadMetadata(
					workloadMetadata{
						Labels:      copyStringValues(resource.Spec.Template.Labels),
						Annotations: copyStringValues(resource.Spec.Template.Annotations),
					},
					workloadMetadata{
						Labels:      copyStringValues(resource.Labels),
						Annotations: copyStringValues(resource.Annotations),
					},
				)
				return mergeWorkloadMetadata(current, parent)
			}
		}
	case "StatefulSet":
		for _, resource := range snapshot.StatefulSets {
			if resource.Namespace == ref.Namespace && resource.Name == ref.Name {
				return mergeWorkloadMetadata(
					workloadMetadata{
						Labels:      copyStringValues(resource.Spec.Template.Labels),
						Annotations: copyStringValues(resource.Spec.Template.Annotations),
					},
					workloadMetadata{
						Labels:      copyStringValues(resource.Labels),
						Annotations: copyStringValues(resource.Annotations),
					},
				)
			}
		}
	case "DaemonSet":
		for _, resource := range snapshot.DaemonSets {
			if resource.Namespace == ref.Namespace && resource.Name == ref.Name {
				return mergeWorkloadMetadata(
					workloadMetadata{
						Labels:      copyStringValues(resource.Spec.Template.Labels),
						Annotations: copyStringValues(resource.Spec.Template.Annotations),
					},
					workloadMetadata{
						Labels:      copyStringValues(resource.Labels),
						Annotations: copyStringValues(resource.Annotations),
					},
				)
			}
		}
	case "Pod":
		for _, resource := range snapshot.Pods {
			if resource.Namespace == ref.Namespace && resource.Name == ref.Name {
				parent := ownerWorkloadMetadata(snapshot, resource.Namespace, resource.OwnerReferences, depth+1)
				return mergeWorkloadMetadata(workloadMetadata{
					Labels:      copyStringValues(resource.Labels),
					Annotations: copyStringValues(resource.Annotations),
				}, parent)
			}
		}
	}
	return workloadMetadata{}
}

func ownerWorkloadMetadata(
	snapshot collect.Snapshot,
	namespace string,
	owners []metav1.OwnerReference,
	depth int,
) workloadMetadata {
	for _, owner := range owners {
		if owner.Controller == nil || !*owner.Controller {
			continue
		}
		if metadata, matched := knownOwnerWorkloadMetadata(snapshot, namespace, owner, depth); matched {
			return metadata
		}
	}
	for _, owner := range owners {
		if metadata, matched := knownOwnerWorkloadMetadata(snapshot, namespace, owner, depth); matched {
			return metadata
		}
	}
	return workloadMetadata{}
}

func knownOwnerWorkloadMetadata(
	snapshot collect.Snapshot,
	namespace string,
	owner metav1.OwnerReference,
	depth int,
) (workloadMetadata, bool) {
	switch owner.Kind {
	case "Deployment", "ReplicaSet", "StatefulSet", "DaemonSet":
		return collectedWorkloadMetadata(snapshot, resolver.WorkloadRef{
			Kind:      owner.Kind,
			Namespace: namespace,
			Name:      owner.Name,
		}, depth), true
	default:
		return workloadMetadata{}, false
	}
}

func mergeWorkloadMetadata(base, override workloadMetadata) workloadMetadata {
	return workloadMetadata{
		Labels:      mergeStringValues(base.Labels, override.Labels),
		Annotations: mergeStringValues(base.Annotations, override.Annotations),
	}
}

func mergeStringValues(base, override map[string]string) map[string]string {
	out := copyStringValues(base)
	if out == nil && override != nil {
		out = make(map[string]string, len(override))
	}
	for key, value := range override {
		out[key] = value
	}
	return out
}

func namespaceAnnotations(snapshot collect.Snapshot, name string) map[string]string {
	for _, namespace := range snapshot.Namespaces {
		if namespace.Name == name {
			return copyStringValues(namespace.Annotations)
		}
	}
	return nil
}

func copyStringValues(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func applyNamespaceContext(namespace *engine.NamespaceInput, classification governance.Classification) {
	namespace.Environment = classification.Environment
	namespace.EnvironmentConfidence = classification.Confidence
	namespace.EnvironmentKnown = classification.Known
	if classification.Source != "" {
		namespace.EvidenceSources = append(namespace.EvidenceSources, classification.Source)
	}
	if !classification.Known {
		if namespace.Availability == nil {
			namespace.Availability = map[string]engine.Availability{}
		}
		namespace.Availability["environment"] = engine.Availability{Reason: classification.Reason}
	}
}

func applyWorkloadContext(workload *engine.WorkloadInput, resolved governance.WorkloadContext) {
	workload.Environment = resolved.Classification.Environment
	workload.EnvironmentConfidence = resolved.Classification.Confidence
	workload.EnvironmentKnown = resolved.Classification.Known
	workload.Owner = resolved.Ownership.Owner
	workload.OwnerKnown = resolved.Ownership.OwnerKnown
	workload.AppID = resolved.Ownership.AppID
	workload.AppIDKnown = resolved.Ownership.AppIDKnown
	workload.EvidenceSources = contextEvidenceSources(resolved)
	if workload.Availability == nil {
		workload.Availability = map[string]engine.Availability{}
	}
	if !workload.EnvironmentKnown {
		workload.Availability["environment"] = engine.Availability{Reason: resolved.Classification.Reason}
	}
	if !workload.OwnerKnown {
		workload.Availability["owner"] = engine.Availability{Reason: resolved.Ownership.OwnerReason}
	}
	if !workload.AppIDKnown {
		workload.Availability["appId"] = engine.Availability{Reason: resolved.Ownership.AppIDReason}
	}
}

func contextEvidenceSources(resolved governance.WorkloadContext) []string {
	values := []string{resolved.Classification.Source}
	for _, source := range []string{resolved.Ownership.AppIDSource, resolved.Ownership.OwnerSource} {
		switch {
		case source == "scan-config":
			values = append(values, "scan-config")
		case source == "ownership-import":
			values = append(values, "ownership-import")
		case strings.Contains(source, "label"), strings.Contains(source, "annotation"):
			values = append(values, "kubernetes-api")
		}
	}
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func engineExceptionInputs(
	records []governance.ExceptionRecord,
	workloads []governance.WorkloadContext,
	now time.Time,
) ([]engine.ResourceInput, []engine.ExceptionInput, []engine.ExceptionBinding) {
	byID := make(map[string]governance.ExceptionRecord, len(records))
	resources := make([]engine.ResourceInput, 0, len(records))
	exceptions := make([]engine.ExceptionInput, 0, len(records))
	for _, record := range records {
		byID[record.Metadata.Name] = record
		expired := len(record.ValidationErrors) == 0 && !record.ExpiresAt.After(now)
		validationErrors := make([]any, 0, len(record.ValidationErrors))
		for _, validationError := range record.ValidationErrors {
			validationErrors = append(validationErrors, validationError)
		}
		valid := len(validationErrors) == 0
		resources = append(resources, engine.ResourceInput{
			APIVersion: governance.APIVersion,
			Kind:       governance.ExceptionKind,
			Name:       record.Metadata.Name,
			Fields: map[string]any{
				"validationErrors": validationErrors,
				"valid":            valid,
				"expired":          expired,
			},
			EvidenceSources: []string{"exception-record"},
		})
		exceptions = append(exceptions, engine.ExceptionInput{
			ID:         record.Metadata.Name,
			Owner:      record.Spec.Owner,
			ControlIDs: append([]string(nil), record.Spec.ControlIDs...),
			Valid:      valid,
			Expired:    expired,
			ExpiresAt:  record.ExpiresAt,
			Approver:   record.Spec.Approver,
			Ticket:     record.Spec.Ticket,
		})
	}

	bindings := make([]engine.ExceptionBinding, 0, len(workloads))
	for _, workload := range workloads {
		if workload.ExceptionID == "" {
			continue
		}
		resource := engine.ResourceRef{
			Kind:      workload.Ref.Kind,
			Namespace: workload.Ref.Namespace,
			Name:      workload.Ref.Name,
		}
		bindings = append(bindings, engine.ExceptionBinding{
			Resource:    resource,
			ExceptionID: workload.ExceptionID,
			Owner:       workload.Ownership.Owner,
			OwnerKnown:  workload.Ownership.OwnerKnown,
		})
		record, exists := byID[workload.ExceptionID]
		if !exists {
			resources = append(resources, exceptionReferenceInput(
				workload,
				[]any{"annotation references an exception record that was not loaded"},
				"",
				false,
			))
			continue
		}
		if len(record.ValidationErrors) > 0 {
			continue
		}
		if !workload.Ownership.OwnerKnown {
			reason := workload.Ownership.OwnerReason
			if reason == "" {
				reason = "workload owner evidence unavailable"
			}
			resources = append(resources, exceptionReferenceInput(workload, []any{}, reason, true))
			continue
		}
		if strings.TrimSpace(workload.Ownership.Owner) == "" ||
			strings.TrimSpace(record.Spec.Owner) != strings.TrimSpace(workload.Ownership.Owner) {
			resources = append(resources, exceptionReferenceInput(
				workload,
				[]any{fmt.Sprintf(
					"exception owner %q does not match workload owner %q",
					record.Spec.Owner,
					workload.Ownership.Owner,
				)},
				"",
				true,
			))
		}
	}
	return resources, exceptions, bindings
}

func exceptionReferenceInput(
	workload governance.WorkloadContext,
	validationErrors []any,
	unknownReason string,
	recordLoaded bool,
) engine.ResourceInput {
	input := engine.ResourceInput{
		APIVersion: governance.APIVersion,
		Kind:       "ExceptionReference",
		Namespace:  workload.Ref.Namespace,
		Name: workload.ExceptionID + "@" +
			workload.Ref.Kind + "/" + workload.Ref.Namespace + "/" + workload.Ref.Name,
		Fields: map[string]any{
			"validationErrors": validationErrors,
		},
		EvidenceSources: []string{"kubernetes-api"},
	}
	if recordLoaded {
		input.EvidenceSources = append(input.EvidenceSources, "exception-record")
	}
	if unknownReason != "" {
		input.Availability = map[string]engine.Availability{
			"validationErrors": {Reason: unknownReason},
		}
	}
	return input
}

func validateExceptionControlIDs(
	records []governance.ExceptionRecord,
	packs []engine.Pack,
) []governance.ExceptionRecord {
	known := make(map[string]struct{})
	for _, pack := range packs {
		for _, control := range pack.Controls {
			known[control.ID] = struct{}{}
		}
	}
	out := append([]governance.ExceptionRecord(nil), records...)
	for index := range out {
		out[index].ValidationErrors = append([]string(nil), records[index].ValidationErrors...)
		for _, controlID := range out[index].Spec.ControlIDs {
			if _, exists := known[controlID]; exists {
				continue
			}
			out[index].ValidationErrors = append(
				out[index].ValidationErrors,
				fmt.Sprintf("spec.controlIds references unknown control %q", controlID),
			)
		}
		sort.Strings(out[index].ValidationErrors)
	}
	return out
}

func reportContext(
	resolved governance.Result,
	classified, unclassified int,
	byEnvironment map[string]int,
	inference, scanConfig, ownershipImport, exceptions bool,
) engine.ReportContext {
	workloads := make([]engine.WorkloadContext, 0, len(resolved.Workloads))
	for _, workload := range resolved.Workloads {
		workloads = append(workloads, engine.WorkloadContext{
			Ref:                   workload.Ref,
			Environment:           workload.Classification.Environment,
			EnvironmentConfidence: workload.Classification.Confidence,
			EnvironmentKnown:      workload.Classification.Known,
			Owner:                 workload.Ownership.Owner,
			OwnerKnown:            workload.Ownership.OwnerKnown,
			AppID:                 workload.Ownership.AppID,
			AppIDKnown:            workload.Ownership.AppIDKnown,
		})
	}
	return engine.ReportContext{
		EnvironmentInference: inference,
		ScanConfig:           scanConfig,
		OwnershipImport:      ownershipImport,
		Exceptions:           exceptions,
		Classification: engine.ClassificationSummary{
			NamespacesClassified:   classified,
			NamespacesUnclassified: unclassified,
			ByEnvironment:          byEnvironment,
		},
		Workloads: workloads,
	}
}

func workloadContextKey(ref resolver.WorkloadRef) string {
	return strings.Join([]string{ref.Cluster, ref.Namespace, ref.Kind, ref.Name}, "/")
}

func namespaceInputs(snapshot collect.Snapshot, workloads []resolver.WorkloadInput, requested []string) []engine.NamespaceInput {
	labelsAvailable := true
	for _, permission := range snapshot.PermissionSummary {
		if permission.APIGroup == "" && permission.Resource == "namespaces" && !permission.Granted {
			labelsAvailable = false
			break
		}
	}

	byName := map[string]engine.NamespaceInput{}
	for _, name := range requested {
		input := engine.NamespaceInput{Name: name}
		if !labelsAvailable {
			input.Availability = map[string]engine.Availability{
				"labels": {Reason: "namespace list permission unavailable"},
			}
		}
		byName[name] = input
	}
	for _, namespace := range snapshot.Namespaces {
		input := engine.NamespaceInput{
			Name:           namespace.Name,
			Labels:         namespace.Labels,
			MeshEnrollment: namespaceMeshEnrollment(namespace.Labels),
		}
		if !labelsAvailable {
			input.Availability = map[string]engine.Availability{
				"labels": {Reason: "namespace list permission unavailable"},
			}
		}
		byName[input.Name] = input
	}
	for _, workload := range workloads {
		name := workload.Namespace.Name
		if name == "" {
			name = workload.Ref.Namespace
		}
		input, exists := byName[name]
		if !exists {
			input = engine.NamespaceInput{Name: name, Labels: workload.Namespace.Labels}
		}
		input.MeshEnrollment = mergeMeshEnrollment(input.MeshEnrollment, workloadEnrollmentObservation(workload))
		if !labelsAvailable {
			input.Availability = map[string]engine.Availability{
				"labels": {Reason: "namespace list permission unavailable"},
			}
		}
		byName[name] = input
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]engine.NamespaceInput, 0, len(names))
	for _, name := range names {
		out = append(out, byName[name])
	}
	return out
}

func workloadEnrollmentObservation(workload resolver.WorkloadInput) resolver.Tristate {
	switch workload.DataPlaneMode {
	case resolver.ModeSidecar, resolver.ModeAmbient, resolver.ModeMixed:
		return resolver.True
	case resolver.ModeNotApplicable:
		return resolver.False
	case resolver.ModeUnknown:
		if workload.Namespace.AmbientEnrolled == resolver.True {
			return resolver.True
		}
		return resolver.Unobserved
	default:
		return resolver.Unobserved
	}
}

func meshNamespaceInputs(namespaces []engine.NamespaceInput) []engine.NamespaceInput {
	out := make([]engine.NamespaceInput, 0, len(namespaces))
	for _, namespace := range namespaces {
		if namespace.MeshEnrollment != "not-enrolled" {
			out = append(out, namespace)
		}
	}
	return out
}

func mergeMeshEnrollment(current string, observed resolver.Tristate) string {
	if current == "enrolled" {
		return current
	}
	switch observed {
	case resolver.True:
		return "enrolled"
	case resolver.False:
		if current == "" {
			return "not-enrolled"
		}
		return current
	default:
		return "unknown"
	}
}

func inventoryAvailability(snapshot collect.Snapshot) map[string]engine.Availability {
	reasons := map[string][]string{}
	for _, permission := range snapshot.PermissionSummary {
		if permission.Granted {
			continue
		}
		reason := "list permission unavailable for " + permission.Resource
		if permission.APIGroup != "" {
			reason += "." + permission.APIGroup
		}
		if len(permission.DeniedScopes) > 0 {
			scopes := append([]string(nil), permission.DeniedScopes...)
			sort.Strings(scopes)
			reason += " in " + strings.Join(scopes, ", ")
		}
		for _, path := range inventoryPathsForResource(permission.APIGroup, permission.Resource) {
			reasons[path] = append(reasons[path], reason)
		}
	}

	availability := make(map[string]engine.Availability, len(reasons))
	for path, pathReasons := range reasons {
		sort.Strings(pathReasons)
		availability[path] = engine.Availability{Reason: strings.Join(pathReasons, "; ")}
	}
	return availability
}

func permissionSummaryWithControls(permissions []collect.Permission, packs []engine.Pack) []collect.Permission {
	out := append([]collect.Permission(nil), permissions...)
	for index := range out {
		paths, scopes := permissionEvidenceImpact(out[index])
		out[index].AffectedControls = engine.AffectedControlIDs(packs, paths, scopes)
	}
	return out
}

func permissionEvidenceImpact(permission collect.Permission) ([]string, []string) {
	paths := make([]string, 0, 6)
	for _, path := range inventoryPathsForResource(permission.APIGroup, permission.Resource) {
		paths = append(paths, "inventory."+path)
	}
	key := permission.APIGroup + "/" + permission.Resource
	switch key {
	case "/namespaces":
		paths = append(paths,
			"namespace.labels",
			"namespace.environment",
			"namespace.meshEnrollment",
			"workload.environment",
			"workload.owner",
			"workload.appId",
		)
		return paths, []string{"namespace", "workload"}
	case "/pods":
		paths = append(paths, "workload.dataPlaneMode", "workload.mtls", "workload.authorization")
		return paths, []string{"workload", "namespace"}
	case "/services":
		paths = append(paths, "workload.mtls.byPort", "workload.mtls.clientTLSContradiction", "workload.authorization")
	case "discovery.k8s.io/endpointslices":
		paths = append(paths, "workload.mtls.byPort", "workload.mtls.clientTLSContradiction", "workload.authorization")
	case "apps/deployments", "apps/replicasets", "apps/statefulsets", "apps/daemonsets":
		return paths, []string{"workload"}
	case "security.istio.io/peerauthentications":
		paths = append(paths, "workload.mtls")
	case "networking.istio.io/destinationrules", "networking.istio.io/sidecars":
		paths = append(paths, "workload.mtls.clientTLSContradiction")
	case "security.istio.io/authorizationpolicies", "gateway.networking.k8s.io/gateways":
		paths = append(paths, "workload.authorization")
	}
	return paths, nil
}

func inventoryPathsForResource(apiGroup, resource string) []string {
	countPaths := map[string]string{
		"/namespaces":                             "counts.namespaces",
		"/nodes":                                  "counts.nodes",
		"/pods":                                   "counts.pods",
		"/services":                               "counts.services",
		"discovery.k8s.io/endpointslices":         "counts.endpointSlices",
		"apps/deployments":                        "counts.deployments",
		"apps/replicasets":                        "counts.replicasets",
		"apps/statefulsets":                       "counts.statefulsets",
		"apps/daemonsets":                         "counts.daemonsets",
		"security.istio.io/peerauthentications":   "counts.peerAuthentications",
		"networking.istio.io/destinationrules":    "counts.destinationRules",
		"networking.istio.io/sidecars":            "counts.sidecars",
		"security.istio.io/authorizationpolicies": "counts.authorizationPolicies",
		"gateway.networking.k8s.io/gateways":      "counts.gateways",
	}
	key := apiGroup + "/" + resource
	var paths []string
	if countPath, exists := countPaths[key]; exists {
		paths = append(paths, countPath)
	}
	if (apiGroup == "" && (resource == "namespaces" || resource == "pods")) ||
		(apiGroup == "apps" && (resource == "deployments" || resource == "replicasets" || resource == "statefulsets" || resource == "daemonsets")) {
		paths = append(paths, "dataPlane.mode")
	}
	if apiGroup == "" && resource == "nodes" {
		paths = append(paths, "dataPlane.ztunnel.nodesTotal")
	}
	if apiGroup == "" && resource == "pods" {
		paths = append(paths, "dataPlane.ztunnel.nodesCovered")
	}
	if apiGroup == "apps" && resource == "daemonsets" {
		paths = append(paths, "dataPlane.ztunnel.present", "dataPlane.ztunnel.nodesCovered")
	}
	if apiGroup == "gateway.networking.k8s.io" && resource == "gateways" {
		paths = append(paths, "dataPlane.waypoints")
	}
	if apiGroup == "" && (resource == "namespaces" || resource == "services") {
		paths = append(paths,
			"multiCluster.participationDetected",
			"multiCluster.signals",
			"multiCluster.meshNetworks",
		)
	}
	return paths
}

func dataPlaneInventory(inventory normalize.Inventory) map[string]any {
	mode := string(inventory.DataPlaneMode)
	if mode == string(resolver.ModeNotApplicable) || mode == "" {
		mode = string(resolver.ModeUnknown)
	}
	ztunnel := map[string]any{"nodesTotal": nil}
	if inventory.Ztunnel.Present != nil {
		ztunnel["present"] = *inventory.Ztunnel.Present
	}
	if inventory.Ztunnel.NodesCovered != nil {
		ztunnel["nodesCovered"] = *inventory.Ztunnel.NodesCovered
	}
	if inventory.Ztunnel.NodesTotal != nil {
		ztunnel["nodesTotal"] = *inventory.Ztunnel.NodesTotal
	}
	value := map[string]any{
		"mode":    mode,
		"ztunnel": ztunnel,
	}
	if inventory.Waypoints != nil {
		value["waypoints"] = *inventory.Waypoints
	}
	return value
}

func namespaceMeshEnrollment(labels map[string]string) string {
	if labels["istio.io/dataplane-mode"] == "ambient" || labels["istio-injection"] == "enabled" || labels["istio.io/rev"] != "" {
		return "enrolled"
	}
	return "not-enrolled"
}

func clientConfig(opts scanOptions) (*rest.Config, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.Kubeconfig != "" {
		loadingRules.ExplicitPath = opts.Kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: opts.Context}
	deferred := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	clusterContext := opts.Context
	if rawConfig, err := deferred.RawConfig(); err == nil && clusterContext == "" {
		clusterContext = rawConfig.CurrentContext
	}
	if clusterContext == "" {
		clusterContext = "in-cluster"
	}

	config, err := deferred.ClientConfig()
	if err == nil {
		return config, clusterContext, nil
	}
	if opts.Kubeconfig == "" && opts.Context == "" {
		inCluster, inClusterErr := rest.InClusterConfig()
		if inClusterErr == nil {
			return inCluster, "in-cluster", nil
		}
	}
	return nil, "", fmt.Errorf("build Kubernetes client config: %w", err)
}

func outputScope(opts scanOptions) output.ScanScope {
	return output.ScanScope{
		AllNamespaces: opts.AllNamespaces,
		Namespaces:    append([]string(nil), opts.Namespaces...),
	}
}
