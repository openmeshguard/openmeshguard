package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	"github.com/openmeshguard/openmeshguard/internal/telemetry"
)

func TestRuntimeControlsUseObservedEvidenceAndCriticalFloor(t *testing.T) {
	at := time.Unix(1770000000, 0)
	for _, tt := range []struct {
		name, policy, value, declared, status string
		wantCritical, noTraffic, denied       bool
	}{
		{name: "strict plaintext", policy: "none", value: "2", declared: "strict", status: "contradicted", wantCritical: true},
		{name: "permissive plaintext", policy: "none", value: "2", declared: "permissive", status: "corroborated"},
		{name: "strict mtls", policy: "mutual_tls", value: "2", declared: "strict", status: "corroborated"},
		{name: "no traffic", policy: "mutual_tls", value: "0", declared: "strict", status: "no-traffic-observed", noTraffic: true},
		{name: "denied", declared: "strict", status: "unknown", denied: true},
		{name: "disabled with mtls", policy: "mutual_tls", value: "2", declared: "disabled", status: "unknown"},
		{name: "disabled plaintext", policy: "none", value: "2", declared: "disabled", status: "corroborated"},
		{name: "mixed by port", policy: "none", value: "2", declared: "mixed-by-port", status: "unknown"},
		{name: "unknown declared", policy: "mutual_tls", value: "2", declared: "unknown", status: "unknown"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.denied {
					w.WriteHeader(403)
					return
				}
				fmt.Fprintf(w, `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"destination_workload_namespace":"apps","destination_workload":"server","connection_security_policy":%q},"values":[[%d,%q]]}]}}`, tt.policy, at.Unix(), tt.value)
			}))
			defer server.Close()
			client, err := telemetry.New(telemetry.Config{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			workload := engine.WorkloadInput{Posture: resolver.WorkloadResult{Ref: resolver.WorkloadRef{Namespace: "apps", Name: "server", Kind: "Deployment"}, Mode: resolver.ModeSidecar, MTLS: resolver.MTLSResult{Effective: resolver.MTLSEffective(tt.declared), Chain: []resolver.Step{{Order: 1, Kind: "PeerAuthentication", Effect: "declared mode"}}}}}
			workloads := []engine.WorkloadInput{workload}
			runtime, permission := runtimeInputs(context.Background(), client, scanOptions{Prometheus: telemetry.Config{URL: server.URL}}, workloads, at)
			if runtime.Verified["apps/server"]["status"] != tt.status || permission.Granted == tt.denied {
				t.Fatalf("runtime=%+v permission=%+v", runtime, permission)
			}
			packs, err := engine.LoadBuiltins()
			if err != nil {
				t.Fatal(err)
			}
			result, err := engine.Evaluate(packs, engine.Input{Workloads: workloads, NamespaceTargetsComplete: true, ControlOverrides: map[string]engine.ControlOverride{"MG-MTLS-101": {SeverityByEnvironment: map[string]string{"": "info"}}}})
			if err != nil {
				t.Fatal(err)
			}
			runtimeFindings := 0
			for _, finding := range result.Findings {
				if finding.EvidenceType != "runtime" {
					continue
				}
				runtimeFindings++
				if tt.noTraffic || tt.denied {
					if finding.Status != "unknown" {
						t.Fatalf("absence passed or failed: %+v", finding)
					}
					continue
				}
				if finding.ControlID == "MG-MTLS-101" {
					if tt.wantCritical && finding.Severity != "critical" {
						t.Fatalf("override weakened contradiction: %+v", finding)
					}
					if finding.Confidence != "observed" {
						t.Fatal("runtime finding not observed")
					}
				}
			}
			if tt.policy == "mutual_tls" && !tt.noTraffic && runtimeFindings != 0 {
				t.Fatal("mTLS traffic failed runtime controls")
			}
		})
	}
}
func TestNoPrometheusAndUnvalidatedPlanesStayUnknown(t *testing.T) {
	workloads := []engine.WorkloadInput{{Posture: resolver.WorkloadResult{Mode: resolver.ModeSidecar}}}
	runtime, permission := runtimeInputs(context.Background(), nil, scanOptions{}, workloads, time.Now())
	if runtime.Enabled || permission.Granted || workloads[0].Availability["verified"].Reason == "" {
		t.Fatal("missing telemetry silently available")
	}
	if _, err := runtimeClient(scanOptions{Prometheus: telemetry.Config{URL: "http://remote.example"}, PrometheusTokenFile: "private"}); err == nil {
		t.Fatal("credentials accepted over remote plaintext HTTP")
	}
}
