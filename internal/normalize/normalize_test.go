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

type buildCase struct {
	name     string
	snapshot collect.Snapshot
	assert   func(*testing.T, Result)
}

func TestBuildWorkloadNormalization(t *testing.T) {
	runBuildCases(t, []buildCase{
		{
			name: "normalizes workloads PeerAuthentications and sidecar mode",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("payments", map[string]string{"istio-injection": "enabled"})},
				Deployments: []appsv1.Deployment{
					deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{}),
				},
				ReplicaSets: []appsv1.ReplicaSet{deploymentReplicaSet("payments", "api-abc123", "api")},
				Pods: []corev1.Pod{
					podForReplicaSet("payments", "api-1", "api-abc123", map[string]string{"app": "api"}, corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}},
					}),
				},
				PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{
					peerAuthentication("istio-system", "default", nil, securityapi.PeerAuthentication_MutualTLS_STRICT),
					peerAuthentication("payments", "default", nil, securityapi.PeerAuthentication_MutualTLS_PERMISSIVE),
				},
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				if result.Inventory.DataPlaneMode != resolver.ModeSidecar {
					t.Fatalf("inventory data plane mode = %q, want sidecar", result.Inventory.DataPlaneMode)
				}
				workload := singleWorkload(t, result)
				if workload.DataPlaneMode != resolver.ModeSidecar {
					t.Fatalf("workload data plane mode = %q, want sidecar", workload.DataPlaneMode)
				}
				if !workload.MeshDefaults.Known {
					t.Fatal("MeshDefaults.Known = false, want true")
				}
				if len(workload.PeerAuthN) != 2 {
					t.Fatalf("peer authentications = %#v, want mesh and namespace", workload.PeerAuthN)
				}
			},
		},
		{
			name: "matches controller pods with match expressions",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("payments", nil)},
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
						Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
					},
				}},
				ReplicaSets: []appsv1.ReplicaSet{deploymentReplicaSet("payments", "api-abc123", "api")},
				Pods: []corev1.Pod{
					podForReplicaSet("payments", "api-1", "api-abc123", map[string]string{"app": "api"}, corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}},
					}),
				},
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				if singleWorkload(t, result).DataPlaneMode != resolver.ModeSidecar {
					t.Fatalf("mode = %q, want sidecar from matched pod", result.Workloads[0].DataPlaneMode)
				}
			},
		},
		{
			name: "detects istio-proxy native sidecar init container",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("payments", nil)},
				Deployments: []appsv1.Deployment{
					deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{
						Containers:     []corev1.Container{{Name: "api"}},
						InitContainers: []corev1.Container{{Name: "istio-proxy"}},
					}),
				},
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				if singleWorkload(t, result).DataPlaneMode != resolver.ModeSidecar {
					t.Fatalf("mode = %q, want sidecar from init container", result.Workloads[0].DataPlaneMode)
				}
			},
		},
		{
			name: "marks observed pods without proxy unknown despite injection labels",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("payments", map[string]string{"istio-injection": "enabled"})},
				Deployments: []appsv1.Deployment{
					deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api"}},
					}),
				},
				ReplicaSets: []appsv1.ReplicaSet{deploymentReplicaSet("payments", "api-abc123", "api")},
				Pods: []corev1.Pod{
					podForReplicaSet("payments", "api-1", "api-abc123", map[string]string{"app": "api"}, corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api"}},
					}),
				},
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				if singleWorkload(t, result).DataPlaneMode != resolver.ModeUnknown {
					t.Fatalf("mode = %q, want unknown for observed pod without proxy", result.Workloads[0].DataPlaneMode)
				}
			},
		},
		{
			name: "marks workload unknown when pod evidence unavailable",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("payments", map[string]string{"istio-injection": "enabled"})},
				Deployments: []appsv1.Deployment{
					deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{}),
				},
				PodAvailability: collect.PeerAuthenticationAvailability{
					Namespaces: map[string]bool{"payments": false},
				},
				PermissionSummary: []collect.Permission{
					{Resource: "pods", Verbs: []string{"list"}, Granted: false},
					{APIGroup: "security.istio.io", Resource: "peerauthentications", Verbs: []string{"list"}, Granted: true},
				},
			},
			assert: func(t *testing.T, result Result) {
				if singleWorkload(t, result).DataPlaneMode != resolver.ModeUnknown {
					t.Fatalf("mode = %q, want unknown when pod evidence is unavailable", result.Workloads[0].DataPlaneMode)
				}
			},
		},
		{
			name: "marks Deployment unknown when ReplicaSet ownership evidence unavailable",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("payments", map[string]string{"istio-injection": "enabled"})},
				Deployments: []appsv1.Deployment{
					deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{}),
				},
				Pods: []corev1.Pod{
					podForReplicaSet("payments", "api-1", "api-abc123", map[string]string{"app": "api"}, corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}},
					}),
				},
				PodAvailability:        collect.PeerAuthenticationAvailability{Namespaces: map[string]bool{"payments": true}},
				ReplicaSetAvailability: collect.PeerAuthenticationAvailability{Namespaces: map[string]bool{"payments": false}},
				PermissionSummary:      peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				workloads := workloadsByKindName(result)
				deployment := workloads["Deployment/api"]
				if deployment.Ref.Name == "" {
					t.Fatalf("missing Deployment/api in %#v", result.Workloads)
				}
				if deployment.DataPlaneMode != resolver.ModeUnknown {
					t.Fatalf("deployment mode = %q, want unknown when ReplicaSet evidence is unavailable", deployment.DataPlaneMode)
				}
			},
		},
		{
			name: "marks mixed observed proxy evidence",
			snapshot: collect.Snapshot{
				Namespaces:  []corev1.Namespace{namespace("payments", nil)},
				Deployments: []appsv1.Deployment{deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{})},
				ReplicaSets: []appsv1.ReplicaSet{deploymentReplicaSet("payments", "api-abc123", "api")},
				Pods: []corev1.Pod{
					podForReplicaSet("payments", "api-1", "api-abc123", map[string]string{"app": "api"}, corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}},
					}),
					podForReplicaSet("payments", "api-2", "api-abc123", map[string]string{"app": "api"}, corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api"}},
					}),
				},
				PodAvailability:   collect.PeerAuthenticationAvailability{Namespaces: map[string]bool{"payments": true}},
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				if singleWorkload(t, result).DataPlaneMode != resolver.ModeMixed {
					t.Fatalf("mode = %q, want mixed for injected and uninjected observed pods", result.Workloads[0].DataPlaneMode)
				}
			},
		},
		{
			name: "ambient stub returns unknown",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("ambient", map[string]string{"istio.io/dataplane-mode": "ambient"})},
				Pods: []corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "ambient", Labels: map[string]string{"app": "api"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
				}},
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				if singleWorkload(t, result).DataPlaneMode != resolver.ModeUnknown {
					t.Fatalf("ambient stub mode = %q, want unknown", result.Workloads[0].DataPlaneMode)
				}
			},
		},
	})
}

