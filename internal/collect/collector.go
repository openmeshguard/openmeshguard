package collect

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultMaxConcurrentLists = 4
	defaultListLimit          = int64(500)
)

// Collector performs bounded, read-only list collection through typed clients.
type Collector struct {
	kube          kubernetes.Interface
	istio         istioclient.Interface
	maxConcurrent int
}

// New returns a collector using typed Kubernetes and Istio clients.
func New(kube kubernetes.Interface, istio istioclient.Interface) *Collector {
	return &Collector{
		kube:          kube,
		istio:         istio,
		maxConcurrent: defaultMaxConcurrentLists,
	}
}

// SetMaxConcurrentLists overrides the list concurrency bound. Values below 1
// are ignored so callers cannot accidentally remove the bound.
func (c *Collector) SetMaxConcurrentLists(value int) {
	if value > 0 {
		c.maxConcurrent = value
	}
}

// Collect lists the M1 resource set. Permission-like failures are recorded and
// degraded; non-permission API failures are returned.
func (c *Collector) Collect(ctx context.Context, scope Scope) (Snapshot, error) {
	if c == nil || c.kube == nil || c.istio == nil {
		return Snapshot{}, errors.New("collector requires Kubernetes and Istio clients")
	}

	scope = normalizeScope(scope)

	var (
		snapshot Snapshot
		mu       sync.Mutex
	)

	appendPermission := func(permission Permission) {
		mu.Lock()
		defer mu.Unlock()
		snapshot.PermissionSummary = append(snapshot.PermissionSummary, permission)
	}
	appendDegraded := func(meta resourceMeta, err error) error {
		if !isDegradedListError(err) {
			return fmt.Errorf("list %s: %w", meta.resource, err)
		}
		appendPermission(meta.permission(false))
		return nil
	}

	namespaces, err := c.collectNamespaces(ctx, scope)
	if err != nil {
		if err := appendDegraded(namespaceMeta, err); err != nil {
			return Snapshot{}, err
		}
	} else {
		mu.Lock()
		snapshot.Namespaces = append(snapshot.Namespaces, namespaces...)
		mu.Unlock()
		appendPermission(namespaceMeta.permission(true))
	}

	var tasks []func(context.Context) error
	for _, namespace := range workloadNamespaces(scope) {
		ns := namespace
		tasks = append(tasks,
			c.listTask(podMeta, func(ctx context.Context) error {
				items, err := listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]corev1.Pod, string, error) {
					list, err := c.kube.CoreV1().Pods(ns).List(ctx, opts)
					if err != nil {
						return nil, "", err
					}
					return list.Items, list.Continue, nil
				})
				if err == nil {
					mu.Lock()
					snapshot.Pods = append(snapshot.Pods, items...)
					mu.Unlock()
				}
				return err
			}, appendPermission, appendDegraded),
			c.listTask(serviceMeta, func(ctx context.Context) error {
				items, err := listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]corev1.Service, string, error) {
					list, err := c.kube.CoreV1().Services(ns).List(ctx, opts)
					if err != nil {
						return nil, "", err
					}
					return list.Items, list.Continue, nil
				})
				if err == nil {
					mu.Lock()
					snapshot.Services = append(snapshot.Services, items...)
					mu.Unlock()
				}
				return err
			}, appendPermission, appendDegraded),
			c.listTask(deploymentMeta, func(ctx context.Context) error {
				items, err := listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]appsv1.Deployment, string, error) {
					list, err := c.kube.AppsV1().Deployments(ns).List(ctx, opts)
					if err != nil {
						return nil, "", err
					}
					return list.Items, list.Continue, nil
				})
				if err == nil {
					mu.Lock()
					snapshot.Deployments = append(snapshot.Deployments, items...)
					mu.Unlock()
				}
				return err
			}, appendPermission, appendDegraded),
			c.listTask(replicaSetMeta, func(ctx context.Context) error {
				items, err := listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]appsv1.ReplicaSet, string, error) {
					list, err := c.kube.AppsV1().ReplicaSets(ns).List(ctx, opts)
					if err != nil {
						return nil, "", err
					}
					return list.Items, list.Continue, nil
				})
				if err == nil {
					mu.Lock()
					snapshot.ReplicaSets = append(snapshot.ReplicaSets, items...)
					mu.Unlock()
				}
				return err
			}, appendPermission, appendDegraded),
			c.listTask(statefulSetMeta, func(ctx context.Context) error {
				items, err := listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]appsv1.StatefulSet, string, error) {
					list, err := c.kube.AppsV1().StatefulSets(ns).List(ctx, opts)
					if err != nil {
						return nil, "", err
					}
					return list.Items, list.Continue, nil
				})
				if err == nil {
					mu.Lock()
					snapshot.StatefulSets = append(snapshot.StatefulSets, items...)
					mu.Unlock()
				}
				return err
			}, appendPermission, appendDegraded),
			c.listTask(daemonSetMeta, func(ctx context.Context) error {
				items, err := listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]appsv1.DaemonSet, string, error) {
					list, err := c.kube.AppsV1().DaemonSets(ns).List(ctx, opts)
					if err != nil {
						return nil, "", err
					}
					return list.Items, list.Continue, nil
				})
				if err == nil {
					mu.Lock()
					snapshot.DaemonSets = append(snapshot.DaemonSets, items...)
					mu.Unlock()
				}
				return err
			}, appendPermission, appendDegraded),
		)
	}
	for _, namespace := range peerAuthenticationNamespaces(scope) {
		ns := namespace
		tasks = append(tasks, c.listTask(peerAuthenticationMeta, func(ctx context.Context) error {
			items, err := listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]*istiosecurityv1beta1.PeerAuthentication, string, error) {
				list, err := c.istio.SecurityV1beta1().PeerAuthentications(ns).List(ctx, opts)
				if err != nil {
					return nil, "", err
				}
				return list.Items, list.Continue, nil
			})
			if err == nil {
				mu.Lock()
				snapshot.PeerAuthentications = append(snapshot.PeerAuthentications, items...)
				mu.Unlock()
			}
			return err
		}, appendPermission, appendDegraded))
	}

	if err := c.runBounded(ctx, tasks); err != nil {
		return Snapshot{}, err
	}
	snapshot.PermissionSummary = mergePermissions(snapshot.PermissionSummary)

	return snapshot, nil
}

