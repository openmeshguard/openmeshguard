package telemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func matrix(at time.Time, policy string, value string) string {
	return fmt.Sprintf(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"destination_workload_namespace":"apps","destination_workload":"server","connection_security_policy":%q},"values":[[%d,%q]]}]}}`, policy, at.Unix(), value)
}
func TestObserveStatesAndExactWindow(t *testing.T) {
	at := time.Unix(1770000000, 0)
	for _, tt := range []struct {
		name, policy, value string
		plain               *bool
		share               *float64
		noTraffic           bool
		unknown             bool
	}{
		{name: "mtls", policy: "mutual_tls", value: "4", plain: boolPointer(false), share: floatPointer(1)},
		{name: "plaintext", policy: "none", value: "2", plain: boolPointer(true), share: floatPointer(0)},
		{name: "zero", policy: "mutual_tls", value: "0", noTraffic: true, unknown: true},
		{name: "unknown policy", policy: "unknown", value: "2", unknown: true},
		{name: "empty", unknown: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				q := r.URL.Query()
				if q.Get("start") != q.Get("end") || q.Get("step") != "3600" || !strings.Contains(q.Get("query"), "[604800s]") || !strings.Contains(q.Get("query"), `reporter="destination"`) {
					t.Errorf("invalid query: %v", q)
				}
				body := matrix(at, tt.policy, tt.value)
				if tt.name == "empty" {
					body = `{"status":"success","data":{"resultType":"matrix","result":[]}}`
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			client, err := New(Config{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			result := client.Observe(context.Background(), []Workload{{"apps", "server"}}, at)
			o := result.Observations[Workload{"apps", "server"}]
			if calls != 2 || o.NoTraffic != tt.noTraffic || (o.UnknownReason != "") != tt.unknown || !sameBool(o.Plaintext, tt.plain) || !sameFloat(o.Share, tt.share) {
				t.Fatalf("result=%+v calls=%d", o, calls)
			}
		})
	}
}
func TestDegradationDiscardsOriginalWindow(t *testing.T) {
	at := time.Unix(1770000000, 0)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		q := r.URL.Query().Get("query")
		if strings.Contains(q, "[604800s]") {
			if strings.Contains(q, "istio_requests_total") {
				fmt.Fprint(w, matrix(at, "none", "8"))
				return
			}
			w.WriteHeader(503)
			fmt.Fprint(w, `{"status":"error","errorType":"timeout","error":"secret"}`)
			return
		}
		fmt.Fprint(w, matrix(at, "mutual_tls", "3"))
	}))
	defer server.Close()
	c, _ := New(Config{URL: server.URL})
	defer c.Close()
	r := c.Observe(context.Background(), []Workload{{"apps", "server"}}, at)
	o := r.Observations[Workload{"apps", "server"}]
	if calls != 4 || r.DegradedTo != "24h0m0s" || o.Window != "24h0m0s" || o.Plaintext == nil || *o.Plaintext || r.Warning == "" {
		t.Fatalf("result=%+v calls=%d", r, calls)
	}
}
func TestPartialFailureRetainsPositiveEvidenceOnly(t *testing.T) {
	at := time.Unix(1770000000, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Query().Get("query"), "istio_tcp") {
			w.WriteHeader(403)
			fmt.Fprint(w, "secret-token")
			return
		}
		fmt.Fprint(w, matrix(at, "none", "1"))
	}))
	defer server.Close()
	c, _ := New(Config{URL: server.URL})
	defer c.Close()
	r := c.Observe(context.Background(), []Workload{{"apps", "server"}}, at)
	o := r.Observations[Workload{"apps", "server"}]
	if r.Granted || o.Share != nil || o.Plaintext == nil || !*o.Plaintext || strings.Contains(o.UnknownReason, "secret") {
		t.Fatalf("result=%+v", r)
	}
}
func TestChunkingAuthAndRedirectBound(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer private-token" {
			t.Error("missing auth")
		}
		fmt.Fprint(w, `{"status":"success","data":{"resultType":"matrix","result":[]}}`)
	}))
	defer server.Close()
	c, _ := New(Config{URL: server.URL, BearerToken: "private-token"})
	defer c.Close()
	var workloads []Workload
	for i := 0; i < 21; i++ {
		workloads = append(workloads, Workload{fmt.Sprintf("ns%d", i), "server"})
	}
	r := c.Observe(context.Background(), workloads, time.Now())
	if calls != 4 || len(r.Observations) != 21 {
		t.Fatalf("calls=%d observations=%d", calls, len(r.Observations))
	}
	reached := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redirect.Close()
	redirectClient, _ := New(Config{URL: redirect.URL, BearerToken: "private-token"})
	defer redirectClient.Close()
	redirectClient.Observe(context.Background(), []Workload{{"apps", "server"}}, time.Now())
	if reached {
		t.Fatal("followed redirect")
	}
}
func TestInvalidResponsesNeverPass(t *testing.T) {
	at := time.Unix(1770000000, 0)
	for _, body := range []string{matrix(at, "mutual_tls", "NaN"), matrix(at, "none", "-1"), `{"status":"success","warnings":["partial"],"data":{"resultType":"matrix","result":[]}}`, strings.Replace(matrix(at, "none", "1"), "destination_workload_namespace", "source_namespace", 1), strings.Replace(matrix(at, "none", "1"), fmt.Sprint(at.Unix()), "0", 1), `{"status":"success","data":{"resultType":"vector","result":[]}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }))
		c, _ := New(Config{URL: server.URL})
		r := c.Observe(context.Background(), []Workload{{"apps", "server"}}, at)
		c.Close()
		server.Close()
		o := r.Observations[Workload{"apps", "server"}]
		if r.Granted || o.Share != nil || o.Plaintext != nil || o.UnknownReason == "" {
			data, _ := json.Marshal(o)
			t.Fatalf("invalid evidence passed: %s", data)
		}
	}
}
func TestConfigurationValidation(t *testing.T) {
	for _, config := range []Config{{URL: "https://user:password@example.com"}, {URL: "https://example.com/?token=private"}, {URL: "file:///tmp/prom"}, {URL: "http://localhost", Lookback: -time.Hour}, {URL: "http://localhost", Step: time.Nanosecond}, {URL: "http://localhost", CertFile: "only-cert"}, {URL: "http://localhost", BearerToken: "bad\nheader"}} {
		if c, err := New(config); err == nil {
			c.Close()
			t.Errorf("accepted invalid config %+v", config)
		}
	}
}
func boolPointer(v bool) *bool        { return &v }
func floatPointer(v float64) *float64 { return &v }
func sameBool(a, b *bool) bool        { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func sameFloat(a, b *float64) bool    { return a == nil && b == nil || a != nil && b != nil && *a == *b }

func TestAggregateOverflowIsUnknownAndJSONSafe(t *testing.T) {
	at := time.Unix(1770000000, 0)
	for _, policy := range []string{"mutual_tls", "none"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, matrix(at, policy, "1e308")) }))
		c, _ := New(Config{URL: server.URL})
		r := c.Observe(context.Background(), []Workload{{"apps", "server"}}, at)
		c.Close()
		server.Close()
		o := r.Observations[Workload{"apps", "server"}]
		if o.Share != nil || o.UnknownReason == "" || o.NoTraffic {
			t.Fatalf("overflow did not become unknown: %+v", o)
		}
		if policy == "none" && (o.Plaintext == nil || !*o.Plaintext) {
			t.Fatal("lost independent positive plaintext observation")
		}
		if _, err := json.Marshal(o); err != nil {
			t.Fatalf("unknown observation breaks JSON: %v", err)
		}
	}
}

