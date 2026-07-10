package main

import (
	"context"
	"fmt"
	"io"
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

	snapshot, err := collect.New(kubeClient, istioClient).Collect(ctx, collect.Scope{
		AllNamespaces: opts.AllNamespaces,
		Namespaces:    opts.Namespaces,
		RootNamespace: opts.RootNamespace,
	})
	if err != nil {
		return fmt.Errorf("collect cluster resources: %w", err)
	}

	normalized := normalize.Build(snapshot)
	resolved := resolver.New()
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
		engineWorkloads = append(engineWorkloads, engine.WorkloadInput{
			Posture: posture,
			Namespace: engine.NamespaceInput{
				Name:           workload.Namespace.Name,
				Labels:         workload.Namespace.Labels,
				MeshEnrollment: meshEnrollmentState(workload.Namespace.AmbientEnrolled),
			},
		})
	}
	evaluated, err := engine.Evaluate(packs, engine.Input{
		Workloads: engineWorkloads,
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

func meshEnrollmentState(value resolver.Tristate) string {
	switch value {
	case resolver.True:
		return "enrolled"
	case resolver.False:
		return "not-enrolled"
	default:
		return "unknown"
	}
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