func (c *Collector) collectNamespaces(ctx context.Context, scope Scope) ([]corev1.Namespace, error) {
	if scope.AllNamespaces {
		return listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]corev1.Namespace, string, error) {
			list, err := c.kube.CoreV1().Namespaces().List(ctx, opts)
			if err != nil {
				return nil, "", err
			}
			return list.Items, list.Continue, nil
		})
	}

	var out []corev1.Namespace
	for _, name := range scope.Namespaces {
		items, err := listPages(ctx, func(ctx context.Context, opts metav1.ListOptions) ([]corev1.Namespace, string, error) {
			opts.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
			list, err := c.kube.CoreV1().Namespaces().List(ctx, opts)
			if err != nil {
				return nil, "", err
			}
			return list.Items, list.Continue, nil
		})
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func (c *Collector) listTask(
	meta resourceMeta,
	list func(context.Context) error,
	appendPermission func(Permission),
	appendDegraded func(resourceMeta, error) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := list(ctx); err != nil {
			return appendDegraded(meta, err)
		}
		appendPermission(meta.permission(true))
		return nil
	}
}

func listPages[T any](ctx context.Context, list func(context.Context, metav1.ListOptions) ([]T, string, error)) ([]T, error) {
	var out []T
	opts := metav1.ListOptions{Limit: defaultListLimit}
	for {
		items, next, err := list(ctx, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if next == "" {
			return out, nil
		}
		opts.Continue = next
	}
}

func (c *Collector) runBounded(ctx context.Context, tasks []func(context.Context) error) error {
	maxConcurrent := c.maxConcurrent
	if maxConcurrent < 1 {
		maxConcurrent = defaultMaxConcurrentLists
	}

	sem := make(chan struct{}, maxConcurrent)
	errs := make(chan error, len(tasks))
	var wg sync.WaitGroup

	for _, task := range tasks {
		task := task
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			}
			errs <- task(ctx)
		}()
	}

	wg.Wait()
	close(errs)

	var joined error
	for err := range errs {
		if err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func normalizeScope(scope Scope) Scope {
	if scope.AllNamespaces || len(scope.Namespaces) == 0 {
		return Scope{AllNamespaces: true, RootNamespace: rootNamespace(scope)}
	}
	seen := map[string]struct{}{}
	namespaces := make([]string, 0, len(scope.Namespaces))
	for _, namespace := range scope.Namespaces {
		if namespace == "" {
			continue
		}
		if _, ok := seen[namespace]; ok {
			continue
		}
		seen[namespace] = struct{}{}
		namespaces = append(namespaces, namespace)
	}
	if len(namespaces) == 0 {
		return Scope{AllNamespaces: true, RootNamespace: rootNamespace(scope)}
	}
	return Scope{Namespaces: namespaces, RootNamespace: rootNamespace(scope)}
}

func workloadNamespaces(scope Scope) []string {
	if scope.AllNamespaces {
		return []string{metav1.NamespaceAll}
	}
	return scope.Namespaces
}

func peerAuthenticationNamespaces(scope Scope) []string {
	if scope.AllNamespaces {
		return []string{metav1.NamespaceAll}
	}
	return appendIfMissing(scope.Namespaces, rootNamespace(scope))
}

func rootNamespace(scope Scope) string {
	if scope.RootNamespace != "" {
		return scope.RootNamespace
	}
	return DefaultRootNamespace
}

func appendIfMissing(values []string, value string) []string {
	out := append([]string(nil), values...)
	for _, existing := range out {
		if existing == value {
			return out
		}
	}
	return append(out, value)
}

func isDegradedListError(err error) bool {
	return apierrors.IsForbidden(err) || apierrors.IsNotFound(err)
}

func mergePermissions(permissions []Permission) []Permission {
	merged := map[string]Permission{}
	for _, permission := range permissions {
		key := permission.APIGroup + "/" + permission.Resource
		existing, ok := merged[key]
		if !ok {
			merged[key] = permission
			continue
		}
		existing.Granted = existing.Granted && permission.Granted
		existing.Verbs = mergeStrings(existing.Verbs, permission.Verbs)
		existing.AffectedControls = mergeStrings(existing.AffectedControls, permission.AffectedControls)
		if existing.Impact == "" {
			existing.Impact = permission.Impact
		}
		existing.Optional = existing.Optional && permission.Optional
		merged[key] = existing
	}

	out := make([]Permission, 0, len(merged))
	for _, permission := range merged {
		out = append(out, permission)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].APIGroup != out[j].APIGroup {
			return out[i].APIGroup < out[j].APIGroup
		}
		return out[i].Resource < out[j].Resource
	})
	return out
}

