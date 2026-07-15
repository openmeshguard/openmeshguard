package engine

import (
	"reflect"
	"strings"
	"testing"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
)

func TestBuiltinControlsCoverEveryOutcome(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}

	tests := []struct {
		name          string
		controlID     string
		workload      WorkloadInput
		wantFindings  int
		wantStatus    string
		unknownReason string
		wantGrade     string
	}{
		{
			name:      "MG-MTLS-001 pass",
			controlID: "MG-MTLS-001",
			workload:  workloadWithMTLS(resolver.MTLSStrict, nil),
			wantGrade: "A",
		},
		{
			name:         "MG-MTLS-001 fail without exception matching",
			controlID:    "MG-MTLS-001",
			workload:     workloadWithMTLS(resolver.MTLSPermissive, nil),
			wantFindings: 1, wantStatus: statusOpen, wantGrade: "F",
		},
		{
			name:         "MG-MTLS-001 unknown",
			controlID:    "MG-MTLS-001",
			workload:     unknownWorkload("PeerAuthentication resources unavailable"),
			wantFindings: 1, wantStatus: statusUnknown,
			unknownReason: "PeerAuthentication resources unavailable", wantGrade: "unknown",
		},
		{
			name:         "MG-MTLS-001 not applicable",
			controlID:    "MG-MTLS-001",
			workload:     notInMeshWorkload(),
			wantFindings: 1, wantStatus: statusNotApplicable, wantGrade: "unknown",
		},
		{
			name:      "MG-MTLS-002 pass",
			controlID: "MG-MTLS-002",
			workload:  workloadWithMTLS(resolver.MTLSStrict, map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict}),
			wantGrade: "A",
		},
		{
			name:         "MG-MTLS-002 fail",
			controlID:    "MG-MTLS-002",
			workload:     workloadWithMTLS(resolver.MTLSMixedByPort, map[int32]resolver.MTLSEffective{8080: resolver.MTLSStrict, 9090: resolver.MTLSPermissive}),
			wantFindings: 1, wantStatus: statusOpen, wantGrade: "F",
		},
		{
			name:         "MG-MTLS-002 unknown",
			controlID:    "MG-MTLS-002",
			workload:     workloadWithMTLS(resolver.MTLSStrict, nil),
			wantFindings: 1, wantStatus: statusUnknown,
			unknownReason: "workload ports unavailable", wantGrade: "unknown",
		},
		{
			name:         "MG-MTLS-002 not applicable before requires",
			controlID:    "MG-MTLS-002",
			workload:     notInMeshWorkload(),
			wantFindings: 1, wantStatus: statusNotApplicable, wantGrade: "unknown",
		},
		{
			name:      "MG-MTLS-003 pass",
			controlID: "MG-MTLS-003",
			workload:  workloadWithMTLS(resolver.MTLSStrict, nil),
			wantGrade: "A",
		},
		{
			name:         "MG-MTLS-003 fail on confirmed disabled effective posture",
			controlID:    "MG-MTLS-003",
			workload:     workloadWithMTLS(resolver.MTLSDisabled, nil),
			wantFindings: 1, wantStatus: statusOpen, wantGrade: "F",
		},
		{
			name:         "MG-MTLS-003 unknown",
			controlID:    "MG-MTLS-003",
			workload:     unknownWorkload("PeerAuthentication resources unavailable"),
			wantFindings: 1, wantStatus: statusUnknown,
			unknownReason: "PeerAuthentication resources unavailable", wantGrade: "unknown",
		},
		{
			name:         "MG-MTLS-003 not applicable before requires",
			controlID:    "MG-MTLS-003",
			workload:     notInMeshWorkload(),
			wantFindings: 1, wantStatus: statusNotApplicable, wantGrade: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pack := packWithControl(t, packs, tt.controlID)
			result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{tt.workload}})
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			if len(result.Findings) != tt.wantFindings {
				t.Fatalf("findings = %d, want %d: %#v", len(result.Findings), tt.wantFindings, result.Findings)
			}
			if tt.wantFindings > 0 {
				finding := result.Findings[0]
				if finding.Status != tt.wantStatus {
					t.Fatalf("status = %q, want %q", finding.Status, tt.wantStatus)
				}
				if tt.wantStatus == statusOpen && finding.ControlID != tt.controlID {
					t.Fatalf("control ID = %q, want %q", finding.ControlID, tt.controlID)
				}
				if tt.unknownReason != "" && !strings.Contains(finding.UnknownReason, tt.unknownReason) {
					t.Fatalf("unknownReason = %q, want to contain %q", finding.UnknownReason, tt.unknownReason)
				}
			}
			if len(result.Scores) != 1 {
				t.Fatalf("scores = %#v, want one category", result.Scores)
			}
			if result.Scores[0].Grade != tt.wantGrade {
				t.Fatalf("grade = %q, want %q", result.Scores[0].Grade, tt.wantGrade)
			}
		})
	}
}

