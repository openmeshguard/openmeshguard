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
	Namespaces          []corev1.Namespace
	Pods                []corev1.Pod
	Deployments         []appsv1.Deployment
	ReplicaSets         []appsv1.ReplicaSet
	StatefulSets        []appsv1.StatefulSet
	DaemonSets          []appsv1.DaemonSet
	Services            []corev1.Service
	PeerAuthentications []*istiosecurityv1beta1.PeerAuthentication
	PermissionSummary   []Permission
}

// PeerAuthenticationsAvailable reports whether the scanner could list Istio
// PeerAuthentication resources. A missing CRD or permission denial makes mTLS
// posture unknown rather than silently defaulting.
func (s Snapshot) PeerAuthenticationsAvailable() bool {
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
