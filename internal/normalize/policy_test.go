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
	discoveryv1 "k8s.io/api/discovery/v1"
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
					Port: &networkingapi.PortSelector{Number: 80},
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
			name: "matches DestinationRule selectors against client proxy labels",
			mutate: func(snapshot *collect.Snapshot) {
				addClientProxy(snapshot, "clients", map[string]string{"app": "caller"})
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{
					destinationRule("clients", "caller-plaintext", "api.payments.svc.cluster.local", nil, map[string]string{"app": "caller"}, networkingapi.ClientTLSSettings_DISABLE),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if len(workload.DestRules) != 1 || workload.DestRules[0].Name != "caller-plaintext" {
					t.Fatalf("DestinationRules = %#v, want caller-selected rule", workload.DestRules)
				}
				result := resolver.New().ResolveMTLS(workload)
				if result.ClientTLSContradiction == nil || !*result.ClientTLSContradiction {
					t.Fatalf("client TLS contradiction = %#v, want true", result.ClientTLSContradiction)
				}
			},
		},
		{
			name: "incomplete client controller discovery makes DestinationRule evidence unavailable",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Deployments = nil
				snapshot.Pods = []corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: "payments", Labels: map[string]string{"app": "api"}},
					Spec: corev1.PodSpec{Containers: []corev1.Container{
						{Name: "api", Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}},
						{Name: "istio-proxy"},
					}},
				}}
				snapshot.Namespaces = append(snapshot.Namespaces, namespaceForPolicyTest("clients"))
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{
					destinationRule("clients", "caller-plaintext", "api.payments.svc.cluster.local", nil, map[string]string{"app": "caller"}, networkingapi.ClientTLSSettings_DISABLE),
				}
				snapshot.DestinationRuleAvailability.Namespaces["clients"] = true
				snapshot.SidecarAvailability.Namespaces["clients"] = true
				snapshot.PermissionSummary = append(snapshot.PermissionSummary, collect.Permission{
					APIGroup: "apps", Resource: "deployments", Verbs: []string{"list"}, Granted: false,
				})
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.DestinationRulesKnown || workload.DestRules != nil {
					t.Fatalf("DestinationRule input = known %t, rules %#v; want unavailable client evidence", workload.DestinationRulesKnown, workload.DestRules)
				}
				result := resolver.New().ResolveMTLS(workload)
				if result.ClientTLSContradiction != nil {
					t.Fatalf("client TLS contradiction = %#v, want unknown", result.ClientTLSContradiction)
				}
			},
		},
		{
			name: "service namespace DestinationRule wins over root namespace rule",
			mutate: func(snapshot *collect.Snapshot) {
				addClientProxy(snapshot, "clients", map[string]string{"app": "caller"})
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{
					destinationRule("payments", "service-mutual", "api", nil, nil, networkingapi.ClientTLSSettings_ISTIO_MUTUAL),
					destinationRule("istio-system", "root-plaintext", "api.payments.svc.cluster.local", nil, nil, networkingapi.ClientTLSSettings_DISABLE),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if len(workload.DestRules) != 1 || workload.DestRules[0].Name != "service-mutual" {
					t.Fatalf("DestinationRules = %#v, want only service namespace winner", workload.DestRules)
				}
				result := resolver.New().ResolveMTLS(workload)
				if result.ClientTLSContradiction == nil || *result.ClientTLSContradiction {
					t.Fatalf("client TLS contradiction = %#v, want false", result.ClientTLSContradiction)
				}
			},
		},
		{
			name: "client namespace DestinationRule wins before service and root namespaces",
			mutate: func(snapshot *collect.Snapshot) {
				addClientProxy(snapshot, "clients", map[string]string{"app": "caller"})
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{
					destinationRule("clients", "client-plaintext", "api.payments.svc.cluster.local", nil, nil, networkingapi.ClientTLSSettings_DISABLE),
					destinationRule("payments", "service-mutual", "api", nil, nil, networkingapi.ClientTLSSettings_ISTIO_MUTUAL),
					destinationRule("istio-system", "root-mutual", "api.payments.svc.cluster.local", nil, nil, networkingapi.ClientTLSSettings_ISTIO_MUTUAL),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				names := map[string]bool{}
				for _, rule := range workload.DestRules {
					names[rule.Name] = true
				}
				if !names["client-plaintext"] || !names["service-mutual"] || names["root-mutual"] {
					t.Fatalf("DestinationRules = %#v, want per-client lookup winners without root union", workload.DestRules)
				}
				result := resolver.New().ResolveMTLS(workload)
				if result.ClientTLSContradiction == nil || !*result.ClientTLSContradiction {
					t.Fatalf("client TLS contradiction = %#v, want true from client namespace winner", result.ClientTLSContradiction)
				}
			},
		},
		{
			name: "translates DestinationRule service port override to workload port",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.Services = []corev1.Service{serviceForWorkload("payments", "api", "api", "http")}
				rule := destinationRule("payments", "api-client-tls", "api", nil, nil, networkingapi.ClientTLSSettings_DISABLE)
				rule.Spec.TrafficPolicy.PortLevelSettings = []*networkingapi.TrafficPolicy_PortTrafficPolicy{{
					Port: &networkingapi.PortSelector{Number: 80},
					Tls:  &networkingapi.ClientTLSSettings{Mode: networkingapi.ClientTLSSettings_ISTIO_MUTUAL},
				}}
				snapshot.DestinationRules = []*istionetworkingv1.DestinationRule{rule}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				want := map[int32]string{8080: "ISTIO_MUTUAL"}
				if len(workload.DestRules) != 1 || !reflect.DeepEqual(workload.DestRules[0].PortTLSModes, want) {
					t.Fatalf("DestinationRule port modes = %#v, want %#v", workload.DestRules, want)
				}
				result := resolver.New().ResolveMTLS(workload)
				if result.ClientTLSContradiction == nil || *result.ClientTLSContradiction {
					t.Fatalf("client TLS contradiction = %#v, want false after service-to-workload port translation", result.ClientTLSContradiction)
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
			name: "maps selectorless Service EndpointSlice to workload ports",
			mutate: func(snapshot *collect.Snapshot) {
				configureSelectorlessServiceCase(snapshot, &corev1.ObjectReference{Kind: "Pod", Namespace: "payments", Name: "api-1"})
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if !reflect.DeepEqual(workload.Ports, []int32{8080}) {
					t.Fatalf("ports = %#v, want selectorless Service target port 8080", workload.Ports)
				}
			},
		},
		{
			name: "selectorless Service with a different observed endpoint is known empty",
			mutate: func(snapshot *collect.Snapshot) {
				configureSelectorlessServiceCase(snapshot, &corev1.ObjectReference{Kind: "Pod", Namespace: "payments", Name: "other-1"})
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Ports == nil || len(workload.Ports) != 0 {
					t.Fatalf("ports = %#v, want observed empty selectorless Service evidence", workload.Ports)
				}
			},
		},
		{
			name: "selectorless Service without EndpointSlice targetRef is unavailable",
			mutate: func(snapshot *collect.Snapshot) {
				configureSelectorlessServiceCase(snapshot, nil)
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Ports != nil || workload.DestinationRulesKnown {
					t.Fatalf("policy evidence = ports %#v, destinationRulesKnown %t; want unavailable selectorless attachment", workload.Ports, workload.DestinationRulesKnown)
				}
			},
		},
		{
			name: "selectorless Service EndpointSlice denial is unavailable rather than empty",
			mutate: func(snapshot *collect.Snapshot) {
				configureSelectorlessServiceCase(snapshot, &corev1.ObjectReference{Kind: "Pod", Namespace: "payments", Name: "api-1"})
				snapshot.EndpointSliceAvailability.Namespaces["payments"] = false
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Ports != nil || workload.DestinationRulesKnown {
					t.Fatalf("policy evidence = ports %#v, destinationRulesKnown %t; want unavailable after EndpointSlice denial", workload.Ports, workload.DestinationRulesKnown)
				}
			},
		},
		{
			name: "selectorless Service Pod denial is unavailable rather than empty",
			mutate: func(snapshot *collect.Snapshot) {
				configureSelectorlessServiceCase(snapshot, &corev1.ObjectReference{Kind: "Pod", Namespace: "payments", Name: "api-1"})
				snapshot.PodAvailability = scopedAvailability("payments")
				snapshot.PodAvailability.Namespaces["payments"] = false
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Ports != nil || workload.DestinationRulesKnown {
					t.Fatalf("policy evidence = ports %#v, destinationRulesKnown %t; want unavailable after Pod denial", workload.Ports, workload.DestinationRulesKnown)
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
			name: "empty selector matches every workload in namespace",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.AuthorizationPolicies = []*istiosecurityv1.AuthorizationPolicy{
					authorizationPolicy("payments", "empty-selector", map[string]string{}, []*securityapi.Rule{{}}, nil),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if len(workload.AuthzPolicies) != 1 || !workload.AuthzPolicies[0].HasSelector || !workload.AuthzPolicies[0].SelectorMatch {
					t.Fatalf("AuthorizationPolicy = %#v, want selector: {} to match", workload.AuthzPolicies)
				}
			},
		},
		{
			name: "projects broad access separately from explicit identity scope",
			mutate: func(snapshot *collect.Snapshot) {
				snapshot.AuthorizationPolicies = []*istiosecurityv1.AuthorizationPolicy{
					authorizationPolicy("payments", "operation-only", nil, l7GetRule(), nil),
					authorizationPolicy("payments", "exact-principal", nil, []*securityapi.Rule{{
						From: []*securityapi.Rule_From{{Source: &securityapi.Source{Principals: []string{"cluster.local/ns/caller/sa/client"}}}},
					}}, nil),
					authorizationPolicy("payments", "wildcard-principal", nil, []*securityapi.Rule{{
						From: []*securityapi.Rule_From{{Source: &securityapi.Source{Principals: []string{"cluster.local/ns/caller/sa/*"}}}},
					}}, nil),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				byName := map[string]resolver.AuthorizationPolicyView{}
				for _, policy := range workload.AuthzPolicies {
					byName[policy.Name] = policy
				}
				if !byName["operation-only"].BroadAllow || byName["operation-only"].IdentityScoped {
					t.Fatalf("operation-only projection = %#v, want broad and identity-unscoped", byName["operation-only"])
				}
				if byName["exact-principal"].BroadAllow || !byName["exact-principal"].IdentityScoped {
					t.Fatalf("exact-principal projection = %#v, want narrow and identity-scoped", byName["exact-principal"])
				}
				if !byName["wildcard-principal"].BroadAllow || byName["wildcard-principal"].IdentityScoped {
					t.Fatalf("wildcard-principal projection = %#v, want broad and identity-unscoped", byName["wildcard-principal"])
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
				if len(workload.AuthzPolicies) != 1 || !workload.AuthzPolicies[0].TargetsWaypoint || !workload.AuthzPolicies[0].RequiresL7 ||
					workload.AuthzPolicies[0].TargetRefKind != "Service" || workload.AuthzPolicies[0].TargetRefName != "api" ||
					workload.AuthzPolicies[0].TargetWaypoint == nil || workload.AuthzPolicies[0].TargetWaypoint.Name != "api-waypoint" {
					t.Fatalf("AuthorizationPolicy targetRef projection = %#v", workload.AuthzPolicies)
				}
				sidecarResult := resolver.New().ResolveAuthz(workload)
				if sidecarResult.Effective != resolver.AuthzNoPolicy || len(sidecarResult.PoliciesInScope) != 0 {
					t.Fatalf("sidecar authorization = %#v, want targetRef policy excluded", sidecarResult)
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
			name: "projects singular targetRef with its specific service waypoint",
			mutate: func(snapshot *collect.Snapshot) {
				serviceA := serviceForWorkload("payments", "a-api", "api", "http")
				serviceA.Labels = map[string]string{useWaypointLabel: "a-waypoint"}
				serviceB := serviceForWorkload("payments", "b-api", "api", "http")
				serviceB.Labels = map[string]string{useWaypointLabel: "b-waypoint"}
				snapshot.Services = []corev1.Service{serviceA, serviceB}
				policy := authorizationPolicy("payments", "get-b", nil, l7GetRule(), nil)
				policy.Spec.TargetRef = &typeapi.PolicyTargetReference{Kind: "Service", Name: "b-api"}
				snapshot.AuthorizationPolicies = []*istiosecurityv1.AuthorizationPolicy{policy}
				snapshot.Gateways = []gatewayv1.Gateway{
					readyWaypoint("payments", "a-waypoint", "service"),
					unreadyWaypoint("payments", "b-waypoint", "service"),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if workload.Waypoint == nil || workload.Waypoint.Name != "a-waypoint" {
					t.Fatalf("workload waypoint = %#v, want first selected service waypoint a-waypoint", workload.Waypoint)
				}
				if len(workload.AuthzPolicies) != 1 {
					t.Fatalf("AuthorizationPolicies = %#v, want singular targetRef projection", workload.AuthzPolicies)
				}
				policy := workload.AuthzPolicies[0]
				if policy.TargetRefKind != "Service" || policy.TargetRefName != "b-api" || policy.TargetWaypoint == nil ||
					policy.TargetWaypoint.Name != "b-waypoint" || policy.TargetWaypoint.Ready {
					t.Fatalf("target-specific policy projection = %#v, want unready b-waypoint", policy)
				}
				workload.DataPlaneMode = resolver.ModeAmbient
				result := resolver.New().ResolveAuthz(workload)
				if result.Effective != resolver.AuthzWaypointUnenforced || len(result.WaypointUnenforced) != 1 {
					t.Fatalf("authorization = %#v, want b-waypoint unenforced", result)
				}
				if got := result.Chain[len(result.Chain)-1]; got.Name != "b-waypoint" {
					t.Fatalf("waypoint chain step = %#v, want target-specific b-waypoint", got)
				}
			},
		},
		{
			name: "preserves multiple targetRef attachments in the resolution chain",
			mutate: func(snapshot *collect.Snapshot) {
				serviceA := serviceForWorkload("payments", "a-api", "api", "http")
				serviceA.Labels = map[string]string{useWaypointLabel: "a-waypoint"}
				serviceB := serviceForWorkload("payments", "b-api", "api", "http")
				serviceB.Labels = map[string]string{useWaypointLabel: "b-waypoint"}
				snapshot.Services = []corev1.Service{serviceA, serviceB}
				snapshot.AuthorizationPolicies = []*istiosecurityv1.AuthorizationPolicy{
					authorizationPolicy("payments", "allow-api", nil, l7GetRule(), []*typeapi.PolicyTargetReference{
						{Kind: "Service", Name: "b-api"},
						{Kind: "Service", Name: "a-api"},
					}),
				}
				snapshot.Gateways = []gatewayv1.Gateway{
					readyWaypoint("payments", "a-waypoint", "service"),
					readyWaypoint("payments", "b-waypoint", "service"),
				}
			},
			assert: func(t *testing.T, workload resolver.WorkloadInput) {
				if len(workload.AuthzPolicies) != 2 ||
					workload.AuthzPolicies[0].TargetRefName != "a-api" ||
					workload.AuthzPolicies[1].TargetRefName != "b-api" {
					t.Fatalf("AuthorizationPolicies = %#v, want ordered a-api and b-api attachments", workload.AuthzPolicies)
				}
				workload.DataPlaneMode = resolver.ModeAmbient
				result := resolver.New().ResolveAuthz(workload)
				wantChain := []resolver.Step{
					{Order: 1, Kind: "AuthorizationDefault", Field: "implicitEnablement", Effect: "establishes Istio's implicit allow when no applicable ALLOW policy exists"},
					{Order: 2, Kind: "AuthorizationPolicy", Namespace: "payments", Name: "allow-api", Field: `spec.targetRefs["Service/a-api"]`, Effect: "adds a structurally broad rule to the additive ALLOW union"},
					{Order: 3, Kind: "Waypoint", Namespace: "payments", Name: "a-waypoint", Field: "status.conditions[Programmed]", Effect: "selected service waypoint is ready and enforces waypoint-attached policy payments/allow-api"},
					{Order: 4, Kind: "AuthorizationPolicy", Namespace: "payments", Name: "allow-api", Field: `spec.targetRefs["Service/b-api"]`, Effect: "adds a structurally broad rule to the additive ALLOW union"},
					{Order: 5, Kind: "Waypoint", Namespace: "payments", Name: "b-waypoint", Field: "status.conditions[Programmed]", Effect: "selected service waypoint is ready and enforces waypoint-attached policy payments/allow-api"},
				}
				if !reflect.DeepEqual(result.Chain, wantChain) {
					t.Fatalf("chain = %#v, want attachment-specific %#v", result.Chain, wantChain)
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

func TestBuildCountsEndpointSlices(t *testing.T) {
	snapshot := policySnapshot()
	snapshot.EndpointSlices = []discoveryv1.EndpointSlice{
		{ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: "payments"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "api-2", Namespace: "payments"}},
	}
	if got := Build(snapshot).Inventory.Counts["endpointSlices"]; got != 2 {
		t.Fatalf("endpointSlices count = %d, want 2", got)
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

func addClientProxy(snapshot *collect.Snapshot, namespace string, labels map[string]string) {
	snapshot.Namespaces = append(snapshot.Namespaces, namespaceForPolicyTest(namespace))
	snapshot.ReplicaSets = append(snapshot.ReplicaSets, appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "caller-rs",
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "Deployment", Name: "caller",
			}},
		},
		Spec: appsv1.ReplicaSetSpec{Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "caller"}, {Name: "istio-proxy"}}},
		}},
	})
	snapshot.DestinationRuleAvailability.Namespaces[namespace] = true
	snapshot.SidecarAvailability.Namespaces[namespace] = true
}

func configureSelectorlessServiceCase(snapshot *collect.Snapshot, targetRef *corev1.ObjectReference) {
	snapshot.Deployments = nil
	snapshot.ReplicaSets = nil
	snapshot.Pods = []corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: "payments", Labels: map[string]string{"app": "api"}},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{Name: "api", Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}},
			{Name: "istio-proxy"},
		}},
	}}
	service := serviceForWorkload("payments", "api", "api", "http")
	service.Spec.Selector = nil
	snapshot.Services = []corev1.Service{service}
	snapshot.EndpointSlices = []discoveryv1.EndpointSlice{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-1",
			Namespace: "payments",
			Labels:    map[string]string{discoveryv1.LabelServiceName: "api"},
		},
		Endpoints: []discoveryv1.Endpoint{{TargetRef: targetRef}},
	}}
	snapshot.EndpointSliceAvailability = scopedAvailability("payments")
}

func namespaceForPolicyTest(name string) corev1.Namespace {
	return corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
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

func unreadyWaypoint(namespace, name, scope string) gatewayv1.Gateway {
	gateway := readyWaypoint(namespace, name, scope)
	gateway.Status.Conditions[0].Status = metav1.ConditionFalse
	return gateway
}