func TestRealScanMissingProducersBecomeUnknownThroughRequires(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	result, err := Evaluate(packs, Input{Workloads: []WorkloadInput{
		workloadWithMTLS(resolver.MTLSPermissive, nil),
	}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	statuses := map[string]string{}
	reasons := map[string]string{}
	for _, finding := range result.Findings {
		statuses[finding.ControlID] = finding.Status
		reasons[finding.ControlID] = finding.UnknownReason
	}
	if statuses["MG-MTLS-001"] != statusOpen {
		t.Fatalf("MG-MTLS-001 status = %q, want open", statuses["MG-MTLS-001"])
	}
	if statuses["MG-MTLS-002"] != statusUnknown {
		t.Fatalf("MG-MTLS-002 status = %q, want unknown", statuses["MG-MTLS-002"])
	}
	if !strings.Contains(reasons["MG-MTLS-002"], "workload ports unavailable") {
		t.Fatalf("MG-MTLS-002 unknownReason = %q, want missing port producer", reasons["MG-MTLS-002"])
	}
	if _, exists := statuses["MG-MTLS-003"]; exists {
		t.Fatalf("MG-MTLS-003 produced finding %#v, want permissive effective posture to pass disabled-only control", statuses["MG-MTLS-003"])
	}

	contradictionPack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: destination-rule-evidence, version: 1.0.0}
controls:
  - id: ACME-MTLS-004
    title: Client and server TLS must agree
    category: mtls
    severity: high
    evidenceType: config
    scope: workload
    requires: [mtls.clientTLSContradiction]
    applicability: 'true'
    expression: 'workload.mtls.clientTLSContradiction == false'
    message: Client and server TLS contradict.
    remediation: {guidance: Correct DestinationRule TLS.}
`)
	tests := []struct {
		name          string
		contradiction *bool
		wantFindings  int
		wantStatus    string
	}{
		{name: "unavailable is unknown", wantFindings: 1, wantStatus: statusUnknown},
		{name: "collected false passes", contradiction: boolPointer(false)},
		{name: "collected true fails", contradiction: boolPointer(true), wantFindings: 1, wantStatus: statusOpen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workload := workloadWithMTLS(resolver.MTLSPermissive, nil)
			workload.Posture.MTLS.ClientTLSContradiction = tt.contradiction
			customResult, err := Evaluate([]Pack{contradictionPack}, Input{Workloads: []WorkloadInput{workload}})
			if err != nil {
				t.Fatalf("Evaluate DestinationRule-dependent control: %v", err)
			}
			if len(customResult.Findings) != tt.wantFindings {
				t.Fatalf("DestinationRule-dependent findings = %#v, want %d", customResult.Findings, tt.wantFindings)
			}
			if tt.wantFindings > 0 && customResult.Findings[0].Status != tt.wantStatus {
				t.Fatalf("DestinationRule-dependent status = %q, want %q", customResult.Findings[0].Status, tt.wantStatus)
			}
			if tt.wantStatus == statusUnknown && !strings.Contains(customResult.Findings[0].UnknownReason, "DestinationRule collection unavailable") {
				t.Fatalf("unknownReason = %q, want DestinationRule availability", customResult.Findings[0].UnknownReason)
			}
		})
	}
}

func TestEvaluateDeterministicIDsAndCategoryGrades(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	input := Input{Workloads: []WorkloadInput{
		workloadWithMTLS(resolver.MTLSPermissive, nil),
		workloadNamed("worker", resolver.MTLSStrict),
	}}
	first, err := Evaluate(packs, input)
	if err != nil {
		t.Fatalf("first Evaluate: %v", err)
	}
	second, err := Evaluate(packs, input)
	if err != nil {
		t.Fatalf("second Evaluate: %v", err)
	}
	if len(first.Findings) != len(second.Findings) {
		t.Fatalf("finding counts differ: %d and %d", len(first.Findings), len(second.Findings))
	}
	for index := range first.Findings {
		if first.Findings[index].ID != second.Findings[index].ID {
			t.Fatalf("finding ID changed: %q and %q", first.Findings[index].ID, second.Findings[index].ID)
		}
	}
	if len(first.Scores) != 1 || first.Scores[0].PassRate == nil || *first.Scores[0].PassRate != 0.75 || first.Scores[0].Grade != "C" {
		t.Fatalf("score = %#v, want mtls 75%% grade C", first.Scores)
	}
}

func TestEvaluateNamespaceAndResourceScopes(t *testing.T) {
	packYAML := []byte(`
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata:
  name: scoped-controls
  version: 1.0.0
controls:
  - id: ACME-ENV-001
    title: Production namespaces must belong to platform
    category: governance
    severity: medium
    evidenceType: context
    scope: namespace
    environments: [production]
    requires: [namespace.labels.team]
    applicability: 'true'
    expression: 'namespace.labels.team == "platform"'
    message: 'Namespace {{ .Namespace }} is not owned by platform.'
    remediation:
      guidance: Correct the team label.
  - id: ACME-GW-001
    title: Public gateways must not use wildcard hosts
    category: exposure
    severity: high
    evidenceType: config
    scope: resource
    environments: []
    match:
      apiGroups: [networking.istio.io]
      kinds: [Gateway]
    requires: [resource.spec.servers]
    applicability: 'resource.isPubliclyExposed'
    expression: '!resource.spec.servers.exists(s, s.hosts.exists(h, h == "*"))'
    message: 'Gateway {{ .Resource }} exposes a wildcard host.'
    remediation:
      guidance: Replace wildcard hosts with explicit names.
`)
	pack, err := decodeAndValidate("scoped-controls.yaml", packYAML, SourceUser)
	if err != nil {
		t.Fatalf("decode scoped pack: %v", err)
	}
	result, err := Evaluate([]Pack{pack}, Input{
		Namespaces: []NamespaceInput{
			{Name: "platform-prod", Environment: "production", Labels: map[string]string{"team": "platform"}},
			{Name: "payments-prod", Environment: "production", Labels: map[string]string{"team": "payments"}},
			{Name: "unclassified", Labels: map[string]string{"team": "payments"}},
		},
		Resources: []ResourceInput{
			{
				APIVersion: "networking.istio.io/v1", Kind: "Gateway", Namespace: "istio-system", Name: "public",
				Fields: map[string]any{
					"isPubliclyExposed": true,
					"spec": map[string]any{
						"servers": []any{map[string]any{"hosts": []any{"*"}}},
					},
				},
			},
			{
				APIVersion: "networking.istio.io/v1", Kind: "Gateway", Namespace: "istio-system", Name: "private",
				Fields: map[string]any{
					"isPubliclyExposed": false,
					"spec": map[string]any{
						"servers": []any{map[string]any{"hosts": []any{"internal.example"}}},
					},
				},
			},
			{
				APIVersion: "gateway.networking.k8s.io/v1", Kind: "Gateway", Namespace: "istio-system", Name: "same-kind-other-api",
				Fields: map[string]any{
					"isPubliclyExposed": true,
					"spec": map[string]any{
						"listeners": []any{map[string]any{"hostname": "*"}},
					},
				},
			},
			{APIVersion: "networking.istio.io/v1", Kind: "ServiceEntry", Namespace: "payments", Name: "ignored"},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	statuses := map[string][]string{}
	resourceStatusCounts := map[string]int{}
	for _, finding := range result.Findings {
		statuses[finding.ControlID] = append(statuses[finding.ControlID], finding.Status)
		if finding.ControlID == "ACME-GW-001" {
			resourceStatusCounts[finding.Status]++
		}
	}
	if len(statuses["ACME-ENV-001"]) != 1 || statuses["ACME-ENV-001"][0] != statusOpen {
		t.Fatalf("namespace statuses = %#v, want one production failure with unclassified namespace filtered", statuses["ACME-ENV-001"])
	}
	if len(statuses["ACME-GW-001"]) != 2 || resourceStatusCounts[statusNotApplicable] != 1 || resourceStatusCounts[statusOpen] != 1 {
		t.Fatalf("resource statuses = %#v, want not-applicable and open Gateway results", statuses["ACME-GW-001"])
	}
}

func TestEquivalentGatewayControlsStaySourceNative(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: source-native-gateways, version: 1.0.0}
controls:
  - id: ACME-GW-001
    title: Public gateways must not use wildcard hosts
    category: exposure
    severity: high
    evidenceType: config
    scope: resource
    match:
      apiGroups: [gateway.networking.k8s.io]
      kinds: [Gateway]
    requires: [resource.spec.listeners]
    applicability: 'resource.isPubliclyExposed'
    expression: '!resource.spec.listeners.exists(l, has(l.hostname) && l.hostname == "*")'
    message: 'Kubernetes Gateway {{ .Resource }} exposes a wildcard hostname.'
    remediation: {guidance: Replace wildcard listener hostnames.}
    frameworks: [nist-csf-2.0/PR.DS]
  - id: ACME-GW-002
    title: Public gateways must not use wildcard hosts
    category: exposure
    severity: high
    evidenceType: config
    scope: resource
    match:
      apiGroups: [networking.istio.io]
      kinds: [Gateway]
    requires: [resource.spec.servers]
    applicability: 'resource.isPubliclyExposed'
    expression: '!resource.spec.servers.exists(s, s.hosts.exists(h, h == "*"))'
    message: 'Istio Gateway {{ .Resource }} exposes a wildcard hostname.'
    remediation: {guidance: Replace wildcard server hosts.}
    frameworks: [nist-csf-2.0/PR.DS]
`)
	if pack.Controls[0].Title != pack.Controls[1].Title ||
		pack.Controls[0].Category != pack.Controls[1].Category ||
		pack.Controls[0].Severity != pack.Controls[1].Severity ||
		!reflect.DeepEqual(pack.Controls[0].Frameworks, pack.Controls[1].Frameworks) {
		t.Fatalf("equivalent controls drifted: %#v and %#v", pack.Controls[0], pack.Controls[1])
	}

	result, err := Evaluate([]Pack{pack}, Input{Resources: []ResourceInput{
		{
			APIVersion: "gateway.networking.k8s.io/v1", Kind: "Gateway", Namespace: "ingress", Name: "kubernetes-public",
			Fields: map[string]any{
				"isPubliclyExposed": true,
				"spec":              map[string]any{"listeners": []any{map[string]any{"hostname": "*"}}},
			},
		},
		{
			APIVersion: "networking.istio.io/v1", Kind: "Gateway", Namespace: "ingress", Name: "istio-public",
			Fields: map[string]any{
				"isPubliclyExposed": true,
				"spec":              map[string]any{"servers": []any{map[string]any{"hosts": []any{"*"}}}},
			},
		},
		{
			APIVersion: "example.io/v1", Kind: "Gateway", Namespace: "ingress", Name: "unmatched",
			Fields: map[string]any{"isPubliclyExposed": true},
		},
	}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 2 {
		t.Fatalf("findings = %#v, want one per matched source API", result.Findings)
	}
	wantAPIVersion := map[string]string{
		"ACME-GW-001": "gateway.networking.k8s.io/v1",
		"ACME-GW-002": "networking.istio.io/v1",
	}
	for _, finding := range result.Findings {
		if len(finding.Resources) != 1 || finding.Resources[0].APIVersion != wantAPIVersion[finding.ControlID] {
			t.Fatalf("finding %s resources = %#v, want source-native API version %q", finding.ControlID, finding.Resources, wantAPIVersion[finding.ControlID])
		}
		if finding.ControlID == "ACME-GW-001" && containsString(finding.EvidenceSources, "istio-crd") {
			t.Fatalf("Gateway API evidence mislabeled as Istio CRD: %#v", finding.EvidenceSources)
		}
		if finding.ControlID == "ACME-GW-001" && !containsString(finding.EvidenceSources, "gateway-api") {
			t.Fatalf("Gateway API evidence missing gateway-api source: %#v", finding.EvidenceSources)
		}
		if finding.ControlID == "ACME-GW-002" && !containsString(finding.EvidenceSources, "istio-crd") {
			t.Fatalf("Istio Gateway evidence missing Istio CRD source: %#v", finding.EvidenceSources)
		}
	}
}

func TestAPIGroupFromAPIVersion(t *testing.T) {
	tests := []struct {
		apiVersion string
		want       string
	}{
		{apiVersion: "v1", want: ""},
		{apiVersion: "apps/v1", want: "apps"},
		{apiVersion: "gateway.networking.k8s.io/v1", want: "gateway.networking.k8s.io"},
		{apiVersion: " networking.istio.io/v1 ", want: "networking.istio.io"},
	}
	for _, tt := range tests {
		t.Run(tt.apiVersion, func(t *testing.T) {
			if got := apiGroupFromAPIVersion(tt.apiVersion); got != tt.want {
				t.Fatalf("apiGroupFromAPIVersion(%q) = %q, want %q", tt.apiVersion, got, tt.want)
			}
		})
	}
}

func TestWorkloadControlHonorsNamespacePathAvailability(t *testing.T) {
	packYAML := []byte(`
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata:
  name: namespace-availability
  version: 1.0.0
controls:
  - id: ACME-ENV-001
    title: Workload namespace must have a team label
    category: governance
    severity: medium
    evidenceType: context
    scope: workload
    environments: []
    requires: [namespace.labels.team]
    applicability: 'true'
    expression: 'namespace.labels.team == "platform"'
    message: 'Workload {{ .Workload }} has no platform team label.'
    remediation:
      guidance: Add the team label.
`)
	pack, err := decodeAndValidate("namespace-availability.yaml", packYAML, SourceUser)
	if err != nil {
		t.Fatalf("decode pack: %v", err)
	}
	workload := workloadWithMTLS(resolver.MTLSStrict, nil)
	workload.Namespace.Labels = map[string]string{"team": "platform"}
	workload.Namespace.Availability = map[string]Availability{
		"labels.team": {Reason: "namespace labels permission unavailable"},
	}
	result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Status != statusUnknown || !strings.Contains(result.Findings[0].UnknownReason, "namespace labels permission unavailable") {
		t.Fatalf("findings = %#v, want namespace availability unknown", result.Findings)
	}
}

func TestApplicabilityEvidenceIsGatedBeforeCEL(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: applicability-evidence, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Platform namespaces only
    category: governance
    severity: medium
    evidenceType: context
    scope: workload
    requires: [workload.workload.name]
    applicability: 'namespace["labels"]["team"] == "platform"'
    expression: 'true'
    message: 'Workload {{ .Workload }} failed.'
    remediation: {guidance: Restore namespace evidence.}
`)
	workload := workloadWithMTLS(resolver.MTLSStrict, nil)
	workload.Namespace.Labels = map[string]string{"team": "payments"}
	workload.Namespace.Availability = map[string]Availability{
		"labels.team": {Reason: "namespace labels permission unavailable"},
	}
	result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Status != statusUnknown {
		t.Fatalf("findings = %#v, want one unknown instead of not-applicable", result.Findings)
	}
	if !strings.Contains(result.Findings[0].UnknownReason, "namespace labels permission unavailable") {
		t.Fatalf("unknownReason = %q", result.Findings[0].UnknownReason)
	}
}

func TestUnknownDataPlaneModeCannotPassOrFail(t *testing.T) {
	packs, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("LoadBuiltins returned error: %v", err)
	}
	workload := workloadWithMTLS(resolver.MTLSStrict, nil)
	workload.Posture.Mode = resolver.ModeUnknown
	result, err := Evaluate([]Pack{packWithControl(t, packs, "MG-MTLS-001")}, Input{Workloads: []WorkloadInput{workload}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Status != statusUnknown || !strings.Contains(result.Findings[0].UnknownReason, "data plane mode unavailable") {
		t.Fatalf("findings = %#v, want data-plane unknown", result.Findings)
	}
}

func TestBracketDependencyUnavailableBecomesUnknown(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: bracket-dependency, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Team label required
    category: governance
    severity: medium
    evidenceType: context
    scope: workload
    requires: [namespace.labels.team]
    applicability: 'true'
    expression: 'namespace["labels"]["team"] == "platform"'
    message: 'Workload {{ .Workload }} failed.'
    remediation: {guidance: Restore namespace evidence.}
`)
	workload := workloadWithMTLS(resolver.MTLSStrict, nil)
	workload.Namespace.Labels = map[string]string{"team": "platform"}
	workload.Namespace.Availability = map[string]Availability{"labels.team": {Reason: "labels unavailable"}}
	result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Status != statusUnknown || result.Scores[0].Grade != "unknown" {
		t.Fatalf("result = %#v, want unknown and no grade", result)
	}
}

func TestRuntimeMessageUsesFrozenVerifiedTemplateShape(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: verified-message, version: 1.0.0}
controls:
  - id: ACME-MTLS-101
    title: No plaintext observed
    category: mtls
    severity: critical
    evidenceType: runtime
    scope: workload
    requires: [verified.plaintextObserved]
    applicability: 'workload.dataPlaneMode != "not-applicable"'
    expression: 'workload.verified.plaintextObserved == false'
    message: 'Plaintext observed within {{ .Verified.Window }} from {{ .Verified.PlaintextSources }}.'
    remediation: {guidance: Investigate plaintext traffic.}
`)
	workload := workloadWithMTLS(resolver.MTLSStrict, nil)
	workload.Verified = map[string]any{
		"status": "contradicted", "window": "15m", "plaintextObserved": true,
		"plaintextSources": []string{"payments/client"},
	}
	result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 || !strings.Contains(result.Findings[0].Reasoning, "15m") || !strings.Contains(result.Findings[0].Reasoning, "payments/client") {
		t.Fatalf("findings = %#v, want rendered Verified fields", result.Findings)
	}
	if !containsString(result.Findings[0].EvidenceSources, "prometheus") {
		t.Fatalf("open runtime evidence sources = %#v, want prometheus", result.Findings[0].EvidenceSources)
	}

	for _, tt := range []struct {
		name     string
		workload WorkloadInput
		status   string
	}{
		{name: "unknown telemetry", workload: workloadWithMTLS(resolver.MTLSStrict, nil), status: statusUnknown},
		{name: "not applicable before telemetry", workload: notInMeshWorkload(), status: statusNotApplicable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{tt.workload}})
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			if len(result.Findings) != 1 || result.Findings[0].Status != tt.status {
				t.Fatalf("findings = %#v, want %q", result.Findings, tt.status)
			}
			if containsString(result.Findings[0].EvidenceSources, "prometheus") {
				t.Fatalf("unevaluated runtime evidence sources = %#v, want prometheus omitted", result.Findings[0].EvidenceSources)
			}
		})
	}
}

func TestSuggestedYAMLOnlyRendersForOpenFindings(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: remediation-status, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Owner must be platform
    category: governance
    severity: medium
    evidenceType: context
    scope: namespace
    requires: [params.owner, params.applies]
    applicability: 'params.applies == true'
    expression: 'false'
    message: 'Namespace {{ .Namespace }} has the wrong owner.'
    remediation: {guidance: Set the owner.}
`)
	pack.Controls[0].Remediation.SuggestedYAML = "owner: {{ .Params.owner }}"
	tests := []struct {
		name       string
		params     map[string]any
		wantStatus string
		wantYAML   string
	}{
		{name: "unknown does not render missing parameter", wantStatus: statusUnknown},
		{name: "not applicable omits suggested YAML", params: map[string]any{"owner": "platform", "applies": false}, wantStatus: statusNotApplicable},
		{name: "open renders suggested YAML", params: map[string]any{"owner": "platform", "applies": true}, wantStatus: statusOpen, wantYAML: "owner: platform"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Evaluate([]Pack{pack}, Input{Namespaces: []NamespaceInput{{Name: "payments"}}, Params: tt.params})
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			if len(result.Findings) != 1 || result.Findings[0].Status != tt.wantStatus {
				t.Fatalf("findings = %#v, want status %q", result.Findings, tt.wantStatus)
			}
			if result.Findings[0].Remediation.SuggestedYAML != tt.wantYAML {
				t.Fatalf("suggested YAML = %q, want %q", result.Findings[0].Remediation.SuggestedYAML, tt.wantYAML)
			}
		})
	}
}

