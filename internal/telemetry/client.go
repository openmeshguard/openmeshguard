package telemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultLookback    = 168 * time.Hour
	DefaultStep        = time.Hour
	DefaultTimeout     = 30 * time.Second
	maxResponseBytes   = 8 << 20
	maxSeries          = 20000
	maxNamespaces      = 10000
	namespaceChunkSize = 20
)

// Config configures only the supplied Prometheus endpoint. Credentials never
// become report metadata; redirects are disabled to constrain their destination.
type Config struct {
	URL         string
	BearerToken string
	CAFile      string
	CertFile    string
	KeyFile     string
	Lookback    time.Duration
	Step        time.Duration
	Timeout     time.Duration
}

// Workload identifies a destination without coupling telemetry to the resolver.
type Workload struct{ Namespace, Name string }

// Observation describes observed events only. HTTP request events and TCP
// connection-opening events are combined; Share is their mixed event fraction,
// not a request-only fraction. UnknownReason is for control availability.
type Observation struct {
	Window        string
	Share         *float64
	Plaintext     *bool
	NoTraffic     bool
	UnknownReason string
}

// Result retains per-workload evidence when another namespace fails.
type Result struct {
	Observations map[Workload]Observation
	Lookback     string
	DegradedTo   string
	Granted      bool
	Warning      string
}

// Client is a bounded Prometheus API reader.
type Client struct {
	config   Config
	endpoint *url.URL
	http     *http.Client
}

