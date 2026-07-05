package normalize

import (
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	securityapi "istio.io/api/security/v1beta1"
	typeapi "istio.io/api/type/v1beta1"
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
		ReplicaSets: []appsv1.ReplicaSet{
			deploymentReplicaSet("payments", "api-abc123", "api"),
		},
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
	if len(workload.PeerAuthN) != 2 {
		t.Fatalf("peer authentications = %#v, want mesh and namespace", workload.PeerAuthN)
	}
}

func TestBuildMatchesControllerPodsWithMatchExpressions(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{{
			ObjectMeta: metav1.ObjectMeta{Name: "payments"},
		}},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{{
						Key:      "app",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"api"},
					}},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}},
				},
			},
		}},
		ReplicaSets: []appsv1.ReplicaSet{
			deploymentReplicaSet("payments", "api-abc123", "api"),
		},
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
	if result.Workloads[0].DataPlaneMode != resolver.ModeSidecar {
		t.Fatalf("mode = %q, want sidecar from matched pod", result.Workloads[0].DataPlaneMode)
	}
}

func TestBuildDetectsIstioProxyNativeSidecarInitContainer(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{{
			ObjectMeta: metav1.ObjectMeta{Name: "payments"},
		}},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}},
					Spec: corev1.PodSpec{
						Containers:     []corev1.Container{{Name: "api"}},
						InitContainers: []corev1.Container{{Name: "istio-proxy"}},
					},
				},
			},
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
	if result.Workloads[0].DataPlaneMode != resolver.ModeSidecar {
		t.Fatalf("mode = %q, want sidecar from init container", result.Workloads[0].DataPlaneMode)
	}
}

func TestBuildMarksObservedPodsWithoutProxyUnknownDespiteInjectionLabels(t *testing.T) {
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
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
				},
			},
		}},
		ReplicaSets: []appsv1.ReplicaSet{
			deploymentReplicaSet("payments", "api-abc123", "api"),
		},
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
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
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
		t.Fatalf("mode = %q, want unknown for observed pod without proxy", result.Workloads[0].DataPlaneMode)
	}
}

func TestBuildMarksWorkloadUnknownWhenPodEvidenceUnavailable(t *testing.T) {
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
				Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
			},
		}},
		PodAvailability: collect.PeerAuthenticationAvailability{
			Namespaces: map[string]bool{"payments": false},
		},
		PermissionSummary: []collect.Permission{
			{Resource: "pods", Verbs: []string{"list"}, Granted: false},
			{APIGroup: "security.istio.io", Resource: "peerauthentications", Verbs: []string{"list"}, Granted: true},
		},
	})

	if len(result.Workloads) != 1 {
		t.Fatalf("workloads = %d, want 1", len(result.Workloads))
	}
	if result.Workloads[0].DataPlaneMode != resolver.ModeUnknown {
		t.Fatalf("mode = %q, want unknown when pod evidence is unavailable", result.Workloads[0].DataPlaneMode)
	}
}

func TestBuildMarksMixedObservedProxyEvidence(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{{
			ObjectMeta: metav1.ObjectMeta{Name: "payments"},
		}},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
			},
		}},
		ReplicaSets: []appsv1.ReplicaSet{
			deploymentReplicaSet("payments", "api-abc123", "api"),
		},
		Pods: []corev1.Pod{
			{
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
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "api-2",
					Namespace: "payments",
					Labels:    map[string]string{"app": "api"},
					OwnerReferences: []metav1.OwnerReference{{
						Kind: "ReplicaSet",
						Name: "api-abc123",
					}},
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
			},
		},
		PodAvailability: collect.PeerAuthenticationAvailability{
			Namespaces: map[string]bool{"payments": true},
		},
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
	if result.Workloads[0].DataPlaneMode != resolver.ModeMixed {
		t.Fatalf("mode = %q, want mixed for injected and uninjected observed pods", result.Workloads[0].DataPlaneMode)
	}
}

func TestBuildScopesPeerAuthenticationAvailabilityToWorkloadNamespace(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "payments"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "orders"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "istio-system"}},
		},
		Deployments: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
					Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "orders"},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
					Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
				},
			},
		},
		PeerAuthAvailability: collect.PeerAuthenticationAvailability{
			Namespaces: map[string]bool{
				"istio-system": true,
				"payments":     true,
				"orders":       false,
			},
		},
		PermissionSummary: []collect.Permission{{
			APIGroup: "security.istio.io",
			Resource: "peerauthentications",
			Verbs:    []string{"list"},
			Granted:  false,
		}},
	})

	workloads := map[string]resolver.WorkloadInput{}
	for _, workload := range result.Workloads {
		workloads[workload.Ref.Namespace] = workload
	}
	if !workloads["payments"].MeshDefaults.Known {
		t.Fatal("payments MeshDefaults.Known = false after payments/root evidence succeeded")
	}
	if workloads["orders"].MeshDefaults.Known {
		t.Fatal("orders MeshDefaults.Known = true after orders PeerAuthentication denial")
	}
}