func TestEvidenceSourcesDescribeEvidenceActuallyUsed(t *testing.T) {
	contextPack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: context-evidence, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Team must be platform
    category: governance
    severity: medium
    evidenceType: context
    scope: workload
    requires: [namespace.labels.team]
    applicability: 'true'
    expression: 'namespace.labels.team == "platform"'
    message: Wrong team.
    remediation: {guidance: Correct the label.}
`)
	contextWorkload := workloadWithMTLS(resolver.MTLSStrict, nil)
	contextWorkload.Namespace.Labels = map[string]string{"team": "payments"}
	contextResult, err := Evaluate([]Pack{contextPack}, Input{Workloads: []WorkloadInput{contextWorkload}})
	if err != nil {
		t.Fatalf("evaluate context control: %v", err)
	}
	if len(contextResult.Findings) != 1 || containsString(contextResult.Findings[0].EvidenceSources, "istio-crd") {
		t.Fatalf("context evidence sources = %#v, want Kubernetes API only", contextResult.Findings)
	}

	builtins, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("load built-ins: %v", err)
	}
	mtlsResult, err := Evaluate([]Pack{packWithControl(t, builtins, "MG-MTLS-001")}, Input{Workloads: []WorkloadInput{workloadWithMTLS(resolver.MTLSPermissive, nil)}})
	if err != nil {
		t.Fatalf("evaluate mTLS control: %v", err)
	}
	if len(mtlsResult.Findings) != 1 || !containsString(mtlsResult.Findings[0].EvidenceSources, "istio-crd") {
		t.Fatalf("mTLS evidence sources = %#v, want Istio CRD", mtlsResult.Findings)
	}
	unknownResult, err := Evaluate([]Pack{packWithControl(t, builtins, "MG-MTLS-001")}, Input{Workloads: []WorkloadInput{unknownWorkload("PeerAuthentication unavailable")}})
	if err != nil {
		t.Fatalf("evaluate unknown mTLS control: %v", err)
	}
	if len(unknownResult.Findings) != 1 || containsString(unknownResult.Findings[0].EvidenceSources, "istio-crd") {
		t.Fatalf("unknown mTLS evidence sources = %#v, want unavailable source omitted", unknownResult.Findings)
	}
}

func TestCanonicalAuthorizationFieldsAndCombinedChain(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: canonical-authz, version: 1.0.0}
controls:
  - id: ACME-AUTHZ-001
    title: Expected authorization posture
    category: authz
    severity: high
    evidenceType: config
    scope: workload
    requires: [mtls.effective, authorization.policiesInScope, authorization.l7Unenforced]
    applicability: 'true'
    expression: 'workload["mtls"]["effective"] == "strict" && workload.authorization.policiesInScope.size() > 1 && workload.authorization.l7Unenforced.size() == 0'
    message: 'Authorization posture for {{ .Workload }} is incomplete.'
    remediation: {guidance: Correct authorization policy.}
`)
	workload := workloadWithMTLS(resolver.MTLSStrict, nil)
	workload.Posture.Authz = resolver.AuthzResult{
		Effective:       resolver.AuthzAllowOnly,
		PoliciesInScope: []string{"payments/default", "payments/api"},
		L7Unenforced:    []string{"payments/api"},
		Chain:           []resolver.Step{{Order: 1, Kind: "AuthorizationPolicy", Name: "api", Namespace: "payments", Effect: "selects workload"}},
	}
	result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Status != statusOpen {
		t.Fatalf("findings = %#v, want one resolved open finding", result.Findings)
	}
	chain := result.Findings[0].ResolutionChain
	if len(chain) != 2 || chain[0].Order != 1 || chain[1].Order != 2 || chain[0].Kind != "PeerAuthentication" || chain[1].Kind != "AuthorizationPolicy" {
		t.Fatalf("resolution chain = %#v, want globally ordered mTLS and authz evidence", chain)
	}
}

