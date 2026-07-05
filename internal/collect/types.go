package collect

import (
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
}

// Snapshot is the raw typed-resource bundle returned by collectors.
type Snapshot struct {
	RootNamespace        string
	Namespaces           []corev1.Namespace
	Pods                 []corev1.Pod
	Deployments          []appsv1.Deployment
	ReplicaSets          []appsv1.ReplicaSet
	StatefulSets         []appsv1.StatefulSet
	DaemonSets           []appsv1.DaemonSet
	Services             []corev1.Service
	PeerAuthentications  []*istiosecurityv1beta1.PeerAuthentication
	PeerAuthAvailability PeerAuthenticationAvailability
	PermissionSummary    []Permission
}

// PeerAuthenticationAvailability records which PeerAuthentication list scopes
// were available. Scoped scans need both root-namespace and workload-namespace
// evidence to resolve M1 mTLS posture.
type PeerAuthenticationAvailability struct {
	AllNamespaces bool
	Namespaces    map[string]bool
}

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
