package collect

import (
	istionetworkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	istiosecurityv1 "istio.io/client-go/pkg/apis/security/v1"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Scope describes whether a scan reads every namespace or a bounded namespace set.
type Scope struct {
	AllNamespaces bool
	Namespaces    []string
	RootNamespace string
}

const DefaultRootNamespace = "istio-system"

// Permission records the evidence available for one attempted API resource list.
type Permission struct {
	APIGroup         string
	Resource         string
	Verbs            []string
	Granted          bool
	Optional         bool
	Impact           string
	AffectedControls []string
	DeniedScopes     []string `json:"-"`
}

// Snapshot is the raw typed-resource bundle returned by collectors.
type Snapshot struct {
	RootNamespace                   string
	Namespaces                      []corev1.Namespace
	Pods                            []corev1.Pod
	PodAvailability                 PeerAuthenticationAvailability
	Deployments                     []appsv1.Deployment
	ReplicaSets                     []appsv1.ReplicaSet
	ReplicaSetAvailability          PeerAuthenticationAvailability
	StatefulSets                    []appsv1.StatefulSet
	DaemonSets                      []appsv1.DaemonSet
	Services                        []corev1.Service
	ServiceAvailability             ScopedAvailability
	EndpointSlices                  []discoveryv1.EndpointSlice
	EndpointSliceAvailability       ScopedAvailability
	PeerAuthentications             []*istiosecurityv1beta1.PeerAuthentication
	PeerAuthAvailability            ScopedAvailability
	DestinationRules                []*istionetworkingv1.DestinationRule
	DestinationRuleAvailability     ScopedAvailability
	Sidecars                        []*istionetworkingv1.Sidecar
	SidecarAvailability             ScopedAvailability
	AuthorizationPolicies           []*istiosecurityv1.AuthorizationPolicy
	AuthorizationPolicyAvailability ScopedAvailability
	Gateways                        []gatewayv1.Gateway
	GatewayAvailability             ScopedAvailability
	PermissionSummary               []Permission
}

// ScopedAvailability records which namespace list scopes were available.
type ScopedAvailability struct {
	AllNamespaces bool
	Namespaces    map[string]bool
}

// PeerAuthenticationAvailability remains an alias for compatibility with
// existing hand-built snapshots and tests.
type PeerAuthenticationAvailability = ScopedAvailability

// PeerAuthenticationsAvailable reports whether the scanner could list Istio
// PeerAuthentication resources. A missing CRD or permission denial makes mTLS
// posture unknown rather than silently defaulting.
func (s Snapshot) PeerAuthenticationsAvailable() bool {
	if s.hasPeerAuthenticationAvailabilityDetails() {
		if s.PeerAuthAvailability.AllNamespaces {
			return true
		}
		seen := false
		for _, available := range s.PeerAuthAvailability.Namespaces {
			seen = true
			if !available {
				return false
			}
		}
		return seen
	}

	seen := false
	for _, permission := range s.PermissionSummary {
		if permission.APIGroup == "security.istio.io" && permission.Resource == "peerauthentications" {
			seen = true
			if !permission.Granted {
				return false
			}
		}
	}
	return seen
}

// PeerAuthenticationsAvailableFor reports whether PeerAuthentication evidence is
// sufficient for a workload namespace under the M1 mesh-wide/namespace-only
// resolver. It falls back to the aggregate permission summary for hand-built
// tests and future callers that do not populate per-scope availability.
func (s Snapshot) PeerAuthenticationsAvailableFor(namespace, rootNamespace string) bool {
	if !s.hasPeerAuthenticationAvailabilityDetails() {
		return s.PeerAuthenticationsAvailable()
	}
	if s.PeerAuthAvailability.AllNamespaces {
		return true
	}
	if rootNamespace == "" {
		rootNamespace = DefaultRootNamespace
	}

	needed := map[string]struct{}{
		rootNamespace: {},
		namespace:     {},
	}
	for ns := range needed {
		if !s.PeerAuthAvailability.Namespaces[ns] {
			return false
		}
	}
	return true
}

func (s Snapshot) hasPeerAuthenticationAvailabilityDetails() bool {
	return s.PeerAuthAvailability.AllNamespaces || len(s.PeerAuthAvailability.Namespaces) > 0
}

// ServicesAvailableFor reports whether Service selection and Service-bound
// port evidence were collected for a workload namespace.
func (s Snapshot) ServicesAvailableFor(namespace string) bool {
	return s.scopedResourceAvailableFor(s.ServiceAvailability, []string{namespace}, "", "services")
}

// EndpointSlicesAvailableFor reports whether selectorless Service endpoint
// attachment evidence was collected for a workload namespace.
func (s Snapshot) EndpointSlicesAvailableFor(namespace string) bool {
	return s.scopedResourceAvailableFor(s.EndpointSliceAvailability, []string{namespace}, "discovery.k8s.io", "endpointslices")
}

// DestinationRulesAvailableFor reports whether DestinationRule evidence was
// collected from both the workload and root configuration namespaces.
func (s Snapshot) DestinationRulesAvailableFor(namespace, rootNamespace string) bool {
	if rootNamespace == "" {
		rootNamespace = DefaultRootNamespace
	}
	return s.scopedResourceAvailableFor(
		s.DestinationRuleAvailability,
		uniqueScopes(namespace, rootNamespace),
		"networking.istio.io",
		"destinationrules",
	)
}

// SidecarsAvailableFor reports whether workload-namespace and mesh-root
// Sidecar resource scoping evidence was collected.
func (s Snapshot) SidecarsAvailableFor(namespace, rootNamespace string) bool {
	if rootNamespace == "" {
		rootNamespace = DefaultRootNamespace
	}
	return s.scopedResourceAvailableFor(
		s.SidecarAvailability,
		uniqueScopes(namespace, rootNamespace),
		"networking.istio.io",
		"sidecars",
	)
}

// AuthorizationPoliciesAvailableFor reports whether mesh-root and workload
// namespace AuthorizationPolicy evidence was collected.
func (s Snapshot) AuthorizationPoliciesAvailableFor(namespace, rootNamespace string) bool {
	if rootNamespace == "" {
		rootNamespace = DefaultRootNamespace
	}
	return s.scopedResourceAvailableFor(
		s.AuthorizationPolicyAvailability,
		uniqueScopes(namespace, rootNamespace),
		"security.istio.io",
		"authorizationpolicies",
	)
}

// GatewaysAvailableFor reports whether Gateway API waypoint evidence was
// collected for the namespace that owns a selected waypoint.
func (s Snapshot) GatewaysAvailableFor(namespace string) bool {
	return s.scopedResourceAvailableFor(s.GatewayAvailability, []string{namespace}, "gateway.networking.k8s.io", "gateways")
}

func (s Snapshot) scopedResourceAvailableFor(
	availability ScopedAvailability,
	namespaces []string,
	apiGroup string,
	resource string,
) bool {
	if hasScopedAvailabilityDetails(availability) {
		if availability.AllNamespaces {
			return true
		}
		for _, namespace := range namespaces {
			if !availability.Namespaces[namespace] {
				return false
			}
		}
		return true
	}
	available, seen := s.resourcePermissionAvailable(apiGroup, resource)
	return seen && available
}

// PodsAvailableFor reports whether pod evidence was available for the namespace.
// Hand-built snapshots without permission metadata default to available so unit
// tests can provide explicit pods without recreating collector bookkeeping.
func (s Snapshot) PodsAvailableFor(namespace string) bool {
	if hasScopedAvailabilityDetails(s.PodAvailability) {
		return scopedAvailableFor(s.PodAvailability, namespace)
	}
	available, seen := s.resourcePermissionAvailable("", "pods")
	if !seen {
		return true
	}
	return available
}

// ReplicaSetsAvailableFor reports whether ReplicaSet ownership evidence was
// available for Deployment pod matching in the namespace.
func (s Snapshot) ReplicaSetsAvailableFor(namespace string) bool {
	if hasScopedAvailabilityDetails(s.ReplicaSetAvailability) {
		return scopedAvailableFor(s.ReplicaSetAvailability, namespace)
	}
	available, seen := s.resourcePermissionAvailable("apps", "replicasets")
	if !seen {
		return true
	}
	return available
}

// ClientProxiesAvailable reports whether every resource collection used to
// discover sidecar client proxies was complete. A gap in any scanned namespace
// can hide a client-selected DestinationRule, so contradiction evidence must
// remain unavailable rather than becoming a known false.
func (s Snapshot) ClientProxiesAvailable() bool {
	if scopedAvailabilityHasDenial(s.PodAvailability) ||
		scopedAvailabilityHasDenial(s.ReplicaSetAvailability) {
		return false
	}
	resources := []struct {
		apiGroup string
		resource string
	}{
		{resource: "pods"},
		{apiGroup: "apps", resource: "deployments"},
		{apiGroup: "apps", resource: "replicasets"},
		{apiGroup: "apps", resource: "statefulsets"},
		{apiGroup: "apps", resource: "daemonsets"},
	}
	for _, candidate := range resources {
		available, seen := s.resourcePermissionAvailable(candidate.apiGroup, candidate.resource)
		if seen && !available {
			return false
		}
	}
	return true
}

func hasScopedAvailabilityDetails(availability ScopedAvailability) bool {
	return availability.AllNamespaces || len(availability.Namespaces) > 0
}

func scopedAvailabilityHasDenial(availability ScopedAvailability) bool {
	if availability.AllNamespaces {
		return false
	}
	for _, available := range availability.Namespaces {
		if !available {
			return true
		}
	}
	return false
}

func scopedAvailableFor(availability ScopedAvailability, namespace string) bool {
	if availability.AllNamespaces {
		return true
	}
	return availability.Namespaces[namespace]
}

func uniqueScopes(values ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (s Snapshot) resourcePermissionAvailable(apiGroup, resource string) (bool, bool) {
	seen := false
	for _, permission := range s.PermissionSummary {
		if permission.APIGroup != apiGroup || permission.Resource != resource {
			continue
		}
		seen = true
		if !permission.Granted {
			return false, true
		}
	}
	return seen, seen
}
