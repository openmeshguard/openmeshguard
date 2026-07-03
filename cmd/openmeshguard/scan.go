package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/openmeshguard/openmeshguard/internal/collect"
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
	if !o.AllNamespaces && len(o.Namespaces) == 0 {
		return fmt.Errorf("scan scope required: pass --all-namespaces or at least one --namespace")
	}
	return nil
}

func runScan(ctx context.Context, info versionInfo, opts scanOptions, stdout io.Writer) error {
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
		RootNamespace: collect.DefaultRootNamespace,
	})
	if err != nil {
		return fmt.Errorf("collect cluster resources: %w", err)
	}

	normalized := normalize.Build(snapshot)
	resolved := resolver.NewProvisional()
	workloadPostures := make([]resolver.WorkloadResult, 0, len(normalized.Workloads))
	for _, workload := range normalized.Workloads {
		workloadPostures = append(workloadPostures, resolver.WorkloadResult{
			Ref:   workload.Ref,
			Mode:  workload.DataPlaneMode,
			MTLS:  resolved.ResolveMTLS(workload),
			Authz: resolved.ResolveAuthz(workload),
		})
	}

	return output.WriteScanJSON(stdout, output.ScanInput{
		ScannerVersion:    info.Version,
		ResolverVersion:   resolved.Version(),
		ClusterContext:    clusterContext,
		Scope:             outputScope(opts),
		PermissionSummary: snapshot.PermissionSummary,
		Inventory:         normalized.Inventory,
		WorkloadPostures:  workloadPostures,
	})
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
