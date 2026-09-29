package output

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/normalize"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestRuntimeOutputMatchesSchemaAndSeparatesDeclared(t *testing.T) {
	packs, err := engine.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile("../../docs/contracts/canonical-json-schema.json")
	if err != nil {
		t.Fatal(err)
	}
	posture := workloadPosture(resolver.ModeSidecar, resolver.MTLSStrict, nil)
	for _, status := range []string{"corroborated", "contradicted", "no-traffic-observed", "unknown"} {
		t.Run(status, func(t *testing.T) {
			verified := map[string]any{"status": status, "window": "24h0m0s"}
			if status == "corroborated" {
				verified["plaintextObserved"] = false
				verified["mtlsTrafficShare"] = 1.0
			}
			if status == "contradicted" {
				verified["plaintextObserved"] = true
				verified["mtlsTrafficShare"] = 0.25
			}
			evaluated, err := engine.Evaluate(packs, engine.Input{Workloads: []engine.WorkloadInput{{Posture: posture, Verified: verified}}, NamespaceTargetsComplete: true})
			if err != nil {
				t.Fatal(err)
			}
			input := ScanInput{GeneratedAt: time.Unix(1770000000, 0), ScannerVersion: "dev", ResolverVersion: "fixture", ClusterContext: "fixture", Scope: ScanScope{AllNamespaces: true}, Inventory: normalize.Inventory{Counts: map[string]int{}, DataPlaneMode: resolver.ModeSidecar}, WorkloadPostures: []resolver.WorkloadResult{posture}}
			var buffer bytes.Buffer
			if err := WriteScanJSONWithRuntime(&buffer, input, packs, evaluated, RuntimeInput{Enabled: true, URL: "https://prom.example", Lookback: "168h0m0s", DegradedTo: "24h0m0s", Verified: map[string]map[string]any{"payments/api": verified}}); err != nil {
				t.Fatal(err)
			}
			decoded, err := jsonschema.UnmarshalJSON(bytes.NewReader(buffer.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(decoded); err != nil {
				t.Fatal(err)
			}
			var report report
			if err := json.Unmarshal(buffer.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(report.WorkloadPostures[0].MTLS, posture.MTLS) || !reflect.DeepEqual(report.WorkloadPostures[0].Authorization, posture.Authz) {
				t.Fatal("runtime evidence changed declared fields")
			}
			if report.WorkloadPostures[0].Verified.Status != status {
				t.Fatal("lost canonical runtime state")
			}
			var html bytes.Buffer
			if err := WriteHTML(&html, bytes.NewReader(buffer.Bytes())); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(html.String(), "24h0m0s") {
				t.Fatal("degraded runtime window missing in HTML")
			}
		})
	}
}

func TestTelemetrylessDeclaredProjectionMatchesPreM7(t *testing.T) {
	packs, err := engine.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	prior := make([]engine.Pack, len(packs))
	copy(prior, packs)
	for i := range prior {
		prior[i].Controls = nil
		for _, control := range packs[i].Controls {
			if control.EvidenceType != "runtime" {
				prior[i].Controls = append(prior[i].Controls, control)
			}
		}
		if prior[i].Metadata.Name == "builtin-mtls" {
			prior[i].Metadata.Version = "0.3.0"
		}
	}
	posture := workloadPosture(resolver.ModeSidecar, resolver.MTLSDisabled, nil)
	input := ScanInput{GeneratedAt: time.Unix(1770000000, 0), ScannerVersion: "v0.1.0", ResolverVersion: "fixture", ClusterContext: "fixture", Scope: ScanScope{AllNamespaces: true}, Inventory: normalize.Inventory{Counts: map[string]int{}, DataPlaneMode: resolver.ModeSidecar}, WorkloadPostures: []resolver.WorkloadResult{posture}}
	engineInput := defaultEngineInput(input)
	oldEvaluation, err := engine.Evaluate(prior, engineInput)
	if err != nil {
		t.Fatal(err)
	}
	newEvaluation, err := engine.Evaluate(packs, engineInput)
	if err != nil {
		t.Fatal(err)
	}
	old := buildReport(input, prior, oldEvaluation)
	current := buildReport(input, packs, newEvaluation)
	declared := []finding{}
	unknowns := 0
	for _, finding := range current.Findings {
		if finding.EvidenceType != "runtime" {
			declared = append(declared, finding)
		} else {
			unknowns++
			if finding.Status != "unknown" || finding.UnknownReason == "" {
				t.Fatalf("unexpected runtime addition: %+v", finding)
			}
		}
	}
	if unknowns != 2 {
		t.Fatalf("runtime unknown additions=%d", unknowns)
	}
	if !reflect.DeepEqual(old.Findings, declared) || !reflect.DeepEqual(old.WorkloadPostures, current.WorkloadPostures) || !reflect.DeepEqual(old.Scan, current.Scan) || !reflect.DeepEqual(old.Inventory, current.Inventory) || !reflect.DeepEqual(old.PermissionSummary, current.PermissionSummary) {
		t.Fatal("pre-M7 declared report projection changed")
	}
	var html bytes.Buffer
	var canonical bytes.Buffer
	if err := WriteScanJSONWithRuntime(&canonical, input, packs, newEvaluation, RuntimeInput{}); err != nil {
		t.Fatal(err)
	}
	if err := WriteHTML(&html, bytes.NewReader(canonical.Bytes())); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html.String(), "no telemetry access") {
		t.Fatal("missing no telemetry rendering")
	}
}

func TestRuntimeOutputDoesNotAttachTelemetryToUnmeshedNameCollision(t *testing.T) {
	packs, err := engine.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	sidecar := workloadPosture(resolver.ModeSidecar, resolver.MTLSStrict, nil)
	outside := workloadPosture(resolver.ModeNotApplicable, resolver.MTLSNotInMesh, nil)
	outside.Ref.Kind = "StatefulSet"
	input := ScanInput{GeneratedAt: time.Unix(1770000000, 0), Inventory: normalize.Inventory{Counts: map[string]int{}, DataPlaneMode: resolver.ModeMixed}, WorkloadPostures: []resolver.WorkloadResult{sidecar, outside}}
	var encoded bytes.Buffer
	if err := WriteScanJSONWithRuntime(&encoded, input, packs, engine.Result{}, RuntimeInput{Enabled: true, Verified: map[string]map[string]any{"payments/api": {"status": "corroborated", "window": "1h", "plaintextObserved": false, "mtlsTrafficShare": 1.0}}}); err != nil {
		t.Fatal(err)
	}
	var report report
	if err := json.Unmarshal(encoded.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.WorkloadPostures[0].Verified == nil || report.WorkloadPostures[1].Verified != nil {
		t.Fatalf("runtime evidence attached across mesh enrollment: %+v", report.WorkloadPostures)
	}
}
