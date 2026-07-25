package context

import (
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

func TestClassificationSourcePrecedenceAndConfidence(t *testing.T) {
	config := ScanConfig{Classification: ClassificationConfig{Sources: []ClassificationSource{
		{Type: "namespace-mapping", Mappings: map[string]string{"mapped": "production"}},
		{Type: "namespace-label", Keys: []string{"platform.example.com/environment"}},
		{Type: "namespace-name", Rules: []NameRule{{Pattern: `-stage$`, Environment: "staging"}}},
		{Type: "cluster-context", Mappings: map[string]string{"kind-prod": "production"}},
	}}}
	tests := []struct {
		name      string
		namespace NamespaceInput
		cluster   string
		infer     bool
		wantEnv   string
		wantConf  string
		wantKnown bool
	}{
		{name: "explicit mapping wins", namespace: NamespaceInput{Name: "mapped", Labels: map[string]string{"platform.example.com/environment": "development"}, LabelsKnown: true}, cluster: "kind-prod", wantEnv: "production", wantConf: ConfidenceUserSupplied, wantKnown: true},
		{name: "configured label is observed", namespace: NamespaceInput{Name: "payments", Labels: map[string]string{"platform.example.com/environment": "development"}, LabelsKnown: true}, cluster: "kind-prod", wantEnv: "development", wantConf: ConfidenceObserved, wantKnown: true},
		{name: "name convention precedes cluster fallback", namespace: NamespaceInput{Name: "payments-stage", LabelsKnown: true}, cluster: "kind-prod", wantEnv: "staging", wantConf: ConfidenceResolved, wantKnown: true},
		{name: "cluster context supplies fallback", namespace: NamespaceInput{Name: "payments", LabelsKnown: true}, cluster: "kind-prod", wantEnv: "production", wantConf: ConfidenceResolved, wantKnown: true},
		{name: "missing namespace permission stops before fallback", namespace: NamespaceInput{Name: "payments", LabelsKnown: false}, cluster: "kind-prod", wantConf: ConfidenceUnavailable},
		{name: "complete unmatched evidence is unclassified", namespace: NamespaceInput{Name: "payments", LabelsKnown: true}, cluster: "other", wantEnv: EnvironmentUnclassified, wantConf: ConfidenceResolved, wantKnown: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyNamespace(tt.namespace, tt.cluster, config.Classification, tt.infer)
			if got.Environment != tt.wantEnv || got.Confidence != tt.wantConf || got.Known != tt.wantKnown {
				t.Fatalf("classification = %#v, want env/confidence/known %q/%q/%v", got, tt.wantEnv, tt.wantConf, tt.wantKnown)
			}
		})
	}
}

func TestInferEnvironmentsIsOptIn(t *testing.T) {
	namespace := NamespaceInput{Name: "payments-prod", LabelsKnown: true}
	withoutInference := classifyNamespace(namespace, "", ClassificationConfig{}, false)
	withInference := classifyNamespace(namespace, "", ClassificationConfig{}, true)
	if withoutInference.Environment != EnvironmentUnclassified || withoutInference.Confidence != ConfidenceResolved {
		t.Fatalf("inference-off classification = %#v", withoutInference)
	}
	if withInference.Environment != "production" || withInference.Confidence != ConfidenceInferred {
		t.Fatalf("inference-on classification = %#v", withInference)
	}
}

func TestClusterEnvironmentClassifiesEveryNamespace(t *testing.T) {
	classification := ClassificationConfig{Sources: []ClassificationSource{{
		Type:        "cluster",
		Environment: "production",
	}}}
	for _, name := range []string{"payments", "checkout-stage"} {
		got := classifyNamespace(NamespaceInput{Name: name, LabelsKnown: true}, "", classification, false)
		if got.Environment != "production" ||
			got.Confidence != ConfidenceUserSupplied ||
			!got.Known {
			t.Fatalf("%s classification = %#v, want cluster-supplied production", name, got)
		}
	}
}