func TestParentAvailabilityIncludesUnknownDescendants(t *testing.T) {
	target := evaluationTarget{
		activation: map[string]any{"workload": map[string]any{"mtls": map[string]any{}}},
		availability: map[string]Availability{
			"workload.mtls.effective": {Reason: "effective posture unavailable"},
		},
	}
	control := Control{Scope: "workload"}
	reason := unavailableReasonForPaths(control, target, []string{"workload.mtls"})
	if !strings.Contains(reason, "effective posture unavailable") {
		t.Fatalf("unavailable reason = %q, want unknown child reason", reason)
	}
}

func TestLiteralUnknownOutsidePostureEnumsRemainsKnown(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: literal-unknown, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Team label must be platform
    category: governance
    severity: medium
    evidenceType: context
    scope: workload
    requires: [namespace.labels.team]
    applicability: 'true'
    expression: 'namespace.labels.team == "platform"'
    message: 'Workload {{ .Workload }} does not have the platform team label.'
    remediation: {guidance: Correct the team label.}
`)
	workload := workloadWithMTLS(resolver.MTLSStrict, nil)
	workload.Namespace.Labels = map[string]string{"team": "unknown"}
	result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Status != statusOpen || result.Scores[0].Grade != "F" {
		t.Fatalf("result = %#v, want resolved open finding for literal label value", result)
	}
}

func TestLiteralBracketRequiresPreservesNativeMapKey(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: native-label-key, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Application label must be api
    category: governance
    severity: medium
    evidenceType: context
    scope: workload
    requires: ['namespace.labels["app.kubernetes.io/name"]']
    applicability: 'true'
    expression: 'namespace.labels["app.kubernetes.io/name"] == "api"'
    message: 'Workload {{ .Workload }} has the wrong application label.'
    remediation: {guidance: Correct the application label.}
`)
	tests := []struct {
		name         string
		availability map[string]Availability
		wantStatus   string
	}{
		{name: "known native key passes"},
		{name: "native key availability becomes unknown", availability: map[string]Availability{`labels["app.kubernetes.io/name"]`: {Reason: "label evidence unavailable"}}, wantStatus: statusUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workload := workloadWithMTLS(resolver.MTLSStrict, nil)
			workload.Namespace.Labels = map[string]string{"app.kubernetes.io/name": "api"}
			workload.Namespace.Availability = tt.availability
			result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			if tt.wantStatus == "" {
				if len(result.Findings) != 0 || result.Scores[0].Grade != "A" {
					t.Fatalf("result = %#v, want pass", result)
				}
				return
			}
			if len(result.Findings) != 1 || result.Findings[0].Status != tt.wantStatus {
				t.Fatalf("result = %#v, want status %q", result, tt.wantStatus)
			}
		})
	}
}

