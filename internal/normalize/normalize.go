package normalize

import (
	"sort"
	"strings"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	securityapi "istio.io/api/security/v1beta1"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const defaultRootNamespace = "istio-system"

// Build converts collected typed resources into the normalized M1 inventory and
// resolver inputs. M1 intentionally omits ports, DestinationRules, authz, and
// ambient resolution.
func Build(snapshot collect.Snapshot) Result {
	rootNamespace := snapshot.RootNamespace
	if rootNamespace == "" {
		rootNamespace = defaultRootNamespace
	}
	namespaceLabels := map[string]map[string]string{}
	for _, namespace := range snapshot.Namespaces {
		namespaceLabels[namespace.Name] = copyStringMap(namespace.Labels)
	}

	peerAuthentications := projectPeerAuthentications(snapshot.PeerAuthentications)

	builder := workloadBuilder{
		namespaces:          namespaceLabels,
		pods:                snapshot.Pods,
		peerAuthentications: peerAuthentications,
		coveredPods:         map[string]struct{}{},
		rootNamespace:       rootNamespace,
		podsAvailableFor: func(namespace string) bool {
			return snapshot.PodsAvailableFor(namespace)
		},
		peerAuthenticationsAvailableFor: func(namespace string) bool {
			return snapshot.PeerAuthenticationsAvailableFor(namespace, rootNamespace)
		},
	}

	for _, deployment := range snapshot.Deployments {
		builder.addController(
			"Deployment",
			deployment.Namespace,
			deployment.Name,
			deployment.Spec.Template,
			deployment.Spec.Selector,
		)
	}
	for _, replicaSet := range snapshot.ReplicaSets {
		if hasOwnerKind(replicaSet.OwnerReferences, "Deployment") {
			continue
		}
		builder.addController(
			"ReplicaSet",
			replicaSet.Namespace,
			replicaSet.Name,
			replicaSet.Spec.Template,
			replicaSet.Spec.Selector,
		)
	}
	for _, statefulSet := range snapshot.StatefulSets {
		builder.addController(
			"StatefulSet",
			statefulSet.Namespace,
			statefulSet.Name,
			statefulSet.Spec.Template,
			statefulSet.Spec.Selector,
		)
	}
	for _, daemonSet := range snapshot.DaemonSets {
		builder.addController(
			"DaemonSet",
			daemonSet.Namespace,
			daemonSet.Name,
			daemonSet.Spec.Template,
			daemonSet.Spec.Selector,
		)
	}
	for _, pod := range snapshot.Pods {
		if builder.podCovered(pod) {
			continue
		}
		builder.addPod(pod)
	}

	workloads := builder.workloads
	sort.Slice(workloads, func(i, j int) bool {
		left, right := workloads[i].Ref, workloads[j].Ref
		if left.Namespace != right.Namespace {
			return left.Namespace < right.Namespace
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.Name < right.Name
	})

	return Result{
		Inventory: Inventory{
			Counts: map[string]int{
				"namespaces":          len(snapshot.Namespaces),
				"pods":                len(snapshot.Pods),
				"services":            len(snapshot.Services),
				"deployments":         len(snapshot.Deployments),
				"replicasets":         len(snapshot.ReplicaSets),
				"statefulsets":        len(snapshot.StatefulSets),
				"daemonsets":          len(snapshot.DaemonSets),
				"peerAuthentications": len(snapshot.PeerAuthentications),
			},
			DataPlaneMode: aggregateDataPlaneMode(workloads),
			MultiCluster:  detectMultiCluster(snapshot),
		},
		Workloads: workloads,
	}
}

type workloadBuilder struct {
	namespaces                      map[string]map[string]string
	pods                            []corev1.Pod
	peerAuthentications             []peerAuthenticationProjection
	coveredPods                     map[string]struct{}
	rootNamespace                   string
	podsAvailableFor                func(namespace string) bool
	peerAuthenticationsAvailableFor func(namespace string) bool

	workloads []resolver.WorkloadInput
}

func (b *workloadBuilder) addController(kind, namespace, name string, template corev1.PodTemplateSpec, selector *metav1.LabelSelector) {
	labels := copyStringMap(template.Labels)
	nsLabels := b.namespaces[namespace]
	pods := podsMatching(b.pods, namespace, selector)
	for _, pod := range pods {
		b.coverPod(pod)
	}
	mode := detectDataPlaneMode(nsLabels, template.Labels, template.Annotations, template.Spec, pods, b.podsAvailable(namespace))

	b.workloads = append(b.workloads, resolver.WorkloadInput{
		Ref: resolver.WorkloadRef{
			Namespace: namespace,
			Name:      name,
			Kind:      kind,
		},
		Labels:        labels,
		DataPlaneMode: mode,
		Namespace: resolver.NamespaceInput{
			Name:            namespace,
			Labels:          copyStringMap(nsLabels),
			AmbientEnrolled: resolver.Unobserved,
		},
		MeshDefaults: resolver.MeshDefaults{
			RootNamespace: b.rootNamespace,
			Known:         b.peerAuthenticationsKnown(namespace),
		},
		PeerAuthN: b.peerAuthenticationsFor(namespace, labels),
	})
}

func (b *workloadBuilder) addPod(pod corev1.Pod) {
	labels := copyStringMap(pod.Labels)
	nsLabels := b.namespaces[pod.Namespace]
	mode := detectDataPlaneMode(nsLabels, pod.Labels, pod.Annotations, pod.Spec, []corev1.Pod{pod}, b.podsAvailable(pod.Namespace))

	b.workloads = append(b.workloads, resolver.WorkloadInput{
		Ref: resolver.WorkloadRef{
			Namespace: pod.Namespace,
			Name:      pod.Name,
			Kind:      "Pod",
		},
		Labels:        labels,
		DataPlaneMode: mode,
		Namespace: resolver.NamespaceInput{
			Name:            pod.Namespace,
			Labels:          copyStringMap(nsLabels),
			AmbientEnrolled: resolver.Unobserved,
		},
		MeshDefaults: resolver.MeshDefaults{
			RootNamespace: b.rootNamespace,
			Known:         b.peerAuthenticationsKnown(pod.Namespace),
		},
		PeerAuthN: b.peerAuthenticationsFor(pod.Namespace, labels),
	})
}

func (b *workloadBuilder) coverPod(pod corev1.Pod) {
	if b.coveredPods == nil {
		b.coveredPods = map[string]struct{}{}
	}
	b.coveredPods[podKey(pod)] = struct{}{}
}

func (b *workloadBuilder) podCovered(pod corev1.Pod) bool {
	_, ok := b.coveredPods[podKey(pod)]
	return ok
}

func (b *workloadBuilder) podsAvailable(namespace string) bool {
	if b.podsAvailableFor == nil {
		return true
	}
	return b.podsAvailableFor(namespace)
}

func (b *workloadBuilder) peerAuthenticationsKnown(namespace string) bool {
	if b.peerAuthenticationsAvailableFor == nil {
		return false
	}
	return b.peerAuthenticationsAvailableFor(namespace)
}

func (b *workloadBuilder) peerAuthenticationsFor(namespace string, workloadLabels map[string]string) []resolver.PeerAuthenticationView {
	var selected []resolver.PeerAuthenticationView
	for _, peerAuthentication := range b.peerAuthentications {
		if !peerAuthentication.hasSelector {
			if peerAuthentication.Namespace == b.rootNamespace || peerAuthentication.Namespace == namespace {
				selected = append(selected, peerAuthentication.PeerAuthenticationView)
			}
			continue
		}
		// Istio root-namespace selectors additionally match workloads in every
		// namespace; M1 passes them through so the provisional resolver returns
		// explicit M2 unknown instead of ignoring them.
		if peerAuthentication.Namespace != namespace && peerAuthentication.Namespace != b.rootNamespace {
			continue
		}
		if matchLabels(peerAuthentication.selectorLabels, workloadLabels) {
			view := peerAuthentication.PeerAuthenticationView
			view.SelectorMatch = true
			selected = append(selected, view)
		}
	}
	return selected
}

type peerAuthenticationProjection struct {
	resolver.PeerAuthenticationView
	hasSelector    bool
	selectorLabels map[string]string
}

func projectPeerAuthentications(peerAuthentications []*istiosecurityv1beta1.PeerAuthentication) []peerAuthenticationProjection {
	out := make([]peerAuthenticationProjection, 0, len(peerAuthentications))
	for _, peerAuthentication := range peerAuthentications {
		if peerAuthentication == nil {
			continue
		}
		selector := peerAuthentication.Spec.GetSelector()
		selectorLabels := map[string]string(nil)
		if selector != nil {
			selectorLabels = selector.GetMatchLabels()
		}
		out = append(out, peerAuthenticationProjection{
			PeerAuthenticationView: resolver.PeerAuthenticationView{
				Name:           peerAuthentication.Name,
				Namespace:      peerAuthentication.Namespace,
				SelectorMatch:  false,
				Mode:           mtlsMode(peerAuthentication.Spec.GetMtls()),
				PortLevelModes: portModes(peerAuthentication.Spec.GetPortLevelMtls()),
			},
			hasSelector:    len(selectorLabels) > 0,
			selectorLabels: copyStringMap(selectorLabels),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func podsMatching(pods []corev1.Pod, namespace string, selector *metav1.LabelSelector) []corev1.Pod {
	compiled, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil || compiled.Empty() {
		return nil
	}
	var matches []corev1.Pod
	for _, pod := range pods {
		if pod.Namespace == namespace && compiled.Matches(labels.Set(pod.Labels)) {
			matches = append(matches, pod)
		}
	}
	return matches
}

func podKey(pod corev1.Pod) string {
	return pod.Namespace + "/" + pod.Name
}

func detectDataPlaneMode(
	namespaceLabels map[string]string,
	workloadLabels map[string]string,
	workloadAnnotations map[string]string,
	templateSpec corev1.PodSpec,
	pods []corev1.Pod,
	podEvidenceAvailable bool,
) resolver.DataPlaneMode {
	if !podEvidenceAvailable {
		return resolver.ModeUnknown
	}
	if len(pods) > 0 {
		withProxy := 0
		withoutProxy := 0
		for _, pod := range pods {
			if hasIstioProxy(pod.Spec) {
				withProxy++
				continue
			}
			withoutProxy++
		}
		switch {
		case withProxy > 0 && withoutProxy == 0:
			return resolver.ModeSidecar
		case withProxy > 0 && withoutProxy > 0:
			return resolver.ModeMixed
		default:
			return resolver.ModeUnknown
		}
	}
	if hasIstioProxy(templateSpec) {
		return resolver.ModeSidecar
	}
	if sidecarInjectionDisabled(workloadLabels, workloadAnnotations) {
		return resolver.ModeUnknown
	}
	if sidecarInjectionEnabled(workloadLabels, workloadAnnotations) || sidecarInjectionEnabled(namespaceLabels, nil) {
		return resolver.ModeSidecar
	}
	return ambientDetectionStub(namespaceLabels, workloadLabels)
}

func ambientDetectionStub(map[string]string, map[string]string) resolver.DataPlaneMode {
	return resolver.ModeUnknown
}

func hasIstioProxy(spec corev1.PodSpec) bool {
	for _, container := range append(spec.Containers, spec.InitContainers...) {
		if container.Name == "istio-proxy" {
			return true
		}
	}
	return false
}

func sidecarInjectionDisabled(labels, annotations map[string]string) bool {
	return stringValue(labels, "sidecar.istio.io/inject") == "false" ||
		stringValue(annotations, "sidecar.istio.io/inject") == "false"
}

func sidecarInjectionEnabled(labels, annotations map[string]string) bool {
	return stringValue(labels, "sidecar.istio.io/inject") == "true" ||
		stringValue(annotations, "sidecar.istio.io/inject") == "true" ||
		stringValue(labels, "istio-injection") == "enabled" ||
		stringValue(labels, "istio.io/rev") != ""
}

func aggregateDataPlaneMode(workloads []resolver.WorkloadInput) resolver.DataPlaneMode {
	if len(workloads) == 0 {
		return resolver.ModeUnknown
	}
	seen := map[resolver.DataPlaneMode]bool{}
	for _, workload := range workloads {
		seen[workload.DataPlaneMode] = true
	}
	if len(seen) == 1 {
		for mode := range seen {
			return mode
		}
	}
	return resolver.ModeMixed
}

func detectMultiCluster(snapshot collect.Snapshot) MultiCluster {
	signals := map[string]struct{}{}
	networks := map[string]struct{}{}
	for _, namespace := range snapshot.Namespaces {
		if network := namespace.Labels["topology.istio.io/network"]; network != "" {
			signals["namespace/"+namespace.Name+" topology.istio.io/network="+network] = struct{}{}
			networks[network] = struct{}{}
		}
	}
	for _, service := range snapshot.Services {
		if network := service.Labels["topology.istio.io/network"]; network != "" {
			signals["service/"+service.Namespace+"/"+service.Name+" topology.istio.io/network="+network] = struct{}{}
			networks[network] = struct{}{}
		}
		if strings.Contains(service.Name, "eastwest") || strings.Contains(service.Name, "east-west") {
			signals["service/"+service.Namespace+"/"+service.Name+" east-west gateway name"] = struct{}{}
		}
	}

	return MultiCluster{
		ParticipationDetected: len(signals) > 0,
		Signals:               sortedKeys(signals),
		MeshNetworks:          sortedKeys(networks),
	}
}

func hasOwnerKind(ownerReferences []metav1.OwnerReference, kind string) bool {
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == kind {
			return true
		}
	}
	return false
}

func matchLabels(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return true
	}
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func stringValue(values map[string]string, key string) string {
	if values == nil {
		return ""
	}
	return values[key]
}

func mtlsMode(mtls *securityapi.PeerAuthentication_MutualTLS) string {
	if mtls == nil {
		return ""
	}
	return mtls.Mode.String()
}

func portModes(modes map[uint32]*securityapi.PeerAuthentication_MutualTLS) map[int32]string {
	if len(modes) == 0 {
		return nil
	}
	out := make(map[int32]string, len(modes))
	for port, mtls := range modes {
		if port > uint32(1<<31-1) {
			continue
		}
		out[int32(port)] = mtlsMode(mtls)
	}
	return out
}
