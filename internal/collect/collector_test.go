package collect

import (
	"context"
	"errors"
	"strings"
	"testing"

	networkingapi "istio.io/api/networking/v1alpha3"
	securityapi "istio.io/api/security/v1beta1"
	istionetworkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	istiosecurityv1 "istio.io/client-go/pkg/apis/security/v1"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"
)

func TestCollectorActionAuditOnlyGetListAndNeverSecrets(t *testing.T) {
	kube := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: "foo"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "foo"}},
		&discoveryv1.EndpointSlice{ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: "foo"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "foo"}},
		&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "api-rs", Namespace: "foo"}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "foo"}},
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "node-agent", Namespace: "foo"}},
	)
	istio := istiofake.NewSimpleClientset(
		&istiosecurityv1beta1.PeerAuthentication{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "foo"},
			Spec: securityapi.PeerAuthentication{
				Mtls: &securityapi.PeerAuthentication_MutualTLS{
					Mode: securityapi.PeerAuthentication_MutualTLS_PERMISSIVE,
				},
			},
		},
		&istiosecurityv1.AuthorizationPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-api", Namespace: "foo"},
			Spec:       securityapi.AuthorizationPolicy{},
		},
		&istionetworkingv1.DestinationRule{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "foo"},
			Spec:       networkingapi.DestinationRule{Host: "api.foo.svc.cluster.local"},
		},
		&istionetworkingv1.Sidecar{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "foo"},
			Spec:       networkingapi.Sidecar{},
		},
	)
	gateway := gatewayfake.NewSimpleClientset(&gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "waypoint", Namespace: "foo"},
		Spec:       gatewayv1.GatewaySpec{GatewayClassName: "istio-waypoint"},
	})

	collector := New(kube, istio, gateway)
	collector.SetMaxConcurrentLists(2)

	snapshot, err := collector.Collect(context.Background(), Scope{AllNamespaces: true})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(snapshot.PermissionSummary) == 0 {
		t.Fatal("expected permission summary entries")
	}

	seenResources := map[string]bool{}
	actions := append([]ktesting.Action{}, kube.Actions()...)
	actions = append(actions, istio.Actions()...)
	actions = append(actions, gateway.Actions()...)
	for _, action := range actions {
		if got := action.GetVerb(); got != "get" && got != "list" {
			t.Fatalf("unexpected action verb %q for %#v", got, action)
		}
		resource := action.GetResource().Resource
		if resource == "secrets" {
			t.Fatalf("collector attempted forbidden secrets access: %#v", action)
		}
		seenResources[resource] = true
	}

	for _, resource := range []string{
		"namespaces",
		"pods",
		"services",
		"endpointslices",
		"deployments",
		"replicasets",
		"statefulsets",
		"daemonsets",
		"peerauthentications",
		"authorizationpolicies",
		"destinationrules",
		"sidecars",
		"gateways",
	} {
		if !seenResources[resource] {
			t.Fatalf("expected a read action for %s; saw %#v", resource, seenResources)
		}
	}
}

func TestCollectorDegradesForbiddenAndNotFound(t *testing.T) {
	kube := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "foo"}},
	)
	kube.PrependReactor("list", "pods", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "", errors.New("denied"))
	})
	kube.PrependReactor("list", "endpointslices", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "discovery.k8s.io", Resource: "endpointslices"}, "", errors.New("denied"))
	})

	istio := istiofake.NewSimpleClientset()
	istio.PrependReactor("list", "peerauthentications", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(
			schema.GroupResource{Group: "security.istio.io", Resource: "peerauthentications"},
			"",
		)
	})

	snapshot, err := newTestCollector(kube, istio).Collect(context.Background(), Scope{AllNamespaces: true})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(snapshot.Deployments) != 1 {
		t.Fatalf("deployments collected after degraded resources = %d, want 1", len(snapshot.Deployments))
	}

	assertPermission(t, snapshot.PermissionSummary, "", "pods", false)
	assertPermission(t, snapshot.PermissionSummary, "discovery.k8s.io", "endpointslices", false)
	assertPermission(t, snapshot.PermissionSummary, "security.istio.io", "peerauthentications", false)
	if snapshot.PodsAvailableFor("foo") {
		t.Fatal("PodsAvailableFor(foo) = true after pod list denial")
	}
	if snapshot.PeerAuthenticationsAvailable() {
		t.Fatal("PeerAuthenticationsAvailable = true after peerauthentications not found")
	}
	if snapshot.EndpointSlicesAvailableFor("foo") {
		t.Fatal("EndpointSlicesAvailableFor(foo) = true after EndpointSlice list denial")
	}
}

