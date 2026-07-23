package resolver

import (
	"encoding/json"
	"testing"
)

func TestTristateZeroValueIsUnobserved(t *testing.T) {
	var value Tristate
	if value != Unobserved {
		t.Fatalf("zero-value Tristate = %v, want Unobserved", value)
	}
}

func TestWorkloadRefJSONShape(t *testing.T) {
	data, err := json.Marshal(WorkloadRef{
		Cluster:   "cluster-a",
		Namespace: "payments",
		Name:      "api",
		Kind:      "Deployment",
	})
	if err != nil {
		t.Fatalf("marshal WorkloadRef: %v", err)
	}

	const want = `{"cluster":"cluster-a","namespace":"payments","name":"api","kind":"Deployment"}`
	if string(data) != want {
		t.Fatalf("WorkloadRef JSON = %s, want %s", data, want)
	}
}

func TestAuthzBroadAllowJSONAvailability(t *testing.T) {
	knownFalse := false
	tests := []struct {
		name        string
		result      AuthzResult
		wantPresent bool
		wantValue   bool
	}{
		{
			name:        "known false is explicit",
			result:      AuthzResult{Effective: AuthzNoPolicy, BroadAllow: &knownFalse, Chain: []Step{}},
			wantPresent: true,
		},
		{
			name:        "unavailable is omitted",
			result:      AuthzResult{Effective: AuthzUnknown, Chain: []Step{}, UnknownReason: "AuthorizationPolicy resources unavailable"},
			wantPresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("marshal AuthzResult: %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal AuthzResult: %v", err)
			}
			value, present := decoded["broadAllow"]
			if present != tt.wantPresent {
				t.Fatalf("broadAllow present = %t, want %t in %s", present, tt.wantPresent, data)
			}
			if present && value != tt.wantValue {
				t.Fatalf("broadAllow = %#v, want %t", value, tt.wantValue)
			}
		})
	}
}

func TestAuthzIdentityScopedJSONAvailability(t *testing.T) {
	knownFalse := false
	tests := []struct {
		name        string
		result      AuthzResult
		wantPresent bool
	}{
		{
			name:        "known false is explicit",
			result:      AuthzResult{Effective: AuthzNoPolicy, IdentityScoped: &knownFalse, Chain: []Step{}},
			wantPresent: true,
		},
		{
			name:        "unavailable is omitted",
			result:      AuthzResult{Effective: AuthzUnknown, Chain: []Step{}, UnknownReason: authorizationPoliciesUnavailableReason},
			wantPresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("marshal AuthzResult: %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal AuthzResult: %v", err)
			}
			_, present := decoded["identityScoped"]
			if present != tt.wantPresent {
				t.Fatalf("identityScoped present = %t, want %t in %s", present, tt.wantPresent, data)
			}
		})
	}
}
