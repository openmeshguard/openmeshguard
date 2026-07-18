package normalize

import (
	"reflect"
	"strings"
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	networkingapi "istio.io/api/networking/v1alpha3"
	securityapi "istio.io/api/security/v1beta1"
	typeapi "istio.io/api/type/v1beta1"
	istionetworkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	istiosecurityv1 "istio.io/client-go/pkg/apis/security/v1"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestBuildPolicyInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*collect.Snapshot)
		assert func(*testing.T, resolver.WorkloadInput)
	}{
		{
			name: "produces named service target ports and DestinationRule client TLS",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{
					destinationRule("payments", "api-client-tls", "api", nil, nil, networkingapi.ClientTLSSettings_DISABLE),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if !reflect.DeepEqual(workload.Ports, []int32{8080}) {
					t.Fatalf("ports = %#v, want observed service-bound port 8080", workload.Ports)
				}
				if !workload.DestinationRulesKnown || len(workload.DestRules) != 1 {
					t.Fatalf("DestinationRule input = known %t, rules %#v", workload.DestinationRulesKnown, workload.DestRules)
				}
				if workload.DestRules[0].Host != "api" || workload.DestRules[0].TLSMode != "DISABLE" {
					t.Fatalf("DestinationRule view = %#v", workload.DestRules[0])
				}

				result := resolver.New().ResolveMTLS(workload)
				if result.ClientTLSContradiction == nil || !*result.ClientTLSContradiction {
					t.Fatalf("client TLS contradiction = %#v, want true", result.ClientTLSContradiction)
				}
				if !reflect.DeepEqual(result.ByPort, map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict}) {
					t.Fatalf("byPort = %#v, want port 8080 strict", result.ByPort)
				}
			},
		},
		{
			name: "preserves a port-level DestinationRule override without TLS settings",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				rule := destinationRule("payments", "api-client-tls", "api", nil, nil, networkingapi.ClientTLSSettings_DISABLE)
				rule.Spec.TrafficPolicy.PortLevelSettings = []*networkingapi.TrafficPolicy_PortTrafficPolicy{{
					Port: &networkingapi.PortSelector{Number: 8080},
				}}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{rule}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				wantPortModes := map[int32]string{8080: ""}
				if len(workload.DestRules) != 1 || !reflect.DeepEqual(workload.DestRules[0].PortTLSModes, wantPortModes) {
					t.Fatalf("DestinationRule port TLS modes = %#v, want %#v", workload.DestRules, wantPortModes)
				}

				result := resolver.New().ResolveMTLS(workload)
				if result.ClientTLSContradiction == nil || *result.ClientTLSContradiction {
					t.Fatalf("client TLS contradiction = %#v, want false", result.ClientTLSContradiction)
				}
			},
		},
		{
			name: "preserves an observed empty port set",
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Ports == nil || len(workload.Ports) != 0 {
					t.Fatalf("ports = %#v, want non-nil observed empty set", workload.Ports)
				}
			},
		},
		{
			name: "marks service port evidence unavailable after collection denial",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.ServiceAvailability.Namespaces["payments"] = false
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Ports != nil {
					t.Fatalf("ports = %#v, want nil unavailable evidence", workload.Ports)
				}
			},
		},
		{
			name: "applies exportTo and selected Sidecar egress scoping",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Services = []corev1.Service{
					serviceForWorkload("payments", "api", "api", "http"),
					serviceForWorkload("payments", "blocked", "api", "http"),
				}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{
					destinationRule("payments", "visible", "api", []string{"."}, nil, networkingapi.ClientTLSSettings_ISTIO_MUTUAL),
					destinationRule("istio-system", "not-exported", "api.payments.svc.cluster.local", []string{"."}, nil, networkingapi.ClientTLSSettings_DISABLE),
					destinationRule("payments", "sidecar-blocked", "blocked", nil, nil, networkingapi.ClientTLSSettings_DISABLE),
				}
				snapshot.Sidecars = []*istionetworkingv1.Sidecar{{
					ObjectMeta: metav1.ObjectMeta{Name: "api-egress", Namespace: "payments"},
					Spec: networkingapi.Sidecar{
						WorkloadSelector: &networkingapi.WorkloadSelector{Labels: map[string]string{"app": "api"}},
						Egress:           []*networkingapi.IstioEgressListener{{Hosts: []string{"./api"}}},
					},
				}}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if len(workload.DestRules) != 1 || workload.DestRules[0].Name != "visible" {
					t.Fatalf("DestinationRules = %#v, want only payments/visible", workload.DestRules)
				}
			},
		},
		{
			name: "applies the root namespace Sidecar default",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{
					destinationRule("payments", "api-client-tls", "api", nil, nil, networkingapi.ClientTLSSettings_ISTIO_MUTUAL),
				}
				snapshot.Sidecars = []*istionetworkingv1.Sidecar{{
					ObjectMeta: metav1.ObjectMeta{Name: "global-default", Namespace: "istio-system"},
					Spec: networkingapi.Sidecar{
						Egress: []*networkingapi.IstioEgressListener{{Hosts: []string{"payments/other"}}},
					},
				}}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if !workload.DestinationRulesKnown || workload.DestRules == nil || len(workload.DestRules) != 0 {
					t.Fatalf("DestinationRules = known %t, %#v; want known empty after root Sidecar scoping", workload.DestinationRulesKnown, workload.DestRules)
				}
			},
		},
		{
			name: "prefers a namespace Sidecar default over the root default",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{
					destinationRule("payments", "api-client-tls", "api", nil, nil, networkingapi.ClientTLSSettings_ISTIO_MUTUAL),
				}
				snapshot.Sidecars = []*istionetworkingv1.Sidecar{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "global-default", Namespace: "istio-system"},
						Spec: networkingapi.Sidecar{
							Egress: []*networkingapi.IstioEgressListener{{Hosts: []string{"payments/other"}}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "namespace-default", Namespace: "payments"},
						Spec: networkingapi.Sidecar{
							Egress: []*networkingapi.IstioEgressListener{{Hosts: []string{"./api"}}},
						},
					},
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if len(workload.DestRules) != 1 || workload.DestRules[0].Name != "api-client-tls" {
					t.Fatalf("DestinationRules = %#v, want namespace Sidecar to expose api-client-tls", workload.DestRules)
				}
			},
		},
		{
			name: "projects root and namespace AuthorizationPolicies including selector exclusions",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.AuthorizationPolicies = []*istiosecurityv1.AuthorizationPolicy{
					authorizationPolicy("istio-system", "default-deny", nil, nil, nil),
					authorizationPolicy("payments", "allow-all", nil, []*securityapi.Rule{{}}, nil),
					authorizationPolicy("payments", "wrong-workload", map[string]string{"app": "other"}, []*securityapi.Rule{{}}, nil),
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "dry-run",
							Namespace:   "payments",
							Annotations: map[string]string{"istio.io/dry-run": "true"},
						},
						Spec: securityapi.AuthorizationPolicy{Rules: []*securityapi.Rule{{}}},
					},
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if len(workload.AuthzPolicies) != 3 {
					t.Fatalf("AuthorizationPolicies = %#v, want root, local, and selector exclusion", workload.AuthzPolicies)
				}
				byName := map[string]resolver.AuthorizationPolicyView{}
				for _, policy := range workload.AuthzPolicies {
					byName[policy.Name] = policy
				}
				if !byName["default-deny"].RootNamespace || byName["default-deny"].HasRules {
					t.Fatalf("empty root ALLOW projection = %#v", byName["default-deny"])
				}
				if !byName["allow-all"].HasRules || !byName["allow-all"].BroadAllow {
					t.Fatalf("rules: [\x7b\x7d] ALLOW projection = %#v", byName["allow-all"])
				}
				if !byName["wrong-workload"].HasSelector || byName["wrong-workload"].SelectorMatch {
					t.Fatalf("selector mismatch projection = %#v", byName["wrong-workload"])
				}

				result := resolver.New().ResolveAuthz(workload)
				if result.Effective != resolver.AuthzDefaultDenyExplicitAllow {
					t.Fatalf("effective authorization = %q, want default-deny-explicit-allow", result.Effective)
				}
				if result.BroadAllow == nil || !*result.BroadAllow {
					t.Fatalf("broadAllow = %#v, want true", result.BroadAllow)
				}
				if got := result.Chain[len(result.Chain)-1]; got.Field != "spec.selector" || !strings.Contains(got.Effect, "excludes") {
					t.Fatalf("last chain step = %#v, want explicit selector exclusion", got)
				}
			},
		},
		{
			name: "discovers a ready service waypoint for targetRef L7 policy",
			mutate: func(snapshot *collect.Snapshot) {
				service := serviceForWorkload("payments", "api", "api", "http")
				service.Labels = map[string]string{useWaypointLabel: "api-waypoint"}
				snapshot.Services = []corev1.Service{service}
				snapshot.AuthorizationPolicies = []*istiosecurityv1.AuthorizationPolicy{
					authorizationPolicy("payments", "get-api", nil, l7GetRule(), []*typeapi.PolicyTargetReference{{Kind: "Service", Name: "api"}}),
				}
				snapshot.Gateways = []gatewayv1.Gateway{readyWaypoint("payments", "api-waypoint", "service")}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Waypoint == nil || !workload.Waypoint.Known || !workload.Waypoint.Ready || workload.Waypoint.Scope != "service" {
					t.Fatalf("waypoint = %#v, want known ready service waypoint", workload.Waypoint)
				}
				if len(workload.AuthzPolicies) != 1 || !workload.AuthzPolicies[0].TargetsWaypoint || !workload.AuthzPolicies[0].RequiresL7 {
					t.Fatalf("AuthorizationPolicy targetRef projection = %#v", workload.AuthzPolicies)
				}
				workload.DataPlaneMode = resolver.ModeAmbient
				result := resolver.New().ResolveAuthz(workload)
				if result.Effective != resolver.AuthzAllowOnly {
					t.Fatalf("effective authorization = %q, want allow-only", result.Effective)
				}
				if got := result.Chain[len(result.Chain)-1]; got.Kind != "Waypoint" || !strings.Contains(got.Effect, "ready and enforces") {
					t.Fatalf("last chain step = %#v, want ready waypoint enforcement", got)
				}
			},
		},
		{
			name: "preserves unavailable cross-namespace waypoint evidence",
			mutate: func(snapshot *collect.Snapshot) {
				service := serviceForWorkload("payments", "api", "api", "http")
				service.Labels = map[string]string{
					useWaypointLabel:          "api-waypoint",
					useWaypointNamespaceLabel: "mesh-infra",
				}
				snapshot.Services = []corev1.Service{service}
				snapshot.AuthorizationPolicies = []*istiosecurityv1.AuthorizationPolicy{
					authorizationPolicy("payments", "get-api", nil, l7GetRule(), []*typeapi.PolicyTargetReference{{Kind: "Service", Name: "api"}}),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Waypoint == nil || workload.Waypoint.Known || workload.Waypoint.Namespace != "mesh-infra" {
					t.Fatalf("waypoint = %#v, want explicit unavailable evidence", workload.Waypoint)
				}
				workload.DataPlaneMode = resolver.ModeAmbient
				result := resolver.New().ResolveAuthz(workload)
				if result.Effective != resolver.AuthzUnknown || !strings.Contains(result.UnknownReason, "waypoint evidence unavailable") {
					t.Fatalf("authorization = %#v, want waypoint evidence unknown", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := policySnapshot()
			if tt.mutate != nil {
				tt.mutate(&snapshot)
			}
			tt.assert(t, singleWorkload(t, Build(snapshot)))
		})
	}
}

func TestBuildSplitsControllerWhenPolicyInputsDifferAcrossPods(t *testing.T) {
	snapshot := policySnapshot()
	snapshot.ReplicaSets = []appsv1.ReplicaSet{
		deploymentReplicaSet("payments", "api-old", "api"),
		deploymentReplicaSet("payments", "api-new", "api"),
	}
	snapshot.Pods = []corev1.Pod{
		podForReplicaSet("payments", "api-old-1", "api-old", map[string]string{"app": "api", "track": "old"}, corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "api", Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}},
				{Name: "istio-proxy"},
			},
		}),
		podForReplicaSet("payments", "api-new-1", "api-new", map[string]string{"app": "api", "track": "new"}, corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "api", Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 9090}}},
				{Name: "istio-proxy"},
			},
		}),
	}
	snapshot.Services = []corev1.Service{{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "api", "track": "old"},
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromString("http"),
			}},
		},
	}}

	workloads := workloadsByKindName(Build(snapshot))
	if len(workloads) != 2 {
		t.Fatalf("workloads = %#v, want two pod-level workloads", workloads)
	}
	if _, ok := workloads["Deployment/api"]; ok {
		t.Fatalf("workloads = %#v, do not want aggregate Deployment/api", workloads)
	}
	if got := workloads["Pod/api-old-1"].Ports; !reflect.DeepEqual(got, []int32{8080}) {
		t.Fatalf("old pod ports = %#v, want service-bound port 8080", got)
	}
	if got := workloads["Pod/api-new-1"].Ports; got == nil || len(got) != 0 {
		t.Fatalf("new pod ports = %#v, want observed empty set", got)
	}
}