func TestCollectorRejectsInvalidScopes(t *testing.T) {
	tests := []struct {
		name    string
		scope   Scope
		wantErr string
	}{
		{
			name:    "empty scope",
			scope:   Scope{},
			wantErr: "collector scope required",
		},
		{
			name:    "blank namespace",
			scope:   Scope{Namespaces: []string{"payments", " "}},
			wantErr: "namespace must not be empty",
		},
		{
			name:    "mixed all namespaces and scoped namespaces",
			scope:   Scope{AllNamespaces: true, Namespaces: []string{"payments"}},
			wantErr: "choose either all namespaces or explicit namespaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestCollector(kubefake.NewSimpleClientset(), istiofake.NewSimpleClientset()).Collect(context.Background(), tt.scope)
			if err == nil {
				t.Fatal("Collect returned nil error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Collect error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestCollectorErrorsWhenScopedNamespaceIsMissing(t *testing.T) {
	kube := kubefake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}})
	istio := istiofake.NewSimpleClientset()

	_, err := newTestCollector(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"paymets"}})
	if err == nil {
		t.Fatal("Collect returned nil error for missing namespace")
	}
	if !strings.Contains(err.Error(), `requested namespace "paymets" not found`) {
		t.Fatalf("Collect error = %v, want missing namespace error", err)
	}
	for _, action := range append(kube.Actions(), istio.Actions()...) {
		if action.GetResource().Resource != "namespaces" {
			t.Fatalf("unexpected action after missing namespace: %#v", action)
		}
	}
}

func TestCollectorScopedScanIncludesRootNamespacePolicies(t *testing.T) {
	rootNamespace := "istio-config"
	kube := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: rootNamespace}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"}},
	)
	istio := istiofake.NewSimpleClientset(
		&istiosecurityv1beta1.PeerAuthentication{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: rootNamespace},
			Spec: securityapi.PeerAuthentication{
				Mtls: &securityapi.PeerAuthentication_MutualTLS{Mode: securityapi.PeerAuthentication_MutualTLS_STRICT},
			},
		},
		&istiosecurityv1.AuthorizationPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "default-deny", Namespace: rootNamespace},
			Spec:       securityapi.AuthorizationPolicy{},
		},
		&istionetworkingv1.DestinationRule{
			ObjectMeta: metav1.ObjectMeta{Name: "global-api", Namespace: rootNamespace},
			Spec:       networkingapi.DestinationRule{Host: "api.payments.svc.cluster.local"},
		},
		&istionetworkingv1.Sidecar{
			ObjectMeta: metav1.ObjectMeta{Name: "global-default", Namespace: rootNamespace},
			Spec:       networkingapi.Sidecar{},
		},
	)

	snapshot, err := newTestCollector(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"payments"}, RootNamespace: rootNamespace})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if snapshot.RootNamespace != rootNamespace {
		t.Fatalf("snapshot root namespace = %q, want %q", snapshot.RootNamespace, rootNamespace)
	}
	if len(snapshot.PeerAuthentications) != 1 || len(snapshot.AuthorizationPolicies) != 1 ||
		len(snapshot.DestinationRules) != 1 || len(snapshot.Sidecars) != 1 {
		t.Fatalf(
			"root policies = peerAuth %d, authz %d, destinationRules %d, sidecars %d; want one each",
			len(snapshot.PeerAuthentications),
			len(snapshot.AuthorizationPolicies),
			len(snapshot.DestinationRules),
			len(snapshot.Sidecars),
		)
	}
	rootResources := map[string]string{
		"peerauthentications":   snapshot.PeerAuthentications[0].Namespace,
		"authorizationpolicies": snapshot.AuthorizationPolicies[0].Namespace,
		"destinationrules":      snapshot.DestinationRules[0].Namespace,
		"sidecars":              snapshot.Sidecars[0].Namespace,
	}
	for resource, namespace := range rootResources {
		if namespace != rootNamespace {
			t.Fatalf("%s namespace = %q, want %q", resource, namespace, rootNamespace)
		}
	}

	listNamespaces := map[string]map[string]bool{}
	for _, action := range istio.Actions() {
		resource := action.GetResource().Resource
		if _, ok := rootResources[resource]; ok {
			if listNamespaces[resource] == nil {
				listNamespaces[resource] = map[string]bool{}
			}
			listNamespaces[resource][action.GetNamespace()] = true
		}
	}
	for resource := range rootResources {
		for _, namespace := range []string{"payments", rootNamespace} {
			if !listNamespaces[resource][namespace] {
				t.Fatalf("missing %s list for namespace %q; saw %#v", resource, namespace, listNamespaces[resource])
			}
		}
	}
}