func TestOwnershipUsesConfiguredIdentityThenConfigAndImport(t *testing.T) {
	config := ScanConfig{Ownership: OwnershipConfig{
		Labels: OwnershipLabels{
			AppID: []string{"platform.example.com/application-id"},
			Owner: []string{"platform.example.com/team"},
		},
		Applications: []Application{{AppID: "checkout", Owner: "config-team"}},
	}}
	ownershipImport := OwnershipImport{Applications: []Application{
		{AppID: "payments", Owner: "payments-team"},
		{AppID: "checkout", Owner: "import-team"},
	}}
	input := ResolveInput{
		Config:          config,
		OwnershipImport: ownershipImport,
		Namespaces: []NamespaceInput{
			{Name: "payments-prod", LabelsKnown: true, Labels: map[string]string{"platform.example.com/application-id": "payments"}},
			{Name: "checkout-stage", LabelsKnown: true, Labels: map[string]string{"platform.example.com/application-id": "checkout"}},
		},
		Workloads: []WorkloadInput{
			{Ref: workloadRef("payments-prod", "api"), Labels: map[string]string{}, Namespace: NamespaceInput{Name: "payments-prod", LabelsKnown: true, Labels: map[string]string{"platform.example.com/application-id": "payments"}}},
			{Ref: workloadRef("checkout-stage", "api"), Labels: map[string]string{}, Namespace: NamespaceInput{Name: "checkout-stage", LabelsKnown: true, Labels: map[string]string{"platform.example.com/application-id": "checkout"}}},
			{Ref: workloadRef("payments-prod", "worker"), Labels: map[string]string{"platform.example.com/team": "workload-team"}, Namespace: NamespaceInput{Name: "payments-prod", LabelsKnown: true, Labels: map[string]string{"platform.example.com/application-id": "payments"}}},
			{Ref: workloadRef("payments-prod", "annotated"), Annotations: map[string]string{"platform.example.com/application-id": "payments", "platform.example.com/team": "annotation-team"}, Namespace: NamespaceInput{Name: "payments-prod", LabelsKnown: true}},
			{Ref: workloadRef("payments-prod", "namespace-annotated"), Namespace: NamespaceInput{Name: "payments-prod", LabelsKnown: true, Annotations: map[string]string{"platform.example.com/application-id": "payments", "platform.example.com/team": "namespace-team"}}},
		},
	}
	got := Resolve(input)
	byKey := map[string]Ownership{}
	for _, workload := range got.Workloads {
		byKey[workload.Ref.Namespace+"/"+workload.Ref.Name] = workload.Ownership
	}
	if byKey["payments-prod/api"].AppID != "payments" || byKey["payments-prod/api"].Owner != "payments-team" || byKey["payments-prod/api"].OwnerSource != "ownership-import" {
		t.Fatalf("payments ownership = %#v", byKey["payments-prod/api"])
	}
	if byKey["checkout-stage/api"].Owner != "config-team" || byKey["checkout-stage/api"].OwnerSource != "scan-config" {
		t.Fatalf("checkout ownership = %#v", byKey["checkout-stage/api"])
	}
	if byKey["payments-prod/worker"].Owner != "workload-team" {
		t.Fatalf("workload owner did not win: %#v", byKey["payments-prod/worker"])
	}
	if byKey["payments-prod/annotated"].AppID != "payments" ||
		byKey["payments-prod/annotated"].Owner != "annotation-team" ||
		byKey["payments-prod/annotated"].AppIDSource != "workload annotation platform.example.com/application-id" {
		t.Fatalf("workload annotations did not resolve ownership: %#v", byKey["payments-prod/annotated"])
	}
	if byKey["payments-prod/namespace-annotated"].AppID != "payments" ||
		byKey["payments-prod/namespace-annotated"].Owner != "namespace-team" ||
		byKey["payments-prod/namespace-annotated"].OwnerSource != "namespace annotation platform.example.com/team" {
		t.Fatalf("namespace annotations did not resolve ownership: %#v", byKey["payments-prod/namespace-annotated"])
	}
}

func TestOwnershipUnknownDoesNotFallThroughUnavailableNamespaceLabels(t *testing.T) {
	tests := []struct {
		name         string
		labels       map[string]string
		wantAppID    string
		wantOwner    string
		wantAppKnown bool
		wantOwnKnown bool
	}{
		{
			name:         "known app ID does not bypass unknown owner metadata",
			labels:       map[string]string{"platform.example.com/application-id": "payments"},
			wantAppID:    "payments",
			wantAppKnown: true,
		},
		{
			name:         "known owner survives unknown app ID metadata",
			labels:       map[string]string{"platform.example.com/team": "payments-team"},
			wantOwner:    "payments-team",
			wantOwnKnown: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workload := WorkloadInput{
				Ref:       workloadRef("payments", "api"),
				Labels:    tt.labels,
				Namespace: NamespaceInput{Name: "payments", LabelsKnown: false},
			}
			got := resolveOwnership(
				workload,
				[]string{"platform.example.com/application-id"},
				[]string{"platform.example.com/team"},
				nil,
				map[string]string{"payments": "import-team"},
			)
			if got.AppID != tt.wantAppID ||
				got.Owner != tt.wantOwner ||
				got.AppIDKnown != tt.wantAppKnown ||
				got.OwnerKnown != tt.wantOwnKnown {
				t.Fatalf(
					"ownership = %#v, want app/owner/known %q/%q/%v/%v",
					got,
					tt.wantAppID,
					tt.wantOwner,
					tt.wantAppKnown,
					tt.wantOwnKnown,
				)
			}
			if (!got.AppIDKnown && got.AppIDReason == "") ||
				(!got.OwnerKnown && got.OwnerReason == "") {
				t.Fatalf("ownership = %#v, want reasons for unavailable fields", got)
			}
		})
	}
}

func TestResolveCapturesExactExceptionAnnotation(t *testing.T) {
	got := Resolve(ResolveInput{
		Namespaces: []NamespaceInput{{Name: "payments", LabelsKnown: true}},
		Workloads: []WorkloadInput{{
			Ref:         workloadRef("payments", "api"),
			Annotations: map[string]string{"openmeshguard.io/exception": "EXC-42"},
			Namespace:   NamespaceInput{Name: "payments", LabelsKnown: true},
		}},
	})
	if len(got.Workloads) != 1 || got.Workloads[0].ExceptionID != "EXC-42" {
		t.Fatalf("resolved context = %#v", got)
	}
}

func workloadRef(namespace, name string) resolver.WorkloadRef {
	return resolver.WorkloadRef{Namespace: namespace, Name: name, Kind: "Deployment"}
}
