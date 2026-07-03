package collect

import (
	"context"
	"errors"
	"fmt"
	"sync"

	istioclient "istio.io/client-go/pkg/clientset/versioned"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const defaultMaxConcurrentLists = 4

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

	namespaces, err := c.kube.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		if err := appendDegraded(namespaceMeta, err); err != nil {
			return Snapshot{}, err
		}
	} else {
		mu.Lock()
		snapshot.Namespaces = append(snapshot.Namespaces, namespaces.Items...)
		mu.Unlock()
		appendPermission(namespaceMeta.permission(true))
	}

	var tasks []func(context.Context) error
	for _, namespace := range collectionNamespaces(scope) {
		ns := namespace
		tasks = append(tasks,
			func(ctx context.Context) error {
				pods, err := c.kube.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return appendDegraded(podMeta, err)
				}
				mu.Lock()
				snapshot.Pods = append(snapshot.Pods, pods.Items...)
				mu.Unlock()
				appendPermission(podMeta.permission(true))
				return nil
			},
			func(ctx context.Context) error {
				services, err := c.kube.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return appendDegraded(serviceMeta, err)
				}
				mu.Lock()
				snapshot.Services = append(snapshot.Services, services.Items...)
				mu.Unlock()
				appendPermission(serviceMeta.permission(true))
				return nil
			},
			func(ctx context.Context) error {
				deployments, err := c.kube.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return appendDegraded(deploymentMeta, err)
				}
				mu.Lock()
				snapshot.Deployments = append(snapshot.Deployments, deployments.Items...)
				mu.Unlock()
				appendPermission(deploymentMeta.permission(true))
				return nil
			},
			func(ctx context.Context) error {
				replicaSets, err := c.kube.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return appendDegraded(replicaSetMeta, err)
				}
				mu.Lock()
				snapshot.ReplicaSets = append(snapshot.ReplicaSets, replicaSets.Items...)
				mu.Unlock()
				appendPermission(replicaSetMeta.permission(true))
				return nil
			},
			func(ctx context.Context) error {
				statefulSets, err := c.kube.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return appendDegraded(statefulSetMeta, err)
				}
				mu.Lock()
				snapshot.StatefulSets = append(snapshot.StatefulSets, statefulSets.Items...)
				mu.Unlock()
				appendPermission(statefulSetMeta.permission(true))
				return nil
			},
			func(ctx context.Context) error {
				daemonSets, err := c.kube.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return appendDegraded(daemonSetMeta, err)
				}
				mu.Lock()
				snapshot.DaemonSets = append(snapshot.DaemonSets, daemonSets.Items...)
				mu.Unlock()
				appendPermission(daemonSetMeta.permission(true))
				return nil
			},
			func(ctx context.Context) error {
				peerAuthentications, err := c.istio.SecurityV1beta1().PeerAuthentications(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					return appendDegraded(peerAuthenticationMeta, err)
				}
				mu.Lock()
				snapshot.PeerAuthentications = append(snapshot.PeerAuthentications, peerAuthentications.Items...)
				mu.Unlock()
				appendPermission(peerAuthenticationMeta.permission(true))
				return nil
			},
		)
	}

	if err := c.runBounded(ctx, tasks); err != nil {
		return Snapshot{}, err
	}

	return snapshot, nil
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
		return Scope{AllNamespaces: true}
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
		return Scope{AllNamespaces: true}
	}
	return Scope{Namespaces: namespaces}
}

func collectionNamespaces(scope Scope) []string {
	if scope.AllNamespaces {
		return []string{metav1.NamespaceAll}
	}
	return scope.Namespaces
}

func isDegradedListError(err error) bool {
	return apierrors.IsForbidden(err) || apierrors.IsNotFound(err)
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