func TestCollectorScopedScanDoesNotListAllNamespaces(t *testing.T) {
	kube := kubefake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}})
	istio := istiofake.NewSimpleClientset()

	if _, err := newTestCollector(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"payments"}}); err != nil {
		t.Fatalf("collect: %v", err)
	}

	for _, action := range kube.Actions() {
		if action.GetResource().Resource != "namespaces" {
			continue
		}
		listAction, ok := action.(ktesting.ListAction)
		if !ok {
			t.Fatalf("namespace action is not ListAction: %#v", action)
		}
		if got := listAction.GetListRestrictions().Fields.String(); got != "metadata.name=payments" {
			t.Fatalf("namespace list field selector = %q, want metadata.name=payments", got)
		}
	}
}

func TestCollectorMergesPermissionSummaryAcrossNamespaces(t *testing.T) {
	kube := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "orders"}},
	)
	istio := istiofake.NewSimpleClientset()

	snapshot, err := newTestCollector(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"payments", "orders"}})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	counts := map[string]int{}
	for _, permission := range snapshot.PermissionSummary {
		counts[permission.APIGroup+"/"+permission.Resource]++
	}
	for key, count := range counts {
		if count != 1 {
			t.Fatalf("permission %s appears %d times in %#v", key, count, snapshot.PermissionSummary)
		}
	}
}

func TestCollectorTracksPeerAuthenticationAvailabilityPerNamespace(t *testing.T) {
	kube := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "orders"}},
	)
	istio := istiofake.NewSimpleClientset(&istiosecurityv1beta1.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: DefaultRootNamespace},
		Spec: securityapi.PeerAuthentication{
			Mtls: &securityapi.PeerAuthentication_MutualTLS{Mode: securityapi.PeerAuthentication_MutualTLS_STRICT},
		},
	})
	istio.PrependReactor("list", "peerauthentications", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.GetNamespace() != "orders" {
			return false, nil, nil
		}
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "security.istio.io", Resource: "peerauthentications"},
			"",
			errors.New("denied"),
		)
	})

	snapshot, err := newTestCollector(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"payments", "orders"}})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	if !snapshot.PeerAuthenticationsAvailableFor("payments", DefaultRootNamespace) {
		t.Fatal("PeerAuthenticationsAvailableFor(payments) = false, want true")
	}
	if snapshot.PeerAuthenticationsAvailableFor("orders", DefaultRootNamespace) {
		t.Fatal("PeerAuthenticationsAvailableFor(orders) = true after namespace denial, want false")
	}
	if snapshot.PeerAuthenticationsAvailable() {
		t.Fatal("PeerAuthenticationsAvailable = true after partial namespace denial, want false")
	}

	permission := findPermission(t, snapshot.PermissionSummary, "security.istio.io", "peerauthentications")
	if !strings.Contains(permission.Impact, "namespace/orders") {
		t.Fatalf("peerauthentications impact = %q, want denied namespace scope", permission.Impact)
	}
}

func TestCollectorTracksReplicaSetAvailabilityPerNamespace(t *testing.T) {
	kube := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "payments"}},
	)
	kube.PrependReactor("list", "replicasets", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "apps", Resource: "replicasets"},
			"",
			errors.New("denied"),
		)
	})

	snapshot, err := newTestCollector(kube, istiofake.NewSimpleClientset()).Collect(context.Background(), Scope{Namespaces: []string{"payments"}})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(snapshot.Deployments) != 1 {
		t.Fatalf("deployments = %d, want 1 despite ReplicaSet denial", len(snapshot.Deployments))
	}
	if snapshot.ReplicaSetsAvailableFor("payments") {
		t.Fatal("ReplicaSetsAvailableFor(payments) = true after ReplicaSet denial, want false")
	}
}