func TestInventoryAvailabilityAppliesToEveryScope(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: inventory-availability, version: 1.0.0}
controls:
  - id: ACME-GOV-001
    title: Services must be present
    category: governance
    severity: medium
    evidenceType: context
    scope: namespace
    requires: [inventory.counts.services]
    applicability: 'true'
    expression: 'inventory.counts.services > 0'
    message: 'Service inventory is empty.'
    remediation: {guidance: Restore service inventory evidence.}
`)
	result, err := Evaluate([]Pack{pack}, Input{
		Namespaces: []NamespaceInput{{Name: "payments"}},
		Inventory:  map[string]any{"counts": map[string]any{"services": 0}},
		InventoryAvailability: map[string]Availability{
			"counts.services": {Reason: "services list permission unavailable"},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Status != statusUnknown || !strings.Contains(result.Findings[0].UnknownReason, "services list permission unavailable") {
		t.Fatalf("result = %#v, want permission-derived inventory unknown", result)
	}
}

func TestResolutionChainsAreCELCompatibleMaps(t *testing.T) {
	pack := decodePackForTest(t, `
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata: {name: chain-shape, version: 1.0.0}
controls:
  - id: ACME-MTLS-001
    title: Default resolution steps omit resource-only fields
    category: mtls
    severity: high
    evidenceType: config
    scope: workload
    requires: [mtls.chain]
    applicability: 'true'
    expression: 'workload.mtls.chain.exists(step, step.kind == "MeshConfigDefault" && step.order == 1 && !has(step.name) && !has(step.namespace))'
    message: 'Workload {{ .Workload }} has an invalid default resolution step.'
    remediation: {guidance: Restore policy evidence.}
