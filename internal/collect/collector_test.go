package collect

import (
	"context"
	"errors"
	"testing"

	securityapi "istio.io/api/security/v1beta1"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	istiofake "istio.io/client-go/pkg/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestCollectorActionAuditOnlyGetListAndNeverSecrets(t *testing.T) {
	kube := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-1", Namespace: "foo"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "foo"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "foo"}},
		&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "api-rs", Namespace: "foo"}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "foo"}},
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "node-agent", Namespace: "foo"}},
	)
	istio := istiofake.NewSimpleClientset(&istiosecurityv1beta1.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "foo"},
		Spec: securityapi.PeerAuthentication{
			Mtls: &securityapi.PeerAuthentication_MutualTLS{
				Mode: securityapi.PeerAuthentication_MutualTLS_PERMISSIVE,
			},
		},
	})

	collector := New(kube, istio)
	collector.SetMaxConcurrentLists(2)

	snapshot, err := collector.Collect(context.Background(), Scope{AllNamespaces: true})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(snapshot.PermissionSummary) == 0 {
		t.Fatal("expected permission summary entries")
	}

	seenResources := map[string]bool{}
	for _, action := range append(kube.Actions(), istio.Actions()...) {
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
		"deployments",
		"replicasets",
		"statefulsets",
		"daemonsets",
		"peerauthentications",
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

	istio := istiofake.NewSimpleClientset()
	istio.PrependReactor("list", "peerauthentications", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(
			schema.GroupResource{Group: "security.istio.io", Resource: "peerauthentications"},
			"",
		)
	})

	snapshot, err := New(kube, istio).Collect(context.Background(), Scope{AllNamespaces: true})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(snapshot.Deployments) != 1 {
		t.Fatalf("deployments collected after degraded resources = %d, want 1", len(snapshot.Deployments))
	}

	assertPermission(t, snapshot.PermissionSummary, "", "pods", false)
	assertPermission(t, snapshot.PermissionSummary, "security.istio.io", "peerauthentications", false)
	if snapshot.PeerAuthenticationsAvailable() {
		t.Fatal("PeerAuthenticationsAvailable = true after peerauthentications not found")
	}
}

func TestCollectorScopedScanIncludesRootNamespacePeerAuthentications(t *testing.T) {
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
	)

	snapshot, err := New(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"payments"}, RootNamespace: rootNamespace})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if snapshot.RootNamespace != rootNamespace {
		t.Fatalf("snapshot root namespace = %q, want %q", snapshot.RootNamespace, rootNamespace)
	}
	if len(snapshot.PeerAuthentications) != 1 {
		t.Fatalf("peer authentications = %d, want root namespace policy", len(snapshot.PeerAuthentications))
	}
	if got := snapshot.PeerAuthentications[0].Namespace; got != rootNamespace {
		t.Fatalf("peer authentication namespace = %q, want %q", got, rootNamespace)
	}

	paNamespaces := map[string]bool{}
	for _, action := range istio.Actions() {
		if action.GetResource().Resource == "peerauthentications" {
			paNamespaces[action.GetNamespace()] = true
		}
	}
	for _, namespace := range []string{"payments", rootNamespace} {
		if !paNamespaces[namespace] {
			t.Fatalf("missing peerauthentication list for namespace %q; saw %#v", namespace, paNamespaces)
		}
	}
}

func TestCollectorScopedScanDoesNotListAllNamespaces(t *testing.T) {
	kube := kubefake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "payments"}})
	istio := istiofake.NewSimpleClientset()

	if _, err := New(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"payments"}}); err != nil {
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

	snapshot, err := New(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"payments", "orders"}})
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

	snapshot, err := New(kube, istio).Collect(context.Background(), Scope{Namespaces: []string{"payments", "orders"}})
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
