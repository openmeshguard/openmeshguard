package rbac_test

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type expectedProfile struct {
	path      string
	kind      string
	name      string
	resources map[string][]string
}

var expectedProfiles = []expectedProfile{
	{
		path: "cluster-role.yaml",
		kind: "ClusterRole",
		name: "openmeshguard-cluster-scan",
		resources: map[string][]string{
			"":                          {"namespaces", "pods", "services"},
			"apps":                      {"daemonsets", "deployments", "replicasets", "statefulsets"},
			"discovery.k8s.io":          {"endpointslices"},
			"gateway.networking.k8s.io": {"backendtlspolicies", "gatewayclasses", "gateways", "grpcroutes", "httproutes", "referencegrants", "tcproutes", "tlsroutes", "udproutes"},
			"networking.istio.io":       {"destinationrules", "envoyfilters", "gateways", "proxyconfigs", "serviceentries", "sidecars", "virtualservices", "workloadentries", "workloadgroups"},
			"security.istio.io":         {"authorizationpolicies", "peerauthentications", "requestauthentications"},
			"telemetry.istio.io":        {"telemetries"},
		},
	},
	{
		path: "namespace-role.yaml",
		kind: "Role",
		name: "openmeshguard-namespace-scan",
		resources: map[string][]string{
			"":                          {"pods", "services"},
			"apps":                      {"daemonsets", "deployments", "replicasets", "statefulsets"},
			"discovery.k8s.io":          {"endpointslices"},
			"gateway.networking.k8s.io": {"backendtlspolicies", "gateways", "grpcroutes", "httproutes", "referencegrants", "tcproutes", "tlsroutes", "udproutes"},
			"networking.istio.io":       {"destinationrules", "envoyfilters", "gateways", "proxyconfigs", "serviceentries", "sidecars", "virtualservices", "workloadentries", "workloadgroups"},
			"security.istio.io":         {"authorizationpolicies", "peerauthentications", "requestauthentications"},
			"telemetry.istio.io":        {"telemetries"},
		},
	},
	{
		path:      filepath.Join("addons", "control-plane-configmaps-role.yaml"),
		kind:      "Role",
		name:      "openmeshguard-evidence-control-plane-configmaps",
		resources: map[string][]string{"": {"configmaps"}},
	},
	{
		path:      filepath.Join("addons", "events-role.yaml"),
		kind:      "Role",
		name:      "openmeshguard-evidence-events",
		resources: map[string][]string{"": {"events"}},
	},
	{
		path:      filepath.Join("addons", "nodes-cluster-role.yaml"),
		kind:      "ClusterRole",
		name:      "openmeshguard-evidence-nodes",
		resources: map[string][]string{"": {"nodes"}},
	},
}

func TestPublishedProfilesMatchSPECSection13Exactly(t *testing.T) {
	t.Parallel()

	for _, expected := range expectedProfiles {
		expected := expected
		t.Run(expected.path, func(t *testing.T) {
			t.Parallel()

			raw, decoded := readProfile(t, expected.path)
			if decoded.APIVersion != "rbac.authorization.k8s.io/v1" {
				t.Fatalf("apiVersion = %q, want rbac.authorization.k8s.io/v1", decoded.APIVersion)
			}
			if decoded.Kind != expected.kind {
				t.Fatalf("kind = %q, want %q", decoded.Kind, expected.kind)
			}
			if decoded.Metadata.Name != expected.name {
				t.Fatalf("metadata.name = %q, want %q", decoded.Metadata.Name, expected.name)
			}
			if got := resourcesByAPIGroup(t, decoded.Rules); !reflect.DeepEqual(got, expected.resources) {
				t.Fatalf("resources by API group = %#v, want %#v", got, expected.resources)
			}
			if whyComments := bytes.Count(raw, []byte("# Why:")); whyComments != len(decoded.Rules) {
				t.Fatalf("%s has %d rules but %d per-rule why comments", expected.path, len(decoded.Rules), whyComments)
			}
			if decoded.AggregationRule != nil {
				t.Fatalf("%s has forbidden aggregationRule %#v", expected.path, decoded.AggregationRule)
			}
			for _, rule := range decoded.Rules {
				if !reflect.DeepEqual(rule.Verbs, []string{"get", "list"}) {
					t.Fatalf("rule %v verbs = %v, want exactly get/list", rule.Resources, rule.Verbs)
				}
				if len(rule.ResourceNames) != 0 {
					t.Fatalf("rule %v has unexpected resourceNames %v", rule.Resources, rule.ResourceNames)
				}
				if len(rule.NonResourceURLs) != 0 {
					t.Fatalf("rule %v has unexpected nonResourceURLs %v", rule.Resources, rule.NonResourceURLs)
				}
			}
		})
	}
}

func TestNoUnverifiedRBACManifestExists(t *testing.T) {
	t.Parallel()

	var got []string
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			got = append(got, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk RBAC manifests: %v", err)
	}
	sort.Strings(got)

	want := make([]string, 0, len(expectedProfiles))
	for _, expected := range expectedProfiles {
		want = append(want, expected.path)
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("published RBAC manifests = %v, want exactly %v", got, want)
	}
}

type profile struct {
	APIVersion string
	Kind       string
	Metadata   struct {
		Name string
	}
	AggregationRule *rbacv1.AggregationRule
	Rules           []rbacv1.PolicyRule
}

func readProfile(t *testing.T, path string) ([]byte, profile) {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewReader(raw))
	var decoded profile
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			t.Fatalf("decode additional YAML document in %s: %v", path, err)
		}
		t.Fatalf("%s contains more than one YAML document", path)
	}
	return raw, decoded
}

func resourcesByAPIGroup(t *testing.T, rules []rbacv1.PolicyRule) map[string][]string {
	t.Helper()

	got := map[string][]string{}
	for _, rule := range rules {
		if len(rule.APIGroups) != 1 {
			t.Fatalf("rule %v API groups = %v, want exactly one", rule.Resources, rule.APIGroups)
		}
		group := rule.APIGroups[0]
		got[group] = append(got[group], rule.Resources...)
	}
	for group := range got {
		sort.Strings(got[group])
	}
	return got
}