func mergeStrings(left, right []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, values := range [][]string{left, right} {
		for _, value := range values {
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

type resourceMeta struct {
	apiGroup string
	resource string
	optional bool
	impact   string
	controls []string
}

func (m resourceMeta) permission(granted bool) Permission {
	return Permission{
		APIGroup:         m.apiGroup,
		Resource:         m.resource,
		Verbs:            []string{"list"},
		Granted:          granted,
		Optional:         m.optional,
		Impact:           m.impact,
		AffectedControls: append([]string(nil), m.controls...),
	}
}

var (
	namespaceMeta = resourceMeta{
		resource: "namespaces",
		impact:   "namespace labels and environment/data-plane inference may be unavailable",
		controls: []string{"MG-MTLS-001"},
	}
	podMeta = resourceMeta{
		resource: "pods",
		impact:   "sidecar detection from running pods may be unavailable",
		controls: []string{"MG-MTLS-001"},
	}
	serviceMeta = resourceMeta{
		resource: "services",
		impact:   "service inventory and multi-cluster gateway signals may be unavailable",
		controls: []string{"MG-MTLS-001"},
	}
	deploymentMeta = resourceMeta{
		apiGroup: "apps",
		resource: "deployments",
		impact:   "deployment workload posture may be unavailable",
		controls: []string{"MG-MTLS-001"},
	}
	replicaSetMeta = resourceMeta{
		apiGroup: "apps",
		resource: "replicasets",
		impact:   "standalone ReplicaSet workload posture may be unavailable",
		controls: []string{"MG-MTLS-001"},
	}
	statefulSetMeta = resourceMeta{
		apiGroup: "apps",
		resource: "statefulsets",
		impact:   "StatefulSet workload posture may be unavailable",
		controls: []string{"MG-MTLS-001"},
	}
	daemonSetMeta = resourceMeta{
		apiGroup: "apps",
		resource: "daemonsets",
		impact:   "DaemonSet workload posture may be unavailable",
		controls: []string{"MG-MTLS-001"},
	}
	peerAuthenticationMeta = resourceMeta{
		apiGroup: "security.istio.io",
		resource: "peerauthentications",
		impact:   "effective mTLS posture resolves to unknown without PeerAuthentication evidence",
		controls: []string{"MG-MTLS-001"},
	}
)