func TestBuildIncludesRootNamespaceSelectorPeerAuthentication(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "payments"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "istio-system"}},
		},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
			},
		}},
		PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{{
			ObjectMeta: metav1.ObjectMeta{Name: "api-override", Namespace: "istio-system"},
			Spec: securityapi.PeerAuthentication{
				Selector: &typeapi.WorkloadSelector{MatchLabels: map[string]string{"app": "api"}},
				Mtls:     &securityapi.PeerAuthentication_MutualTLS{Mode: securityapi.PeerAuthentication_MutualTLS_DISABLE},
			},
		}},
		PeerAuthAvailability: collect.PeerAuthenticationAvailability{
			Namespaces: map[string]bool{
				"istio-system": true,
				"payments":     true,
			},
		},
	})

	if len(result.Workloads) != 1 {
		t.Fatalf("workloads = %d, want 1", len(result.Workloads))
	}
	peerAuthentications := result.Workloads[0].PeerAuthN
	if len(peerAuthentications) != 1 {
		t.Fatalf("peer authentications = %#v, want root selector policy", peerAuthentications)
	}
	if !peerAuthentications[0].SelectorMatch {
		t.Fatal("root namespace selector PeerAuthentication was included without SelectorMatch")
	}
	if peerAuthentications[0].Namespace != "istio-system" {
		t.Fatalf("peer authentication namespace = %q, want istio-system", peerAuthentications[0].Namespace)
	}
}

func TestBuildUsesConfiguredRootNamespace(t *testing.T) {
	result := Build(collect.Snapshot{
		RootNamespace: "istio-config",
		Namespaces: []corev1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "payments"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "istio-config"}},
		},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
			},
		}},
		PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "istio-config"},
			Spec: securityapi.PeerAuthentication{
				Mtls: &securityapi.PeerAuthentication_MutualTLS{Mode: securityapi.PeerAuthentication_MutualTLS_STRICT},
			},
		}},
		PeerAuthAvailability: collect.PeerAuthenticationAvailability{
			Namespaces: map[string]bool{
				"istio-config": true,
				"payments":     true,
			},
		},
	})

	if len(result.Workloads) != 1 {
		t.Fatalf("workloads = %d, want 1", len(result.Workloads))
	}
	workload := result.Workloads[0]
	if workload.MeshDefaults.RootNamespace != "istio-config" {
		t.Fatalf("root namespace = %q, want istio-config", workload.MeshDefaults.RootNamespace)
	}
	if len(workload.PeerAuthN) != 1 || workload.PeerAuthN[0].Namespace != "istio-config" {
		t.Fatalf("peer authentications = %#v, want configured root policy", workload.PeerAuthN)
	}
}

func TestBuildKeepsOwnedPodsWithoutNormalizedController(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{{
			ObjectMeta: metav1.ObjectMeta{Name: "payments"},
		}},
		Pods: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "migrate-1",
				Namespace: "payments",
				Labels:    map[string]string{"job-name": "migrate"},
				OwnerReferences: []metav1.OwnerReference{{
					Kind: "Job",
					Name: "migrate",
				}},
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "migrate"}, {Name: "istio-proxy"}}},
		}},
		PermissionSummary: []collect.Permission{{
			APIGroup: "security.istio.io",
			Resource: "peerauthentications",
			Verbs:    []string{"list"},
			Granted:  true,
		}},
	})

	if len(result.Workloads) != 1 {
		t.Fatalf("workloads = %d, want pod workload", len(result.Workloads))
	}
	workload := result.Workloads[0]
	if workload.Ref.Kind != "Pod" || workload.Ref.Name != "migrate-1" {
		t.Fatalf("workload ref = %#v, want Pod payments/migrate-1", workload.Ref)
	}
}

func TestBuildDoesNotCoverUnrelatedPodsBySelectorOnly(t *testing.T) {
	result := Build(collect.Snapshot{
		Namespaces: []corev1.Namespace{{
			ObjectMeta: metav1.ObjectMeta{Name: "payments"},
		}},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
				Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
			},
		}},
		Pods: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-migrate-1",
				Namespace: "payments",
				Labels:    map[string]string{"app": "api"},
				OwnerReferences: []metav1.OwnerReference{{
					Kind: "Job",
					Name: "api-migrate",
				}},
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "migrate"}, {Name: "istio-proxy"}}},
		}},
		PodAvailability: collect.PeerAuthenticationAvailability{
			Namespaces: map[string]bool{"payments": true},
		},
		PermissionSummary: []collect.Permission{{
			APIGroup: "security.istio.io",
			Resource: "peerauthentications",
			Verbs:    []string{"list"},
			Granted:  true,
		}},
	})

	seen := map[string]bool{}
	for _, workload := range result.Workloads {
		seen[workload.Ref.Kind+"/"+workload.Ref.Name] = true
	}
	for _, key := range []string{"Deployment/api", "Pod/api-migrate-1"} {
		if !seen[key] {
			t.Fatalf("missing workload %s in %#v", key, result.Workloads)
		}
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

func deploymentReplicaSet(namespace, name, deployment string) appsv1.ReplicaSet {
	return appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "Deployment",
				Name: deployment,
			}},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": deployment}},
		},
	}
}