func policySnapshot() collect.Snapshot {
	return collect.Snapshot{
		Namespaces: []corev1.Namespace{
			namespace("payments", nil),
			namespace("istio-system", nil),
		},
		Deployments: []appsv1.Deployment{deployment("payments", "api", map[string]string{"app": "api"}, corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "api", Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}},
				{Name: "istio-proxy"},
			},
		})},
		PeerAuthentications: []*istiosecurityv1beta1.PeerAuthentication{
			peerAuthentication("istio-system", "default", nil, securityapi.PeerAuthentication_MutualTLS_STRICT),
		},
		ServiceAvailability:             scopedAvailability("payments"),
		PeerAuthAvailability:            scopedAvailability("payments", "istio-system"),
		DestinationRuleAvailability:     scopedAvailability("payments", "istio-system"),
		SidecarAvailability:             scopedAvailability("payments", "istio-system"),
		AuthorizationPolicyAvailability: scopedAvailability("payments", "istio-system"),
		GatewayAvailability:             scopedAvailability("payments"),
	}
}

func scopedAvailability(namespaces ...string) collect.ScopedAvailability {
	available := collect.ScopedAvailability{Namespaces: map[string]bool{}}
	for _, namespace := range namespaces {
		available.Namespaces[namespace] = true
	}
	return available
}

