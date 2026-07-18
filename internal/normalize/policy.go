package normalize

import (
	"reflect"
	"sort"
	"strings"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	securityapi "istio.io/api/security/v1beta1"
	typeapi "istio.io/api/type/v1beta1"
	istionetworkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	istiosecurityv1 "istio.io/client-go/pkg/apis/security/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	useWaypointLabel          = "istio.io/use-waypoint"
	useWaypointNamespaceLabel = "istio.io/use-waypoint-namespace"
	waypointForLabel          = "istio.io/waypoint-for"
	waypointGatewayClass      = "istio-waypoint"
)

type workloadPolicyInputs struct {
	ports                 []int32
	destinationRules      []resolver.DestinationRuleView
	destinationRulesKnown bool
	authorizationPolicies []resolver.AuthorizationPolicyView
	waypoint              *resolver.WaypointView
}

type destinationRuleProjection struct {
	view           resolver.DestinationRuleView
	exportTo       []string
	selectorLabels map[string]string
}

type sidecarProjection struct {
	name           string
	namespace      string
	selectorLabels map[string]string
	egressHosts    []string
}

type clientProxy struct {
	namespace string
	labels    map[string]string
}

type policyTargetRef struct {
	group string
	kind  string
	name  string
}

type authorizationPolicyProjection struct {
	view           resolver.AuthorizationPolicyView
	selectorLabels map[string]string
	targetRefs     []policyTargetRef
}

type gatewayProjection struct {
	name      string
	namespace string
	scope     string
	ready     bool
}

func uniformWorkloadPolicyInputs(inputs []workloadPolicyInputs) bool {
	for i := 1; i < len(inputs); i++ {
		if !reflect.DeepEqual(inputs[0], inputs[i]) {
			return false
		}
	}
	return true
}

func (b *workloadBuilder) policyInputs(
	namespace string,
	labelSets []map[string]string,
	podSpecs []corev1.PodSpec,
	workloadLabels map[string]string,
	namespaceLabels map[string]string,
) workloadPolicyInputs {
	services := b.selectedServices(namespace, labelSets)
	waypoint := b.waypointFor(namespace, workloadLabels, namespaceLabels, services)
	ports := b.serviceBoundPorts(namespace, services, podSpecs)
	destinationRules, destinationRulesKnown := b.destinationRulesFor(namespace, services, podSpecs)
	return workloadPolicyInputs{
		ports:                 ports,
		destinationRules:      destinationRules,
		destinationRulesKnown: destinationRulesKnown,
		authorizationPolicies: b.authorizationPoliciesFor(namespace, labelSets, services, namespaceLabels, waypoint),
		waypoint:              waypoint,
	}
}

func (b *workloadBuilder) selectedServices(namespace string, labelSets []map[string]string) []corev1.Service {
	var selected []corev1.Service
	for _, service := range b.services {
		if service.Namespace != namespace || len(service.Spec.Selector) == 0 {
			continue
		}
		if matchAnyLabels(service.Spec.Selector, labelSets) {
			selected = append(selected, service)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Name < selected[j].Name })
	return selected
}