func TestBuildPeerAuthenticationNormalization(t *testing.T) {
	runBuildCases(t, []buildCase{
		{
			name: "scopes PeerAuthentication availability to workload namespace",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{
					namespace("payments", nil),
					namespace("orders", nil),
					namespace("istio-system", nil),
				},
				Deployments: []appsv1.Deployment{
					deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{}),
					deployment("orders", "api", map[string]string{"app": "api"}, corev1.PodSpec{}),
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
			},
			assert: func(t *testing.T, result Result) {
				workloads := workloadsByNamespace(result)
				if !workloads["payments"].MeshDefaults.Known {
					t.Fatal("payments MeshDefaults.Known = false after payments/root evidence succeeded")
				}
				if workloads["orders"].MeshDefaults.Known {
					t.Fatal("orders MeshDefaults.Known = true after orders PeerAuthentication denial")
				}
			},
		},
		{
			name: "includes root namespace selector PeerAuthentication",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{
					namespace("payments", nil),
					namespace("istio-system", nil),
				},
				Deployments: []appsv1.Deployment{deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{})},
				PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{
					peerAuthentication("istio-system", "api-override", map[string]string{"app": "api"}, securityapi.PeerAuthentication_MutualTLS_DISABLE),
				},
				PeerAuthAvailability: rootAndNamespacePeerAuthenticationAvailability("payments", "istio-system"),
			},
			assert: func(t *testing.T, result Result) {
				peerAuthentications := singleWorkload(t, result).PeerAuthN
				if len(peerAuthentications) != 1 {
					t.Fatalf("peer authentications = %#v, want root selector policy", peerAuthentications)
				}
				if !peerAuthentications[0].SelectorMatch {
					t.Fatal("root namespace selector PeerAuthentication was included without SelectorMatch")
				}
				if peerAuthentications[0].Namespace != "istio-system" {
					t.Fatalf("peer authentication namespace = %q, want istio-system", peerAuthentications[0].Namespace)
				}
			},
		},
		{
			name: "includes selector PeerAuthentication matched by observed controller pod",
			snapshot: collect.Snapshot{
				Namespaces:  []corev1.Namespace{namespace("payments", nil)},
				Deployments: []appsv1.Deployment{deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{})},
				ReplicaSets: []appsv1.ReplicaSet{deploymentReplicaSet("payments", "api-abc123", "api")},
				Pods: []corev1.Pod{
					podForReplicaSet("payments", "api-1", "api-abc123", map[string]string{
						"app":               "api",
						"pod-template-hash": "abc123",
					}, corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}}}),
				},
				PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{
					peerAuthentication("payments", "pod-selected", map[string]string{"pod-template-hash": "abc123"}, securityapi.PeerAuthentication_MutualTLS_STRICT),
				},
				PeerAuthAvailability: rootAndNamespacePeerAuthenticationAvailability("payments", "istio-system"),
			},
			assert: func(t *testing.T, result Result) {
				workload := singleWorkload(t, result)
				peerAuthentications := workload.PeerAuthN
				if len(peerAuthentications) != 1 {
					t.Fatalf("peer authentications = %#v, want observed pod selector policy", peerAuthentications)
				}
				if !peerAuthentications[0].SelectorMatch {
					t.Fatal("observed pod selector PeerAuthentication was included without SelectorMatch")
				}

				mtls := resolver.New().ResolveMTLS(workload)
				if mtls.Effective != resolver.MTLSStrict {
					t.Fatalf("effective mTLS = %q, want strict for selector PeerAuthentication", mtls.Effective)
				}
			},
		},
		{
			name: "uses configured root namespace",
			snapshot: collect.Snapshot{
				RootNamespace: "istio-config",
				Namespaces: []corev1.Namespace{
					namespace("payments", nil),
					namespace("istio-config", nil),
				},
				Deployments: []appsv1.Deployment{deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{})},
				PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{
					peerAuthentication("istio-config", "default", nil, securityapi.PeerAuthentication_MutualTLS_STRICT),
				},
				PeerAuthAvailability: rootAndNamespacePeerAuthenticationAvailability("payments", "istio-config"),
			},
			assert: func(t *testing.T, result Result) {
				workload := singleWorkload(t, result)
				if workload.MeshDefaults.RootNamespace != "istio-config" {
					t.Fatalf("root namespace = %q, want istio-config", workload.MeshDefaults.RootNamespace)
				}
				if len(workload.PeerAuthN) != 1 || workload.PeerAuthN[0].Namespace != "istio-config" {
					t.Fatalf("peer authentications = %#v, want configured root policy", workload.PeerAuthN)
				}
			},
		},
	})
}

