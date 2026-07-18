package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/normalize"
	"github.com/openmeshguard/openmeshguard/internal/output"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	"github.com/spf13/cobra"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type scanOptions struct {
	Kubeconfig    string
	Context       string
	AllNamespaces bool
	Namespaces    []string
	RootNamespace string
	ControlPacks  []string
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
	return nil
}

func runScan(ctx context.Context, info versionInfo, opts scanOptions, stdout io.Writer) error {
	packs, err := engine.LoadPacks(opts.ControlPacks)
	if err != nil {
		return fmt.Errorf("load control packs: %w", err)
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
	namespacesByName := make(map[string]engine.NamespaceInput, len(engineNamespaces))
	for _, namespace := range engineNamespaces {
		namespacesByName[namespace.Name] = namespace
	}
	workloadPostures := make([]resolver.WorkloadResult, 0, len(normalized.Workloads))
	engineWorkloads := make([]engine.WorkloadInput, 0, len(normalized.Workloads))
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
		engineWorkloads = append(engineWorkloads, engine.WorkloadInput{Posture: posture, Namespace: namespacesByName[namespaceName]})
	}
	evaluated, err := engine.Evaluate(packs, engine.Input{
		Workloads:                engineWorkloads,
		Namespaces:               meshNamespaceInputs(engineNamespaces),
		NamespaceTargetsComplete: true,
		InventoryAvailability:    inventoryAvailability(snapshot),
		Inventory: map[string]any{
			"counts": normalized.Inventory.Counts,
			"dataPlane": map[string]any{
				"mode": string(normalized.Inventory.DataPlaneMode),
			},
			"multiCluster": map[string]any{
				"participationDetected": normalized.Inventory.MultiCluster.ParticipationDetected,
				"evaluated":             false,
				"signals":               normalized.Inventory.MultiCluster.Signals,
				"meshNetworks":          normalized.Inventory.MultiCluster.MeshNetworks,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("evaluate controls: %w", err)
	}

	return output.WriteScanJSONWithEvaluation(stdout, output.ScanInput{
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
			if control.Scope == "resource" {
				return fmt.Errorf("%s: control %s: resource scope is unavailable in scan until normalized resource collection is implemented", pack.File, control.ID)
			}
		}
	}
	return nil
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
		return workload.Namespace.AmbientEnrolled
	default:
		return workload.Namespace.AmbientEnrolled
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
	switch observed {
	case resolver.True:
		return "enrolled"
	case resolver.False:
		if current == "" || current == "unknown" {
			return "not-enrolled"
		}
		return current
	default:
		if current == "" {
			return "unknown"
		}
		return current
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
		paths = append(paths, "namespace.labels", "namespace.meshEnrollment")
		return paths, []string{"namespace"}
	case "/pods":
		paths = append(paths, "workload.dataPlaneMode", "workload.mtls")
		return paths, []string{"workload", "namespace"}
	case "apps/deployments", "apps/replicasets", "apps/statefulsets", "apps/daemonsets":
		return paths, []string{"workload"}
	case "security.istio.io/peerauthentications":
		paths = append(paths, "workload.mtls")
	}
	return paths, nil
}

func inventoryPathsForResource(apiGroup, resource string) []string {
	countPaths := map[string]string{
		"/namespaces":                           "counts.namespaces",
		"/pods":                                 "counts.pods",
		"/services":                             "counts.services",
		"apps/deployments":                      "counts.deployments",
		"apps/replicasets":                      "counts.replicasets",
		"apps/statefulsets":                     "counts.statefulsets",
		"apps/daemonsets":                       "counts.daemonsets",
		"security.istio.io/peerauthentications": "counts.peerAuthentications",
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
	if apiGroup == "" && (resource == "namespaces" || resource == "services") {
		paths = append(paths,
			"multiCluster.participationDetected",
			"multiCluster.signals",
			"multiCluster.meshNetworks",
		)
	}
	return paths
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