func TestListPagesRestartsOnceWhenContinueTokenExpires(t *testing.T) {
	var continues []string
	items, err := listPages(context.Background(), func(_ context.Context, opts metav1.ListOptions) ([]string, string, error) {
		continues = append(continues, opts.Continue)
		switch len(continues) {
		case 1:
			return []string{"stale"}, "expired-token", nil
		case 2:
			return nil, "", apierrors.NewResourceExpired("continue token expired")
		case 3:
			return []string{"fresh"}, "fresh-token", nil
		case 4:
			return []string{"done"}, "", nil
		default:
			t.Fatalf("unexpected list call %d", len(continues))
			return nil, "", nil
		}
	})
	if err != nil {
		t.Fatalf("listPages: %v", err)
	}
	if got, want := strings.Join(items, ","), "fresh,done"; got != want {
		t.Fatalf("items = %q, want %q", got, want)
	}
	if got, want := strings.Join(continues, ","), ",expired-token,,fresh-token"; got != want {
		t.Fatalf("continue tokens = %q, want %q", got, want)
	}
}

func TestListPagesReturnsRepeatedExpiredContinueToken(t *testing.T) {
	calls := 0
	_, err := listPages(context.Background(), func(_ context.Context, opts metav1.ListOptions) ([]string, string, error) {
		calls++
		if opts.Continue == "" {
			return []string{"page"}, "expired-token", nil
		}
		return nil, "", apierrors.NewResourceExpired("continue token expired")
	})
	if err == nil {
		t.Fatal("listPages returned nil error after repeated expired continue token")
	}
	if !apierrors.IsResourceExpired(err) {
		t.Fatalf("listPages error = %v, want resource expired", err)
	}
	if calls != 4 {
		t.Fatalf("list calls = %d, want 4", calls)
	}
}

func TestPermissionMetadataDoesNotEmbedControlCatalog(t *testing.T) {
	tests := []struct {
		name string
		meta resourceMeta
	}{
		{name: "namespaces", meta: namespaceMeta},
		{name: "pods", meta: podMeta},
		{name: "services", meta: serviceMeta},
		{name: "endpointslices", meta: endpointSliceMeta},
		{name: "deployments", meta: deploymentMeta},
		{name: "replicasets", meta: replicaSetMeta},
		{name: "statefulsets", meta: statefulSetMeta},
		{name: "daemonsets", meta: daemonSetMeta},
		{name: "peerauthentications", meta: peerAuthenticationMeta},
		{name: "destinationrules", meta: destinationRuleMeta},
		{name: "sidecars", meta: sidecarMeta},
		{name: "authorizationpolicies", meta: authorizationPolicyMeta},
		{name: "gateways", meta: gatewayMeta},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permission := tt.meta.permissionForScope(false, "cluster")
			if len(permission.AffectedControls) != 0 {
				t.Fatalf("affected controls = %#v, want scan-time derivation", permission.AffectedControls)
			}
		})
	}
}

func newTestCollector(kube *kubefake.Clientset, istio *istiofake.Clientset) *Collector {
	return New(kube, istio, gatewayfake.NewSimpleClientset())
}

func assertPermission(t *testing.T, permissions []Permission, apiGroup, resource string, granted bool) {
	t.Helper()

	for _, permission := range permissions {
		if permission.APIGroup == apiGroup && permission.Resource == resource && permission.Granted == granted {
			return
		}
	}
	t.Fatalf("missing permission entry apiGroup=%q resource=%q granted=%t in %#v", apiGroup, resource, granted, permissions)
}

func findPermission(t *testing.T, permissions []Permission, apiGroup, resource string) Permission {
	t.Helper()

	for _, permission := range permissions {
		if permission.APIGroup == apiGroup && permission.Resource == resource {
			return permission
		}
	}
	t.Fatalf("missing permission entry apiGroup=%q resource=%q in %#v", apiGroup, resource, permissions)
	return Permission{}
}
