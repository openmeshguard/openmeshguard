package normalize

import (
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	securityapi "istio.io/api/security/v1beta1"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildNormalizesWorkloadsPeerAuthenticationsAndSidecarMode(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "payments",
				Labels: map[string]string{"istio-injection": "enabled"},
			},
		}},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}},
				},
			},
		}},
		Pods: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-1",
				Namespace: "payments",
				Labels:    map[string]string{"app": "api"},
				OwnerReferences: []metav1.OwnerReference{{
					Kind: "ReplicaSet",
					Name: "api-abc123",
				}},
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}}},
		}},
		PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "istio-system"},
				Spec: securityapi.PeerAuthentication{
					Mtls: &securityapi.PeerAuthentication_MutualTLS{Mode: securityapi.PeerAuthentication_MutualTLS_STRICT},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "payments"},
				Spec: securityapi.PeerAuthentication{
					Mtls: &securityapi.PeerAuthentication_MutualTLS{Mode: securityapi.PeerAuthentication_MutualTLS_PERMISSIVE},
				},
			},
		},
		PermissionSummary: []collect.Permission{{
			APIGroup: "security.istio.io",
			Resource: "peerauthentications",
			Verbs:    []string{"list"},
			Granted:  true,
		}},
	})

	if result.Inventory.DataPlaneMode != resolver.ModeSidecar {
		t.Fatalf("inventory data plane mode = %q, want sidecar", result.Inventory.DataPlaneMode)
	}
	if len(result.Workloads) != 1 {
		t.Fatalf("workloads = %d, want 1", len(result.Workloads))
	}
	workload := result.Workloads[0]
	if workload.DataPlaneMode != resolver.ModeSidecar {
		t.Fatalf("workload data plane mode = %q, want sidecar", workload.DataPlaneMode)
	}
	if !workload.MeshDefaults.Known {
		t.Fatal("MeshDefaults.Known = false, want true")
	}
	if workload.MeshDefaults.MeshMTLSMode != "STRICT" {
		t.Fatalf("mesh mTLS mode = %q, want STRICT", workload.MeshDefaults.MeshMTLSMode)
	}
	if len(workload.PeerAuthN) != 2 {
		t.Fatalf("peer authentications = %#v, want mesh and namespace", workload.PeerAuthN)
	}
}

func TestBuildAmbientStubReturnsUnknown(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "ambient",
				Labels: map[string]string{"istio.io/dataplane-mode": "ambient"},
			},
		}},
		Pods: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "ambient", Labels: map[string]string{"app": "api"}},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
		}},
		PermissionSummary: []collect.Permission{{
			APIGroup: "security.istio.io",
			Resource: "peerauthentications",
			Verbs:    []string{"list"},
			Granted:  true,
		}},
	})

	if len(result.Workloads) != 1 {
		t.Fatalf("workloads = %d, want 1", len(result.Workloads))
	}
	if result.Workloads[0].DataPlaneMode != resolver.ModeUnknown {
		t.Fatalf("ambient stub mode = %q, want unknown", result.Workloads[0].DataPlaneMode)
	}
}