// New validates local configuration. Runtime endpoint failures are returned by
// Observe as unavailable evidence, never as fatal scan errors.
func New(config Config) (*Client, error) {
	endpoint, err := url.Parse(config.URL)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, fmt.Errorf("prometheus URL must be an HTTP(S) endpoint without credentials, query, or fragment")
	}
	if config.Lookback == 0 {
		config.Lookback = DefaultLookback
	}
	if config.Step == 0 {
		config.Step = DefaultStep
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}
	if config.Lookback < time.Second || config.Lookback > 365*24*time.Hour || config.Lookback%time.Second != 0 {
		return nil, fmt.Errorf("prometheus lookback must use whole seconds between 1s and 8760h")
	}
	if config.Step < time.Second || config.Step > config.Lookback {
		return nil, fmt.Errorf("prometheus step must be between 1s and lookback")
	}
	if config.Timeout < time.Millisecond || config.Timeout > 5*time.Minute {
		return nil, fmt.Errorf("prometheus timeout must be between 1ms and 5m")
	}
	if strings.ContainsAny(config.BearerToken, "\r\n") {
		return nil, fmt.Errorf("prometheus bearer token contains invalid header characters")
	}
	if (config.CertFile == "") != (config.KeyFile == "") {
		return nil, fmt.Errorf("prometheus client certificate and key must be supplied together")
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if config.CAFile != "" {
		pem, readErr := os.ReadFile(config.CAFile)
		if readErr != nil {
			return nil, fmt.Errorf("read Prometheus CA file: %w", readErr)
		}
		pool, poolErr := x509.SystemCertPool()
		if poolErr != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("prometheus CA file contains no certificates")
		}
		tlsConfig.RootCAs = pool
	}
	if config.CertFile != "" {
		cert, certErr := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if certErr != nil {
			return nil, fmt.Errorf("load Prometheus client certificate: %w", certErr)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	transport.Proxy = nil
	transport.MaxConnsPerHost = 1
	transport.MaxIdleConnsPerHost = 1
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/v1/query_range"
	return &Client{config: config, endpoint: endpoint, http: &http.Client{Transport: transport, Timeout: config.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// Close releases connections after the one-shot scan.
func (c *Client) Close() { c.http.CloseIdleConnections() }

// Observe fixes one evaluation timestamp for all chunks and metric families.
// query_range intentionally evaluates one point (start=end): increase's range
// selector covers the complete lookback. Step is sent to the API, but does not
// divide totals into overlapping windows or truncate a partial final hour.
func (c *Client) Observe(ctx context.Context, workloads []Workload, at time.Time) Result {
	result := Result{Observations: map[Workload]Observation{}, Lookback: c.config.Lookback.String(), Granted: true}
	namespaces := map[string]bool{}
	for _, w := range workloads {
		namespaces[w.Namespace] = true
	}
	if len(namespaces) > maxNamespaces {
		result.Granted = false
		result.Warning = "Prometheus namespace limit exceeded"
		for _, w := range workloads {
			result.Observations[w] = Observation{Window: result.Lookback, UnknownReason: result.Warning}
		}
		return result
	}
	names := make([]string, 0, len(namespaces))
	for name := range namespaces {
		names = append(names, name)
	}
	sort.Strings(names)
	for start := 0; start < len(names); start += namespaceChunkSize {
		end := min(start+namespaceChunkSize, len(names))
		chunk := names[start:end]
		totals, err := c.queryChunk(ctx, chunk, at, c.config.Lookback)
		window := c.config.Lookback
		if err != nil && err.cost && c.config.Lookback > 24*time.Hour && ctx.Err() == nil {
			window = 24 * time.Hour
			totals, err = c.queryChunk(ctx, chunk, at, window)
			result.DegradedTo = window.String()
			result.Warning = "Prometheus query cost forced a 24h verification window"
		}
		if err != nil {
			result.Granted = false
		}
		for _, w := range workloads {
			if !contains(chunk, w.Namespace) {
				continue
			}
			observation := assemble(totals[w], window)
			if err != nil {
				observation.Share = nil
				observation.NoTraffic = false
				observation.UnknownReason = err.reason
				if observation.Plaintext == nil || !*observation.Plaintext {
					observation.Plaintext = nil
				}
			}
			result.Observations[w] = observation
		}
	}
	return result
}

type counts struct {
	total, mtls, plain      float64
	seen, unknown, overflow bool
}
type queryError struct {
	reason string
	cost   bool
}

func (c *Client) queryChunk(ctx context.Context, namespaces []string, at time.Time, window time.Duration) (map[Workload]counts, *queryError) {
	totals := map[Workload]counts{}
	var failure *queryError
	for _, metric := range []string{"istio_requests_total", "istio_tcp_connections_opened_total"} {
		samples, err := c.query(ctx, metric, namespaces, at, window)
		if err != nil {
			if failure == nil || err.cost {
				failure = err
			}
			continue
		}
		for _, sample := range samples {
			key := Workload{sample.Metric["destination_workload_namespace"], sample.Metric["destination_workload"]}
			count := totals[key]
			count.seen = true
			value := sample.number
			count.total += value
			switch sample.Metric["connection_security_policy"] {
			case "mutual_tls":
				count.mtls += value
			case "none":
				count.plain += value
			default:
				count.unknown = true
			}
			count.overflow = count.overflow || !finite(count.total) || !finite(count.mtls) || !finite(count.plain)
			totals[key] = count
		}
	}
	return totals, failure
}
func assemble(count counts, window time.Duration) Observation {
	o := Observation{Window: window.String()}
	if !count.seen {
		o.UnknownReason = "no destination telemetry series available"
		return o
	}
	if count.plain > 0 {
		yes := true
		o.Plaintext = &yes
	}
	if count.overflow {
		o.UnknownReason = "Prometheus event total exceeds numeric limits"
		return o
	}
	if count.unknown {
		o.UnknownReason = "connection security policy unavailable in observed telemetry"
		return o
	}
	if count.total == 0 {
		o.NoTraffic = true
		o.UnknownReason = "no traffic observed in verification window"
		return o
	}
	share := count.mtls / count.total
	if !finite(share) || share < 0 || share > 1 {
		o.UnknownReason = "Prometheus event share exceeds numeric limits"
		return o
	}
	plain := count.plain > 0
	o.Share = &share
	o.Plaintext = &plain
	return o
}

type sample struct {
	Metric map[string]string   `json:"metric"`
	Values [][]json.RawMessage `json:"values"`
	number float64
}
type response struct {
	Status    string   `json:"status"`
	ErrorType string   `json:"errorType"`
	Error     string   `json:"error"`
	Warnings  []string `json:"warnings"`
	Data      struct {
		ResultType string   `json:"resultType"`
		Result     []sample `json:"result"`
	} `json:"data"`
}

func (c *Client) query(ctx context.Context, metric string, namespaces []string, at time.Time, window time.Duration) ([]sample, *queryError) {
	alternatives := make([]string, len(namespaces))
	for i, name := range namespaces {
		alternatives[i] = regexp.QuoteMeta(name)
	}
	matcher := strconv.Quote(strings.Join(alternatives, "|"))
	expression := fmt.Sprintf("sum by (destination_workload_namespace, destination_workload, connection_security_policy) (increase(%s{reporter=\"destination\",destination_workload_namespace=~%s}[%ds]))", metric, matcher, int64(window/time.Second))
	endpoint := *c.endpoint
	params := url.Values{"query": {expression}, "start": {strconv.FormatFloat(float64(at.UnixNano())/1e9, 'f', 3, 64)}, "end": {strconv.FormatFloat(float64(at.UnixNano())/1e9, 'f', 3, 64)}, "step": {strconv.FormatFloat(c.config.Step.Seconds(), 'f', -1, 64)}, "timeout": {strconv.FormatFloat(c.config.Timeout.Seconds(), 'f', -1, 64) + "s"}}
	endpoint.RawQuery = params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, &queryError{reason: "invalid Prometheus request"}
	}
	if c.config.BearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	}
	res, err := c.http.Do(request)
	if err != nil {
		cost := errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil
		return nil, &queryError{reason: "Prometheus request unavailable", cost: cost}
	}
	defer res.Body.Close()
	if res.StatusCode == 401 || res.StatusCode == 403 {
		return nil, &queryError{reason: "Prometheus authentication or authorization denied"}
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil {
		return nil, &queryError{reason: "Prometheus response unavailable", cost: errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil}
	}
	if len(body) > maxResponseBytes {
		return nil, &queryError{reason: "Prometheus response size limit exceeded", cost: true}
	}
	var envelope response
	if err = json.Unmarshal(body, &envelope); err != nil {
		return nil, &queryError{reason: "invalid Prometheus response"}
	}
	if res.StatusCode != http.StatusOK || envelope.Status != "success" {
		cost := envelope.ErrorType == "timeout" || strings.Contains(strings.ToLower(envelope.Error), "too many samples") || strings.Contains(strings.ToLower(envelope.Error), "sample limit")
		return nil, &queryError{reason: "Prometheus query failed", cost: cost}
	}
	if len(envelope.Warnings) > 0 {
		return nil, &queryError{reason: "Prometheus returned incomplete query evidence"}
	}
	if envelope.Data.ResultType != "matrix" {
		return nil, &queryError{reason: "unexpected Prometheus result type"}
	}
	if len(envelope.Data.Result) > maxSeries {
		return nil, &queryError{reason: "Prometheus series limit exceeded", cost: true}
	}
	seen := map[string]bool{}
	for i := range envelope.Data.Result {
		s := &envelope.Data.Result[i]
		ns := s.Metric["destination_workload_namespace"]
		name := s.Metric["destination_workload"]
		if !contains(namespaces, ns) || name == "" || name == "unknown" || len(s.Metric) != 3 || len(s.Values) != 1 || len(s.Values[0]) != 2 {
			return nil, &queryError{reason: "invalid Prometheus destination aggregation"}
		}
		key := ns + "\x00" + name + "\x00" + s.Metric["connection_security_policy"]
		if seen[key] {
			return nil, &queryError{reason: "duplicate Prometheus aggregate"}
		}
		seen[key] = true
		var timestamp float64
		var value string
		if json.Unmarshal(s.Values[0][0], &timestamp) != nil || math.Abs(timestamp-float64(at.UnixNano())/1e9) > 0.01 || json.Unmarshal(s.Values[0][1], &value) != nil {
			return nil, &queryError{reason: "invalid Prometheus sample"}
		}
		number, parseErr := strconv.ParseFloat(value, 64)
		if parseErr != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
			return nil, &queryError{reason: "invalid Prometheus counter increase"}
		}
		s.number = number
	}
	return envelope.Data.Result, nil
}
func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
