package normalize

import (
	"strings"
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
			name: "detects ambient enrollment from namespace label",
			snapshot: collect.Snapshot{
				Namespaces: []corev1.Namespace{namespace("ambient", map[string]string{"istio.io/dataplane-mode": "ambient"})},
				Pods: []corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "ambient", Labels: map[string]string{"app": "api"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}}},
				}},
				PermissionSummary: peerAuthenticationGrantedPermissions(),
			},
			assert: func(t *testing.T, result Result) {
				workload := singleWorkload(t, result)
				if workload.DataPlaneMode != resolver.ModeAmbient {
					t.Fatalf("ambient mode = %q, want ambient", workload.DataPlaneMode)
				}
				if workload.Namespace.AmbientEnrolled != resolver.True {
					t.Fatalf("namespace ambient enrollment = %v, want true", workload.Namespace.AmbientEnrolled)
				}
				if workload.ZtunnelOnNode != resolver.Unobserved {
					t.Fatalf("ztunnel on node = %v, want unobserved without ztunnel evidence", workload.ZtunnelOnNode)
				}
			},
		},
	})
}

func TestBuildAmbientEnrollmentAndZtunnelCoverage(t *testing.T) {
	ready := corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}
	base := collect.Snapshot{
		Namespaces: []corev1.Namespace{namespace("ambient", map[string]string{"istio.io/dataplane-mode": "ambient"})},
		Nodes:      []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "worker"}}},
		NodesKnown: true,
		Deployments: []appsv1.Deployment{
			deployment("ambient", "api", map[string]string{"app": "api"}, corev1.PodSpec{}),
		},
		ReplicaSets: []appsv1.ReplicaSet{deploymentReplicaSet("ambient", "api-rs", "api")},
		Pods: []corev1.Pod{
			podForReplicaSet("ambient", "api-1", "api-rs", map[string]string{"app": "api"}, corev1.PodSpec{
				NodeName: "worker", Containers: []corev1.Container{{Name: "api"}},
			}),
		},
		ZtunnelDaemonSets:      []appsv1.DaemonSet{{ObjectMeta: metav1.ObjectMeta{Name: "ztunnel", Namespace: "istio-system"}}},
		ZtunnelDaemonSetsKnown: true,
		ZtunnelPods: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{Name: "ztunnel-1", Namespace: "istio-system"},
			Spec:       corev1.PodSpec{NodeName: "worker"},
			Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{ready}},
		}},
		ZtunnelPodsKnown:  true,
		PermissionSummary: peerAuthenticationGrantedPermissions(),
	}

	tests := []struct {
		name             string
		mutate           func(*collect.Snapshot)
		wantMode         resolver.DataPlaneMode
		wantZtunnel      resolver.Tristate
		wantCovered      int
		wantTotalKnown   bool
		wantAmbientState resolver.Tristate
	}{
		{
			name:             "ready ztunnel covers ambient workload node",
			wantMode:         resolver.ModeAmbient,
			wantZtunnel:      resolver.True,
			wantCovered:      1,
			wantTotalKnown:   true,
			wantAmbientState: resolver.True,
		},
		{
			name: "pod none overrides ambient namespace",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Pods[0].Labels["istio.io/dataplane-mode"] = "none"
			},
			wantMode:         resolver.ModeNotApplicable,
			wantZtunnel:      resolver.Unobserved,
			wantCovered:      1,
			wantTotalKnown:   true,
			wantAmbientState: resolver.True,
		},
		{
			name: "pod ambient overrides unenrolled namespace",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Namespaces[0].Labels = nil
				snapshot.Pods[0].Labels["istio.io/dataplane-mode"] = "ambient"
			},
			wantMode:         resolver.ModeAmbient,
			wantZtunnel:      resolver.True,
			wantCovered:      1,
			wantTotalKnown:   true,
			wantAmbientState: resolver.False,
		},
		{
			name: "missing ready ztunnel is conclusive uncovered",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.ZtunnelPods[0].Status.Conditions = nil
			},
			wantMode:         resolver.ModeAmbient,
			wantZtunnel:      resolver.False,
			wantCovered:      0,
			wantTotalKnown:   true,
			wantAmbientState: resolver.True,
		},
		{
			name: "node permission absence preserves null total",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Nodes = nil
				snapshot.NodesKnown = false
			},
			wantMode:         resolver.ModeAmbient,
			wantZtunnel:      resolver.True,
			wantCovered:      1,
			wantTotalKnown:   false,
			wantAmbientState: resolver.True,
		},
		{
			name: "ztunnel pod evidence absence is unknown",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.ZtunnelPods = nil
				snapshot.ZtunnelPodsKnown = false
			},
			wantMode:         resolver.ModeAmbient,
			wantZtunnel:      resolver.Unobserved,
			wantCovered:      -1,
			wantTotalKnown:   true,
			wantAmbientState: resolver.True,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := base
			snapshot.Namespaces = append([]corev1.Namespace(nil), base.Namespaces...)
			snapshot.Namespaces[0].Labels = copyStringMap(base.Namespaces[0].Labels)
			snapshot.Pods = append([]corev1.Pod(nil), base.Pods...)
			snapshot.Pods[0].Labels = copyStringMap(base.Pods[0].Labels)
			snapshot.ZtunnelPods = append([]corev1.Pod(nil), base.ZtunnelPods...)
			snapshot.ZtunnelPods[0].Status.Conditions = append(
				[]corev1.PodCondition(nil),
				base.ZtunnelPods[0].Status.Conditions...,
			)
			if tt.mutate != nil {
				tt.mutate(&snapshot)
			}

			result := Build(snapshot)
			workload := singleWorkload(t, result)
			if workload.DataPlaneMode != tt.wantMode {
				t.Fatalf("data plane mode = %q, want %q", workload.DataPlaneMode, tt.wantMode)
			}
			if workload.ZtunnelOnNode != tt.wantZtunnel {
				t.Fatalf("ztunnel on node = %v, want %v", workload.ZtunnelOnNode, tt.wantZtunnel)
			}
			if workload.Namespace.AmbientEnrolled != tt.wantAmbientState {
				t.Fatalf("ambient enrollment = %v, want %v", workload.Namespace.AmbientEnrolled, tt.wantAmbientState)
			}
			if got := result.Inventory.Ztunnel.NodesCovered; tt.wantCovered < 0 {
				if got != nil {
					t.Fatalf("nodes covered = %v, want unavailable", *got)
				}
			} else if got == nil || *got != tt.wantCovered {
				t.Fatalf("nodes covered = %v, want %d", got, tt.wantCovered)
			}
			if got := result.Inventory.Ztunnel.NodesTotal; (got != nil) != tt.wantTotalKnown {
				t.Fatalf("nodes total = %v, want known=%t", got, tt.wantTotalKnown)
			}
		})
	}
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
				Deployments: []appsv1.Deployment{deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{
					Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}},
				})},
				PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{
					peerAuthentication("istio-system", "api-override", map[string]string{"app": "api"}, securityapi.PeerAuthentication_MutualTLS_DISABLE),
				},
				PeerAuthAvailability: rootAndNamespacePeerAuthenticationAvailability("payments", "istio-system"),
			},
			assert: func(t *testing.T, result Result) {
				workload := singleWorkload(t, result)
				peerAuthentications := workload.PeerAuthN
				if len(peerAuthentications) != 1 {
					t.Fatalf("peer authentications = %#v, want root selector policy", peerAuthentications)
				}
				if !peerAuthentications[0].SelectorMatch {
					t.Fatal("root namespace selector PeerAuthentication was included without SelectorMatch")
				}
				if peerAuthentications[0].Namespace != "istio-system" {
					t.Fatalf("peer authentication namespace = %q, want istio-system", peerAuthentications[0].Namespace)
				}
				mtls := resolver.New().ResolveMTLS(workload)
				if mtls.Effective != resolver.MTLSUnknown {
					t.Fatalf("effective mTLS = %q, want unknown for root namespace selector ambiguity", mtls.Effective)
				}
				if !strings.Contains(mtls.UnknownReason, "root-namespace selector PeerAuthentication") {
					t.Fatalf("unknown reason = %q, want root namespace selector reason", mtls.UnknownReason)
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
			name: "splits controller when PeerAuthentication selectors differ across pods",
			snapshot: collect.Snapshot{
				Namespaces:  []corev1.Namespace{namespace("payments", nil)},
				Deployments: []appsv1.Deployment{deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{})},
				ReplicaSets: []appsv1.ReplicaSet{
					deploymentReplicaSet("payments", "api-old", "api"),
					deploymentReplicaSet("payments", "api-new", "api"),
				},
				Pods: []corev1.Pod{
					podForReplicaSet("payments", "api-old-1", "api-old", map[string]string{
						"app":               "api",
						"pod-template-hash": "old",
					}, corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}}}),
					podForReplicaSet("payments", "api-new-1", "api-new", map[string]string{
						"app":               "api",
						"pod-template-hash": "new",
					}, corev1.PodSpec{Containers: []corev1.Container{{Name: "api"}, {Name: "istio-proxy"}}}),
				},
				PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{
					peerAuthentication("payments", "old-pods", map[string]string{"pod-template-hash": "old"}, securityapi.PeerAuthentication_MutualTLS_STRICT),
				},
				PeerAuthAvailability: rootAndNamespacePeerAuthenticationAvailability("payments", "istio-system"),
			},
			assert: func(t *testing.T, result Result) {
				workloads := workloadsByKindName(result)
				if len(workloads) != 2 {
					t.Fatalf("workloads = %#v, want two pod-level workloads", result.Workloads)
				}
				if _, ok := workloads["Deployment/api"]; ok {
					t.Fatalf("workloads = %#v, do not want aggregate Deployment/api", result.Workloads)
				}

				oldPod := workloads["Pod/api-old-1"]
				if mtls := resolver.New().ResolveMTLS(oldPod); mtls.Effective != resolver.MTLSStrict {
					t.Fatalf("old pod effective mTLS = %q, want strict", mtls.Effective)
				}
				newPod := workloads["Pod/api-new-1"]
				if mtls := resolver.New().ResolveMTLS(newPod); mtls.Effective != resolver.MTLSPermissive {
					t.Fatalf("new pod effective mTLS = %q, want permissive", mtls.Effective)
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