func serviceForWorkload(namespace, name, app, targetPort string) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": app},
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromString(targetPort),
			}},
		},
	}
}

func destinationRule(
	namespace string,
	name string,
	host string,
	exportTo []string,
	selector map[string]string,
	mode networkingapi.ClientTLSSettings_TLSmode,
) *istionetworkingv1.DestinationRule {
	rule := &istionetworkingv1.DestinationRule{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: networkingapi.DestinationRule{
			Host:          host,
			ExportTo:      exportTo,
			TrafficPolicy: &networkingapi.TrafficPolicy{Tls: &networkingapi.ClientTLSSettings{Mode: mode}},
		},
	}
	if selector != nil {
		rule.Spec.WorkloadSelector = &typeapi.WorkloadSelector{MatchLabels: selector}
	}
	return rule
}

func authorizationPolicy(
	namespace string,
	name string,
	selector map[string]string,
	rules []*securityapi.Rule,
	targetRefs []*typeapi.PolicyTargetReference,
) *istiosecurityv1.AuthorizationPolicy {
	policy := &istiosecurityv1.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: securityapi.AuthorizationPolicy{
			Rules:      rules,
			TargetRefs: targetRefs,
		},
	}
	if selector != nil {
		policy.Spec.Selector = &typeapi.WorkloadSelector{MatchLabels: selector}
	}
	return policy
}

func l7GetRule() []*securityapi.Rule {
	return []*securityapi.Rule{{
		To: []*securityapi.Rule_To{{Operation: &securityapi.Operation{Methods: []string{"GET"}}}},
	}}
}

func readyWaypoint(namespace, name, scope string) gatewayv1.Gateway {
	return gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{waypointForLabel: scope},
		},
		Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName(waypointGatewayClass)},
		Status: gatewayv1.GatewayStatus{Conditions: []metav1.Condition{{
			Type:   string(gatewayv1.GatewayConditionProgrammed),
			Status: metav1.ConditionTrue,
		}}},
	}
}