func TestBodyTimeoutDegradesButParentCancellationDoesNot(t *testing.T) {
	at := time.Unix(1770000000, 0)
	for _, cancelParent := range []bool{false, true} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if strings.Contains(r.URL.Query().Get("query"), "[604800s]") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				w.(http.Flusher).Flush()
				<-r.Context().Done()
				return
			}
			fmt.Fprint(w, matrix(at, "mutual_tls", "1"))
		}))
		c, _ := New(Config{URL: server.URL, Timeout: 20 * time.Millisecond})
		ctx := context.Background()
		var cancel context.CancelFunc
		if cancelParent {
			ctx, cancel = context.WithTimeout(ctx, 5*time.Millisecond)
		}
		r := c.Observe(ctx, []Workload{{"apps", "server"}}, at)
		if cancel != nil {
			cancel()
		}
		c.Close()
		server.Close()
		if !cancelParent && (r.DegradedTo != "24h0m0s" || calls != 4) {
			t.Fatalf("body timeout did not degrade: %+v calls=%d", r, calls)
		}
		if cancelParent && r.DegradedTo != "" {
			t.Fatal("retried after caller canceled")
		}
	}
}

func TestFractionalLookbackRejected(t *testing.T) {
	if c, err := New(Config{URL: "http://localhost", Lookback: 1500 * time.Millisecond, Step: time.Second}); err == nil {
		c.Close()
		t.Fatal("fractional lookback was silently truncated")
	}
}