func (b *workloadBuilder) serviceBoundPorts(namespace string, services []corev1.Service, podSpecs []corev1.PodSpec) []int32 {
	if !b.servicesKnown(namespace) {
		return nil
	}
	ports := map[int32]struct{}{}
	for _, service := range services {
		for _, servicePort := range service.Spec.Ports {
			port, ok := resolveTargetPort(servicePort, podSpecs)
			if !ok {
				return nil
			}
			ports[port] = struct{}{}
		}
	}
	out := make([]int32, 0, len(ports))
	for port := range ports {
		out = append(out, port)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func resolveTargetPort(servicePort corev1.ServicePort, podSpecs []corev1.PodSpec) (int32, bool) {
	switch servicePort.TargetPort.Type {
	case intstr.Int:
		if servicePort.TargetPort.IntVal > 0 {
			return servicePort.TargetPort.IntVal, true
		}
		return servicePort.Port, servicePort.Port > 0
	case intstr.String:
		name := servicePort.TargetPort.StrVal
		if name == "" {
			return servicePort.Port, servicePort.Port > 0
		}
		var found int32
		for _, spec := range podSpecs {
			for _, container := range append(spec.Containers, spec.InitContainers...) {
				for _, containerPort := range container.Ports {
					if containerPort.Name != name {
						continue
					}
					if found != 0 && found != containerPort.ContainerPort {
						return 0, false
					}
					found = containerPort.ContainerPort
				}
			}
		}
		return found, found > 0
	default:
		return 0, false
	}
}

func (b *workloadBuilder) destinationRulesFor(
	namespace string,
	services []corev1.Service,
	podSpecs []corev1.PodSpec,
) ([]resolver.DestinationRuleView, bool) {
	if !b.servicesKnown(namespace) || !b.destinationRulesKnown(namespace) {
		return nil, false
	}
	out := make([]resolver.DestinationRuleView, 0)
	for _, client := range b.clientProxies {
		if !b.destinationRulesKnown(client.namespace) || !b.sidecarsKnown(client.namespace) {
			return nil, false
		}
		selectedSidecars := b.sidecarsFor(client.namespace, []map[string]string{client.labels})
		for _, service := range services {
			if !b.destinationRulesKnown(service.Namespace) {
				return nil, false
			}
			if !sidecarsAllowService(selectedSidecars, client.namespace, service) {
				continue
			}
			for _, destinationRule := range b.destinationRulesForClientService(client, service) {
				view, ok := destinationRuleViewForService(destinationRule.view, service, podSpecs)
				if !ok {
					return nil, false
				}
				if !containsDestinationRuleView(out, view) {
					out = append(out, view)
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	if out == nil {
		out = []resolver.DestinationRuleView{}
	}
	return out, true
}

func (b *workloadBuilder) destinationRulesForClientService(client clientProxy, service corev1.Service) []destinationRuleProjection {
	lookupNamespaces := uniqueNonEmptyStrings(client.namespace, service.Namespace, b.rootNamespace)
	for _, lookupNamespace := range lookupNamespaces {
		var selected []destinationRuleProjection
		for _, rule := range b.destinationRules {
			if rule.view.Namespace != lookupNamespace ||
				!exportedTo(rule.exportTo, rule.view.Namespace, client.namespace) ||
				!destinationRuleTargetsService(rule, service) {
				continue
			}
			if len(rule.selectorLabels) > 0 &&
				(rule.view.Namespace != client.namespace || !matchAnyLabels(rule.selectorLabels, []map[string]string{client.labels})) {
				continue
			}
			selected = append(selected, rule)
		}
		if len(selected) > 0 {
			return selected
		}
	}
	return nil
}

func destinationRuleViewForService(
	view resolver.DestinationRuleView,
	service corev1.Service,
	podSpecs []corev1.PodSpec,
) (resolver.DestinationRuleView, bool) {
	if view.PortTLSModes == nil {
		return view, true
	}
	translated := map[int32]string{}
	for servicePort, mode := range view.PortTLSModes {
		for _, candidate := range service.Spec.Ports {
			if candidate.Port != servicePort {
				continue
			}
			workloadPort, ok := resolveTargetPort(candidate, podSpecs)
			if !ok {
				return resolver.DestinationRuleView{}, false
			}
			if existing, exists := translated[workloadPort]; exists && existing != mode {
				return resolver.DestinationRuleView{}, false
			}
			translated[workloadPort] = mode
		}
	}
	if len(translated) == 0 {
		view.PortTLSModes = nil
	} else {
		view.PortTLSModes = translated
	}
	return view, true
}

func containsDestinationRuleView(views []resolver.DestinationRuleView, candidate resolver.DestinationRuleView) bool {
	for _, view := range views {
		if reflect.DeepEqual(view, candidate) {
			return true
		}
	}
	return false
}

func uniqueNonEmptyStrings(values ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
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

func projectClientProxies(snapshot collect.Snapshot) []clientProxy {
	var proxies []clientProxy
	add := func(namespace string, labels map[string]string, spec corev1.PodSpec) {
		if namespace == "" || !hasContainer(spec, "istio-proxy") {
			return
		}
		candidate := clientProxy{namespace: namespace, labels: copyStringMap(labels)}
		for _, proxy := range proxies {
			if proxy.namespace == candidate.namespace && reflect.DeepEqual(proxy.labels, candidate.labels) {
				return
			}
		}
		proxies = append(proxies, candidate)
	}
	for _, pod := range snapshot.Pods {
		add(pod.Namespace, pod.Labels, pod.Spec)
	}
	for _, deployment := range snapshot.Deployments {
		add(deployment.Namespace, deployment.Spec.Template.Labels, deployment.Spec.Template.Spec)
	}
	for _, replicaSet := range snapshot.ReplicaSets {
		add(replicaSet.Namespace, replicaSet.Spec.Template.Labels, replicaSet.Spec.Template.Spec)
	}
	for _, statefulSet := range snapshot.StatefulSets {
		add(statefulSet.Namespace, statefulSet.Spec.Template.Labels, statefulSet.Spec.Template.Spec)
	}
	for _, daemonSet := range snapshot.DaemonSets {
		add(daemonSet.Namespace, daemonSet.Spec.Template.Labels, daemonSet.Spec.Template.Spec)
	}
	sort.Slice(proxies, func(i, j int) bool {
		if proxies[i].namespace != proxies[j].namespace {
			return proxies[i].namespace < proxies[j].namespace
		}
		return canonicalLabels(proxies[i].labels) < canonicalLabels(proxies[j].labels)
	})
	return proxies
}

func hasContainer(spec corev1.PodSpec, name string) bool {
	for _, container := range append(spec.Containers, spec.InitContainers...) {
		if container.Name == name {
			return true
		}
	}
	return false
}

func canonicalLabels(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		out.WriteString(key)
		out.WriteByte('=')
		out.WriteString(values[key])
		out.WriteByte('\x00')
	}
	return out.String()
}

func projectDestinationRules(rules []*istionetworkingv1.DestinationRule) []destinationRuleProjection {
	out := make([]destinationRuleProjection, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		trafficPolicy := rule.Spec.GetTrafficPolicy()
		view := resolver.DestinationRuleView{
			Name:      rule.Name,
			Namespace: rule.Namespace,
			Host:      rule.Spec.GetHost(),
		}
		if trafficPolicy != nil && trafficPolicy.GetTls() != nil {
			view.TLSMode = trafficPolicy.GetTls().GetMode().String()
		}
		if trafficPolicy != nil {
			for _, portPolicy := range trafficPolicy.GetPortLevelSettings() {
				if portPolicy == nil || portPolicy.GetPort() == nil || portPolicy.GetPort().GetNumber() == 0 {
					continue
				}
				if view.PortTLSModes == nil {
					view.PortTLSModes = map[int32]string{}
				}
				mode := ""
				if portPolicy.GetTls() != nil {
					mode = portPolicy.GetTls().GetMode().String()
				}
				view.PortTLSModes[int32(portPolicy.GetPort().GetNumber())] = mode
			}
		}
		selector := rule.Spec.GetWorkloadSelector()
		var selectorLabels map[string]string
		if selector != nil {
			selectorLabels = copyStringMap(selector.GetMatchLabels())
		}
		out = append(out, destinationRuleProjection{
			view:           view,
			exportTo:       append([]string(nil), rule.Spec.GetExportTo()...),
			selectorLabels: selectorLabels,
		})
	}
	return out
}

func destinationRuleTargetsService(rule destinationRuleProjection, service corev1.Service) bool {
	host := canonicalDestinationHost(rule.view.Host, rule.view.Namespace)
	serviceHost := service.Name + "." + service.Namespace + ".svc.cluster.local"
	return wildcardDNSMatch(host, serviceHost)
}

func canonicalDestinationHost(host, ruleNamespace string) string {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	switch strings.Count(host, ".") {
	case 0:
		return host + "." + ruleNamespace + ".svc.cluster.local"
	case 1:
		return host + ".svc.cluster.local"
	case 2:
		if strings.HasSuffix(host, ".svc") {
			return host + ".cluster.local"
		}
	}
	return host
}

func wildcardDNSMatch(pattern, value string) bool {
	if pattern == value || pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		return strings.HasSuffix(value, pattern[1:])
	}
	return false
}

func exportedTo(exportTo []string, sourceNamespace, targetNamespace string) bool {
	if len(exportTo) == 0 {
		return true
	}
	for _, namespace := range exportTo {
		switch namespace {
		case "*":
			return true
		case ".":
			if sourceNamespace == targetNamespace {
				return true
			}
		case targetNamespace:
			return true
		}
	}
	return false
}

func projectSidecars(sidecars []*istionetworkingv1.Sidecar) []sidecarProjection {
	out := make([]sidecarProjection, 0, len(sidecars))
	for _, sidecar := range sidecars {
		if sidecar == nil {
			continue
		}
		projection := sidecarProjection{name: sidecar.Name, namespace: sidecar.Namespace}
		if selector := sidecar.Spec.GetWorkloadSelector(); selector != nil {
			projection.selectorLabels = copyStringMap(selector.GetLabels())
		}
		for _, listener := range sidecar.Spec.GetEgress() {
			if listener != nil {
				projection.egressHosts = append(projection.egressHosts, listener.GetHosts()...)
			}
		}
		out = append(out, projection)
	}
	return out
}

func (b *workloadBuilder) sidecarsFor(namespace string, labelSets []map[string]string) []sidecarProjection {
	var namespaceDefaults []sidecarProjection
	var rootDefaults []sidecarProjection
	var selected []sidecarProjection
	for _, sidecar := range b.sidecars {
		if sidecar.namespace == b.rootNamespace && sidecar.namespace != namespace {
			if len(sidecar.selectorLabels) == 0 {
				rootDefaults = append(rootDefaults, sidecar)
			}
			continue
		}
		if sidecar.namespace != namespace {
			continue
		}
		if len(sidecar.selectorLabels) == 0 {
			namespaceDefaults = append(namespaceDefaults, sidecar)
			continue
		}
		if matchAnyLabels(sidecar.selectorLabels, labelSets) {
			selected = append(selected, sidecar)
		}
	}
	if len(selected) > 0 {
		return selected
	}
	if len(namespaceDefaults) > 0 {
		return namespaceDefaults
	}
	return rootDefaults
}

func sidecarsAllowService(sidecars []sidecarProjection, workloadNamespace string, service corev1.Service) bool {
	if len(sidecars) == 0 {
		return true
	}
	for _, sidecar := range sidecars {
		if len(sidecar.egressHosts) == 0 {
			return true
		}
		for _, host := range sidecar.egressHosts {
			parts := strings.SplitN(host, "/", 2)
			if len(parts) != 2 {
				continue
			}
			namespacePattern := parts[0]
			if namespacePattern == "." {
				namespacePattern = workloadNamespace
			}
			if namespacePattern != "*" && namespacePattern != service.Namespace {
				continue
			}
			hostPattern := canonicalDestinationHost(parts[1], service.Namespace)
			if parts[1] == "*" || wildcardDNSMatch(hostPattern, service.Name+"."+service.Namespace+".svc.cluster.local") {
				return true
			}
		}
	}
	return false
}

func projectAuthorizationPolicies(
	policies []*istiosecurityv1.AuthorizationPolicy,
	rootNamespace string,
) []authorizationPolicyProjection {
	out := make([]authorizationPolicyProjection, 0, len(policies))
	for _, policy := range policies {
		if policy == nil || strings.EqualFold(policy.Annotations["istio.io/dry-run"], "true") {
			continue
		}
		selector := policy.Spec.GetSelector()
		projection := authorizationPolicyProjection{
			view: resolver.AuthorizationPolicyView{
				Name:           policy.Name,
				Namespace:      policy.Namespace,
				Action:         policy.Spec.GetAction().String(),
				HasSelector:    selector != nil,
				RequiresL7:     authorizationPolicyRequiresL7(&policy.Spec),
				HasRules:       len(policy.Spec.GetRules()) > 0,
				BroadAllow:     authorizationPolicyBroadAllow(&policy.Spec),
				IdentityScoped: authorizationPolicyIdentityScoped(&policy.Spec),
				RootNamespace:  policy.Namespace == rootNamespace,
			},
		}
		if selector != nil {
			projection.selectorLabels = copyStringMap(selector.GetMatchLabels())
		}
		targetRefs := make([]*typeapi.PolicyTargetReference, 0, len(policy.Spec.GetTargetRefs())+1)
		if targetRef := policy.Spec.GetTargetRef(); targetRef != nil {
			targetRefs = append(targetRefs, targetRef)
		}
		targetRefs = append(targetRefs, policy.Spec.GetTargetRefs()...)
		seenTargetRefs := map[policyTargetRef]struct{}{}
		for _, targetRef := range targetRefs {
			if targetRef == nil {
				continue
			}
			ref := policyTargetRef{
				group: targetRef.GetGroup(),
				kind:  targetRef.GetKind(),
				name:  targetRef.GetName(),
			}
			if _, exists := seenTargetRefs[ref]; exists {
				continue
			}
			seenTargetRefs[ref] = struct{}{}
			projection.targetRefs = append(projection.targetRefs, ref)
		}
		sort.Slice(projection.targetRefs, func(i, j int) bool {
			if projection.targetRefs[i].kind != projection.targetRefs[j].kind {
				return projection.targetRefs[i].kind < projection.targetRefs[j].kind
			}
			return projection.targetRefs[i].name < projection.targetRefs[j].name
		})
		out = append(out, projection)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].view.Namespace != out[j].view.Namespace {
			return out[i].view.Namespace < out[j].view.Namespace
		}
		return out[i].view.Name < out[j].view.Name
	})
	return out
}

func (b *workloadBuilder) authorizationPoliciesFor(
	namespace string,
	labelSets []map[string]string,
	services []corev1.Service,
	namespaceLabels map[string]string,
	waypoint *resolver.WaypointView,
) []resolver.AuthorizationPolicyView {
	if !b.authorizationPoliciesKnown(namespace) {
		return nil
	}
	out := make([]resolver.AuthorizationPolicyView, 0)
	for _, policy := range b.authorizationPolicies {
		if !policy.view.RootNamespace && policy.view.Namespace != namespace {
			continue
		}
		view := policy.view
		if view.HasSelector {
			view.SelectorMatch = matchAnyLabels(policy.selectorLabels, labelSets)
		}
		if len(policy.targetRefs) > 0 {
			if !b.targetRefsKnown(namespace, policy.targetRefs) {
				return nil
			}
			for _, ref := range policy.targetRefs {
				targetView, matches := b.authorizationPolicyTargetView(view, ref, namespace, services, namespaceLabels, waypoint)
				if matches {
					out = append(out, targetView)
				}
			}
			continue
		}
		out = append(out, view)
	}
	return out
}

func (b *workloadBuilder) targetRefsKnown(namespace string, refs []policyTargetRef) bool {
	for _, ref := range refs {
		switch strings.ToLower(ref.kind) {
		case "service":
			if !b.servicesKnown(namespace) {
				return false
			}
		case "gateway":
			// Gateway unavailability is represented on WaypointView so the pure
			// resolver can distinguish unavailable evidence from a missing path.
			continue
		}
	}
	return true
}

func (b *workloadBuilder) authorizationPolicyTargetView(
	view resolver.AuthorizationPolicyView,
	ref policyTargetRef,
	workloadNamespace string,
	services []corev1.Service,
	namespaceLabels map[string]string,
	workloadWaypoint *resolver.WaypointView,
) (resolver.AuthorizationPolicyView, bool) {
	if view.Namespace != workloadNamespace {
		return resolver.AuthorizationPolicyView{}, false
	}
	view.TargetsWaypoint = true
	view.TargetRefKind = ref.kind
	view.TargetRefName = ref.name
	switch strings.ToLower(ref.kind) {
	case "service":
		if ref.group != "" && ref.group != "core" {
			return resolver.AuthorizationPolicyView{}, false
		}
		for _, service := range services {
			if service.Namespace == view.Namespace && service.Name == ref.name {
				view.TargetWaypoint = b.waypointForService(service, namespaceLabels)
				return view, true
			}
		}
	case "gateway":
		if ref.group != "gateway.networking.k8s.io" || workloadWaypoint == nil ||
			workloadWaypoint.Namespace != view.Namespace || workloadWaypoint.Name != ref.name {
			return resolver.AuthorizationPolicyView{}, false
		}
		waypoint := *workloadWaypoint
		view.TargetWaypoint = &waypoint
		return view, true
	}
	return resolver.AuthorizationPolicyView{}, false
}

func authorizationPolicyRequiresL7(policy *securityapi.AuthorizationPolicy) bool {
	for _, rule := range policy.GetRules() {
		if rule == nil {
			continue
		}
		for _, from := range rule.GetFrom() {
			if from == nil || from.GetSource() == nil {
				continue
			}
			source := from.GetSource()
			if len(source.GetRequestPrincipals()) > 0 || len(source.GetNotRequestPrincipals()) > 0 {
				return true
			}
		}
		for _, to := range rule.GetTo() {
			if to == nil || to.GetOperation() == nil {
				continue
			}
			operation := to.GetOperation()
			if len(operation.GetHosts()) > 0 || len(operation.GetNotHosts()) > 0 ||
				len(operation.GetMethods()) > 0 || len(operation.GetNotMethods()) > 0 ||
				len(operation.GetPaths()) > 0 || len(operation.GetNotPaths()) > 0 {
				return true
			}
		}
		for _, condition := range rule.GetWhen() {
			if condition != nil && strings.HasPrefix(condition.GetKey(), "request.") {
				return true
			}
		}
	}
	return false
}

func authorizationPolicyBroadAllow(policy *securityapi.AuthorizationPolicy) bool {
	for _, rule := range policy.GetRules() {
		if rule == nil {
			continue
		}
		if len(rule.GetFrom()) == 0 {
			return true
		}
		for _, from := range rule.GetFrom() {
			if from == nil || from.GetSource() == nil {
				return true
			}
			source := from.GetSource()
			if containsWildcard(source.GetPrincipals()) || containsWildcard(source.GetNamespaces()) ||
				containsWildcard(source.GetServiceAccounts()) || containsWildcard(source.GetRequestPrincipals()) {
				return true
			}
		}
	}
	return false
}

func containsWildcard(values []string) bool {
	for _, value := range values {
		if strings.Contains(value, "*") {
			return true
		}
	}
	return false
}

func authorizationPolicyIdentityScoped(policy *securityapi.AuthorizationPolicy) bool {
	if len(policy.GetRules()) == 0 {
		return false
	}
	for _, rule := range policy.GetRules() {
		if rule == nil || len(rule.GetFrom()) == 0 {
			return false
		}
		for _, from := range rule.GetFrom() {
			if from == nil || !sourceHasExplicitIdentity(from.GetSource()) {
				return false
			}
		}
	}
	return true
}

func sourceHasExplicitIdentity(source *securityapi.Source) bool {
	if source == nil {
		return false
	}
	for _, identities := range [][]string{
		source.GetPrincipals(),
		source.GetNamespaces(),
		source.GetServiceAccounts(),
		source.GetRequestPrincipals(),
	} {
		if len(identities) > 0 && !containsWildcard(identities) {
			return true
		}
	}
	return false
}

func uniformAuthorizationPolicySets(sets [][]resolver.AuthorizationPolicyView) bool {
	for i := 1; i < len(sets); i++ {
		if !sameAuthorizationPolicySet(sets[0], sets[i]) {
			return false
		}
	}
	return true
}

func sameAuthorizationPolicySet(left, right []resolver.AuthorizationPolicyView) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !reflect.DeepEqual(left[i], right[i]) {
			return false
		}
	}
	return true
}

func projectGateways(gateways []gatewayv1.Gateway) []gatewayProjection {
	out := make([]gatewayProjection, 0, len(gateways))
	for _, gateway := range gateways {
		if gateway.Spec.GatewayClassName != gatewayv1.ObjectName(waypointGatewayClass) {
			continue
		}
		scope := gateway.Labels[waypointForLabel]
		if scope == "" {
			scope = "service"
		}
		out = append(out, gatewayProjection{
			name:      gateway.Name,
			namespace: gateway.Namespace,
			scope:     scope,
			ready:     gatewayProgrammed(gateway),
		})
	}
	return out
}

func gatewayProgrammed(gateway gatewayv1.Gateway) bool {
	for _, condition := range gateway.Status.Conditions {
		if condition.Type == string(gatewayv1.GatewayConditionProgrammed) && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func (b *workloadBuilder) waypointFor(
	namespace string,
	workloadLabels map[string]string,
	namespaceLabels map[string]string,
	services []corev1.Service,
) *resolver.WaypointView {
	name, waypointNamespace, scope := selectedWaypointLabel(namespace, workloadLabels, namespaceLabels, services)
	if name == "" {
		return nil
	}
	return b.waypointView(name, waypointNamespace, scope)
}

func (b *workloadBuilder) waypointForService(service corev1.Service, namespaceLabels map[string]string) *resolver.WaypointView {
	name := service.Labels[useWaypointLabel]
	labels := service.Labels
	scope := "service"
	if name == "" {
		name = namespaceLabels[useWaypointLabel]
		labels = namespaceLabels
		scope = "namespace"
	}
	if name == "" {
		return nil
	}
	return b.waypointView(name, waypointNamespace(service.Namespace, labels), scope)
}

func (b *workloadBuilder) waypointView(name, waypointNamespace, scope string) *resolver.WaypointView {
	if !b.gatewaysKnown(waypointNamespace) {
		return &resolver.WaypointView{Name: name, Namespace: waypointNamespace, Known: false, Scope: scope}
	}
	for _, gateway := range b.gateways {
		if gateway.name != name || gateway.namespace != waypointNamespace {
			continue
		}
		return &resolver.WaypointView{
			Name:      name,
			Namespace: waypointNamespace,
			Known:     true,
			Ready:     gateway.ready && gatewaySupportsScope(gateway.scope, scope),
			Scope:     scope,
		}
	}
	return &resolver.WaypointView{Name: name, Namespace: waypointNamespace, Known: true, Ready: false, Scope: scope}
}

func selectedWaypointLabel(
	namespace string,
	workloadLabels map[string]string,
	namespaceLabels map[string]string,
	services []corev1.Service,
) (string, string, string) {
	if name := workloadLabels[useWaypointLabel]; name != "" {
		return name, waypointNamespace(namespace, workloadLabels), "workload"
	}
	for _, service := range services {
		if name := service.Labels[useWaypointLabel]; name != "" {
			return name, waypointNamespace(namespace, service.Labels), "service"
		}
	}
	if name := namespaceLabels[useWaypointLabel]; name != "" {
		return name, waypointNamespace(namespace, namespaceLabels), "namespace"
	}
	return "", "", ""
}

func waypointNamespace(defaultNamespace string, labels map[string]string) string {
	if namespace := labels[useWaypointNamespaceLabel]; namespace != "" {
		return namespace
	}
	return defaultNamespace
}

func gatewaySupportsScope(gatewayScope, selectedScope string) bool {
	switch gatewayScope {
	case "all":
		return true
	case "service":
		return selectedScope == "service" || selectedScope == "namespace"
	case "workload":
		return selectedScope == "workload"
	default:
		return false
	}
}

func (b *workloadBuilder) servicesKnown(namespace string) bool {
	return b.servicesAvailableFor != nil && b.servicesAvailableFor(namespace)
}

func (b *workloadBuilder) destinationRulesKnown(namespace string) bool {
	return b.destinationRulesAvailableFor != nil && b.destinationRulesAvailableFor(namespace)
}

func (b *workloadBuilder) sidecarsKnown(namespace string) bool {
	return b.sidecarsAvailableFor != nil && b.sidecarsAvailableFor(namespace)
}

func (b *workloadBuilder) authorizationPoliciesKnown(namespace string) bool {
	return b.authorizationPoliciesAvailableFor != nil && b.authorizationPoliciesAvailableFor(namespace)
}

func (b *workloadBuilder) gatewaysKnown(namespace string) bool {
	return b.gatewaysAvailableFor != nil && b.gatewaysAvailableFor(namespace)
}