`)
	workload := workloadWithMTLS(resolver.MTLSStrict, nil)
	workload.Posture.MTLS.Chain = []resolver.Step{{Order: 1, Kind: "MeshConfigDefault", Effect: "uses the mesh default"}}
	result, err := Evaluate([]Pack{pack}, Input{Workloads: []WorkloadInput{workload}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(result.Findings) != 0 || len(result.Scores) != 1 || result.Scores[0].Grade != "A" {
		t.Fatalf("result = %#v, want CEL-compatible chain pass", result)
	}
}

func TestDeterministicFindingIDIgnoresServedAPIVersion(t *testing.T) {
	target := evaluationTarget{cluster: "cluster-a", resource: ResourceRef{
		APIVersion: "gateway.networking.k8s.io/v1beta1", Kind: "Gateway", Namespace: "ingress", Name: "public",
	}}
	first := deterministicFindingID("ACME-GW-001", target)
	target.resource.APIVersion = "gateway.networking.k8s.io/v1"
	if second := deterministicFindingID("ACME-GW-001", target); second != first {
		t.Fatalf("finding ID changed across served versions: %q and %q", first, second)
	}
	target.resource.APIVersion = "networking.istio.io/v1"
	if otherGroup := deterministicFindingID("ACME-GW-001", target); otherGroup == first {
		t.Fatalf("finding ID %q did not distinguish API groups", otherGroup)
	}
}

func decodePackForTest(t *testing.T, data string) Pack {
	t.Helper()
	pack, err := decodeAndValidate("test-pack.yaml", []byte(data), SourceUser)
	if err != nil {
		t.Fatalf("decode test pack: %v", err)
	}
	return pack
}

func packWithControl(t *testing.T, packs []Pack, controlID string) Pack {
	t.Helper()
	for _, pack := range packs {
		for _, control := range pack.Controls {
			if control.ID == controlID {
				pack.Controls = []Control{control}
				return pack
			}
		}
	}
	t.Fatalf("built-in control %s not found", controlID)
	return Pack{}
}

func workloadWithMTLS(effective resolver.MTLSEffective, byPort map[int32]resolver.MTLSEffective) WorkloadInput {
	return WorkloadInput{
		Posture: resolver.WorkloadResult{
			Ref:  resolver.WorkloadRef{Cluster: "cluster-a", Namespace: "payments", Name: "api", Kind: "Deployment"},
			Mode: resolver.ModeSidecar,
			MTLS: resolver.MTLSResult{
				Effective: effective,
				ByPort:    byPort,
				Chain:     []resolver.Step{{Order: 1, Kind: "PeerAuthentication", Namespace: "payments", Name: "default", Effect: "sets effective mTLS"}},
			},
			Authz: resolver.AuthzResult{Effective: resolver.AuthzUnknown, Chain: []resolver.Step{}, UnknownReason: "authorization resolver not yet implemented (M5)"},
		},
		Namespace: NamespaceInput{Name: "payments", Labels: map[string]string{"team": "payments"}},
	}
}

func workloadNamed(name string, effective resolver.MTLSEffective) WorkloadInput {
	workload := workloadWithMTLS(effective, nil)
	workload.Posture.Ref.Name = name
	return workload
}

func unknownWorkload(reason string) WorkloadInput {
	workload := workloadWithMTLS(resolver.MTLSUnknown, nil)
	workload.Posture.MTLS.UnknownReason = reason
	workload.Posture.MTLS.Chain = []resolver.Step{}
	return workload
}

func notInMeshWorkload() WorkloadInput {
	workload := workloadWithMTLS(resolver.MTLSNotInMesh, nil)
	workload.Posture.Mode = resolver.ModeNotApplicable
	workload.Posture.MTLS.Chain = []resolver.Step{{Order: 1, Kind: "DataPlane", Effect: "not enrolled"}}
	return workload
}

func boolPointer(value bool) *bool {
	return &value
}
