package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/openmeshguard/openmeshguard/internal/collect"
	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/openmeshguard/openmeshguard/internal/output"
	"github.com/openmeshguard/openmeshguard/internal/resolver"
	"github.com/openmeshguard/openmeshguard/internal/telemetry"
)

func runtimeClient(opts scanOptions) (*telemetry.Client, error) {
	if opts.Prometheus.URL == "" {
		if opts.PrometheusTokenFile != "" || opts.Prometheus.CAFile != "" || opts.Prometheus.CertFile != "" || opts.Prometheus.KeyFile != "" {
			return nil, fmt.Errorf("prometheus credentials require --prometheus-url")
		}
		return nil, nil
	}
	config := opts.Prometheus
	if config.BearerToken != "" || opts.PrometheusTokenFile != "" || config.CertFile != "" || config.KeyFile != "" {
		endpoint, err := url.Parse(config.URL)
		if err == nil && endpoint.Scheme == "http" && endpoint.Hostname() != "localhost" && endpoint.Hostname() != "127.0.0.1" && endpoint.Hostname() != "::1" {
			return nil, fmt.Errorf("prometheus credentials require HTTPS except on loopback endpoints")
		}
	}
	if opts.PrometheusTokenFile != "" {
		file, err := os.Open(opts.PrometheusTokenFile)
		if err != nil {
			return nil, fmt.Errorf("open Prometheus bearer token file: %w", err)
		}
		defer file.Close()
		token, err := io.ReadAll(io.LimitReader(file, 16385))
		if err != nil {
			return nil, fmt.Errorf("read Prometheus bearer token file: %w", err)
		}
		if len(token) > 16384 {
			return nil, fmt.Errorf("prometheus token file exceeds 16KiB")
		}
		config.BearerToken = strings.TrimSpace(string(token))
	}
	return telemetry.New(config)
}

func runtimeInputs(ctx context.Context, client *telemetry.Client, opts scanOptions, workloads []engine.WorkloadInput, at time.Time) (output.RuntimeInput, collect.Permission) {
	runtime := output.RuntimeInput{Enabled: client != nil, URL: opts.Prometheus.URL, Verified: map[string]map[string]any{}}
	permission := collect.Permission{APIGroup: "prometheus", Resource: "query", Verbs: []string{"get"}, Optional: true, AffectedControls: []string{"MG-MTLS-101", "MG-MTLS-102"}, Impact: "Runtime verification unavailable: no telemetry access"}
	if client == nil {
		for i := range workloads {
			if workloads[i].Availability == nil {
				workloads[i].Availability = map[string]engine.Availability{}
			}
			workloads[i].Availability["verified"] = engine.Availability{Reason: "no telemetry access: Prometheus endpoint not configured"}
		}
		return runtime, permission
	}
	refs := make([]telemetry.Workload, 0, len(workloads))
	matches := map[telemetry.Workload]int{}
	for _, workload := range workloads {
		if workload.Posture.Mode == resolver.ModeNotApplicable {
			continue
		}
		ref := telemetry.Workload{Namespace: workload.Posture.Ref.Namespace, Name: workload.Posture.Ref.Name}
		refs = append(refs, ref)
		matches[ref]++
	}
	// Query cost is bounded across the entire one-shot telemetry phase as well as
	// per request. Cancellation never triggers another reduced-window attempt.
	queryContext, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	observed := client.Observe(queryContext, refs, at)
	runtime.Lookback = observed.Lookback
	runtime.DegradedTo = observed.DegradedTo
	permission.Granted = observed.Granted
	permission.Impact = "Runtime controls use destination workload aggregates; source attribution unavailable"
	if observed.Warning != "" {
		permission.Impact += "; " + observed.Warning
	}
	if !observed.Granted {
		permission.Impact += "; one or more Prometheus queries unavailable"
	}
	for i := range workloads {
		workload := &workloads[i]
		if workload.Posture.Mode == resolver.ModeNotApplicable {
			continue
		}
		ref := telemetry.Workload{Namespace: workload.Posture.Ref.Namespace, Name: workload.Posture.Ref.Name}
		o := observed.Observations[ref]
		if matches[ref] > 1 {
			o.Share = nil
			o.Plaintext = nil
			o.NoTraffic = false
			o.UnknownReason = "destination workload labels match multiple collected workload kinds"
		}
		if workload.Posture.Mode != resolver.ModeSidecar {
			o.Share = nil
			o.Plaintext = nil
			o.NoTraffic = false
			o.UnknownReason = "destination-reporter verification is not validated for this data plane"
		}
		status := "unknown"
		switch {
		case o.Plaintext != nil && *o.Plaintext && workload.Posture.MTLS.Effective == resolver.MTLSStrict:
			status = "contradicted"
		case o.NoTraffic:
			status = "no-traffic-observed"
		case o.Share != nil && o.UnknownReason == "":
			switch workload.Posture.MTLS.Effective {
			case resolver.MTLSStrict:
				if o.Plaintext != nil && !*o.Plaintext {
					status = "corroborated"
				}
			case resolver.MTLSPermissive:
				status = "corroborated"
			case resolver.MTLSDisabled:
				if *o.Share == 0 {
					status = "corroborated"
				}
			}

		}
		verified := map[string]any{"status": status, "window": o.Window, "plaintextSources": []string{}}
		if o.Share != nil {
			verified["mtlsTrafficShare"] = *o.Share
		}
		if o.Plaintext != nil {
			verified["plaintextObserved"] = *o.Plaintext
		}
		workload.Verified = verified
		runtime.Verified[ref.Namespace+"/"+ref.Name] = verified
		if workload.Availability == nil {
			workload.Availability = map[string]engine.Availability{}
		}
		if o.Share == nil {
			workload.Availability["verified.mtlsTrafficShare"] = engine.Availability{Reason: o.UnknownReason}
		}
		if o.Plaintext == nil {
			workload.Availability["verified.plaintextObserved"] = engine.Availability{Reason: o.UnknownReason}
		}
	}
	return runtime, permission
}