func TestMutualTLSClientAuthenticationAndTrust(t *testing.T) {
	at := time.Unix(1770000000, 0)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.PeerCertificates) == 0 {
			t.Error("missing client certificate")
		}
		fmt.Fprint(w, matrix(at, "mutual_tls", "1"))
	}))
	server.TLS = &tls.Config{ClientAuth: tls.RequireAnyClientCert, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	defer server.Close()
	certificate := server.TLS.Certificates[0]
	key, err := x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile := filepath.Join(dir, "client.pem")
	keyFile := filepath.Join(dir, "client-key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := New(Config{URL: server.URL, CAFile: certFile, CertFile: certFile, KeyFile: keyFile})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	r := c.Observe(context.Background(), []Workload{{"apps", "server"}}, at)
	if !r.Granted || r.Observations[Workload{"apps", "server"}].Share == nil {
		t.Fatalf("mTLS client auth failed: %+v", r)
	}
}

func TestCostBoundsAndSampleFailure(t *testing.T) {
	at := time.Unix(1770000000, 0)
	for _, kind := range []string{"response bytes", "returned series", "server sample limit"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				switch kind {
				case "response bytes":
					fmt.Fprint(w, strings.Repeat("x", maxResponseBytes+1))
				case "returned series":
					fmt.Fprint(w, `{"status":"success","data":{"resultType":"matrix","result":[`+strings.Repeat(`{},`, maxSeries)+`{}]}}`)
				case "server sample limit":
					w.WriteHeader(422)
					fmt.Fprint(w, `{"status":"error","errorType":"execution","error":"query processing would load too many samples into memory"}`)
				}
			}))
			defer server.Close()
			c, _ := New(Config{URL: server.URL})
			defer c.Close()
			r := c.Observe(context.Background(), []Workload{{"apps", "server"}}, at)
			if calls != 4 || r.DegradedTo != "24h0m0s" || r.Granted || r.Observations[Workload{"apps", "server"}].Share != nil {
				t.Fatalf("cost bound failed: %+v calls=%d", r, calls)
			}
		})
	}
	c, _ := New(Config{URL: "http://127.0.0.1:1"})
	defer c.Close()
	var workloads []Workload
	for i := 0; i <= maxNamespaces; i++ {
		workloads = append(workloads, Workload{fmt.Sprint(i), "server"})
	}
	r := c.Observe(context.Background(), workloads, at)
	if r.Granted || len(r.Observations) != maxNamespaces+1 || r.Warning != "Prometheus namespace limit exceeded" {
		t.Fatal("namespace cost bound failed")
	}
}

func TestMixedEventShareAndEmptyFamily(t *testing.T) {
	at := time.Unix(1770000000, 0)
	for _, emptyTCP := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Query().Get("query"), "istio_tcp") {
				if emptyTCP {
					fmt.Fprint(w, `{"status":"success","data":{"resultType":"matrix","result":[]}}`)
				} else {
					fmt.Fprint(w, matrix(at, "none", "2"))
				}
				return
			}
			fmt.Fprint(w, matrix(at, "mutual_tls", "4"))
		}))
		c, _ := New(Config{URL: server.URL})
		r := c.Observe(context.Background(), []Workload{{"apps", "server"}}, at)
		c.Close()
		server.Close()
		o := r.Observations[Workload{"apps", "server"}]
		want := 2.0 / 3.0
		if emptyTCP {
			want = 1
		}
		if o.Share == nil || *o.Share != want || o.Plaintext == nil || *o.Plaintext == emptyTCP {
			t.Fatalf("mixed event observation=%+v", o)
		}
	}
}
