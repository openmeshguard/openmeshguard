package rbac_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestPublishedProfilesMatchSPECSection13(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want map[string][]string
	}{
		{
			name: "cluster profile",
			path: "cluster-role.yaml",
			want: map[string][]string{
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
			name: "namespace profile",
			path: "namespace-role.yaml",
			want: map[string][]string{
				"":                          {"pods", "services"},
				"apps":                      {"daemonsets", "deployments", "replicasets", "statefulsets"},
				"discovery.k8s.io":          {"endpointslices"},
				"gateway.networking.k8s.io": {"backendtlspolicies", "gateways", "grpcroutes", "httproutes", "referencegrants", "tcproutes", "tlsroutes", "udproutes"},
				"networking.istio.io":       {"destinationrules", "envoyfilters", "gateways", "proxyconfigs", "serviceentries", "sidecars", "virtualservices", "workloadentries", "workloadgroups"},
				"security.istio.io":         {"authorizationpolicies", "peerauthentications", "requestauthentications"},
				"telemetry.istio.io":        {"telemetries"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := readProfile(t, tt.path)
			got := resourcesByAPIGroup(t, profile.Rules)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("resources by API group = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEveryPublishedRuleIsReadOnlyAndExplained(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("**", "*.yaml"))
	if err != nil {
		t.Fatalf("glob RBAC manifests: %v", err)
	}
	paths = append(paths, "cluster-role.yaml", "namespace-role.yaml")
	sort.Strings(paths)

	seen := map[string]struct{}{}
	for _, path := range paths {
		if _, duplicate := seen[path]; duplicate {
			continue
		}
		seen[path] = struct{}{}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		profile := decodeProfile(t, path, raw)
		whyComments := bytes.Count(raw, []byte("# Why:"))
		if whyComments != len(profile.Rules) {
			t.Errorf("%s has %d rules but %d per-rule why comments", path, len(profile.Rules), whyComments)
		}
		for _, rule := range profile.Rules {
			if !reflect.DeepEqual(rule.Verbs, []string{"get", "list"}) {
				t.Errorf("%s rule %v verbs = %v, want exactly get/list", path, rule.Resources, rule.Verbs)
			}
			for _, resource := range rule.Resources {
				if resource == "secrets" || strings.HasPrefix(resource, "secrets/") {
					t.Errorf("%s grants forbidden Secrets access", path)
				}
			}
		}
	}
}

type profile struct {
	Rules []rbacv1.PolicyRule `json:"rules"`
}

func readProfile(t *testing.T, path string) profile {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return decodeProfile(t, path, raw)
}

func decodeProfile(t *testing.T, path string, raw []byte) profile {
	t.Helper()
	var decoded profile
	if err := yaml.NewYAMLToJSONDecoder(bytes.NewReader(raw)).Decode(&decoded); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return decoded
}

func resourcesByAPIGroup(t *testing.T, rules []rbacv1.PolicyRule) map[string][]string {
	t.Helper()
	got := map[string][]string{}
	for _, rule := range rules {
		if len(rule.APIGroups) != 1 {
			t.Fatalf("rule %v API groups = %v, want exactly one", rule.Resources, rule.APIGroups)
		}
		got[rule.APIGroups[0]] = append(got[rule.APIGroups[0]], rule.Resources...)
	}
	for group := range got {
		sort.Strings(got[group])
	}
	return got
}