func TestBuildPodOwnership(t *testing.T) {
	runBuildCases(t, []buildCase{
		{
			name: "keeps owned pods without normalized controller",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("payments", nil)},
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
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				workload := singleWorkload(t, result)
				if workload.Ref.Kind != "Pod" || workload.Ref.Name != "migrate-1" {
					t.Fatalf("workload ref = %#v, want Pod payments/migrate-1", workload.Ref)
				}
			},
		},
		{
			name: "does not cover unrelated pods by selector only",
			snapshot: collect.Snapshot{
				Namespaces:  []corev1.Namespace{namespace("payments", nil)},
				Deployments: []appsv1.Deployment{deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{})},
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
				PodAvailability:   collect.PeerAuthenticationAvailability{Namespaces: map[string]bool{"payments": true}},
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				seen := map[string]bool{}
				for _, workload := range result.Workloads {
					seen[workload.Ref.Kind+"/"+workload.Ref.Name] = true
				}
				for _, key := range []string{"Deployment/api", "Pod/api-migrate-1"} {
					if !seen[key] {
						t.Fatalf("missing workload %s in %#v", key, result.Workloads)
					}
				}
			},
		},
	})
}

func runBuildCases(t *testing.T, tests []buildCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assert(t, Build(tt.snapshot))
		})
	}
}

func singleWorkload(t *testing.T, result Result) resolver.WorkloadInput {
	t.Helper()

	if len(result.Workloads) != 1 {
		t.Fatalf("workloads = %d, want 1", len(result.Workloads))
	}
	return result.Workloads[0]
}

func workloadsByNamespace(result Result) map[string]resolver.WorkloadInput {
	out := map[string]resolver.WorkloadInput{}
	for _, workload := range result.Workloads {
		out[workload.Ref.Namespace] = workload
	}
	return out
}

func workloadsByKindName(result Result) map[string]resolver.WorkloadInput {
	out := map[string]resolver.WorkloadInput{}
	for _, workload := range result.Workloads {
		out[workload.Ref.Kind+"/"+workload.Ref.Name] = workload
	}
	return out
}

func namespace(name string, labels map[string]string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func deployment(namespace, name string, labels map[string]string, spec corev1.PodSpec) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       spec,
			},
		},
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

func podForReplicaSet(namespace, name, replicaSet string, labels map[string]string, spec corev1.PodSpec) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "ReplicaSet",
				Name: replicaSet,
			}},
		},
		Spec: spec,
	}
}

func peerAuthentication(
	namespace string,
	name string,
	selectorLabels map[string]string,
	mode securityapi.PeerAuthentication_MutualTLS_Mode,
) *istiosecurityv1beta1.PeerAuthentication {
	peerAuthentication := &istiosecurityv1beta1.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: securityapi.PeerAuthentication{
			Mtls: &securityapi.PeerAuthentication_MutualTLS{Mode: mode},
		},
	}
	if len(selectorLabels) > 0 {
		peerAuthentication.Spec.Selector = &typeapi.WorkloadSelector{MatchLabels: selectorLabels}
	}
	return peerAuthentication
}

func peerAuthenticationGrantedPermissions() []collect.Permission {
	return []collect.Permission{{
		APIGroup: "security.istio.io",
		Resource: "peerauthentications",
		Verbs:    []string{"list"},
		Granted:  true,
	}}
}

func rootAndNamespacePeerAuthenticationAvailability(namespace, rootNamespace string) collect.PeerAuthenticationAvailability {
	return collect.PeerAuthenticationAvailability{
		Namespaces: map[string]bool{
			rootNamespace: true,
			namespace:     true,
		},
	}
}
