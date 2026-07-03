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
