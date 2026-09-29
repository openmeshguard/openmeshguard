# Runtime verification with Prometheus

OpenMeshGuard reads Istio counters from one configured Prometheus endpoint. It
keeps declared mTLS and authorization unchanged and adds runtime evidence beside
them. The endpoint must contain metrics for the scanned cluster only: destination
namespace and workload names cannot disambiguate federated clusters.

```sh
openmeshguard scan --namespace payments \
  --prometheus-url https://prometheus.example.com \
  --prometheus-token-file /path/to/prometheus-token \
  --prometheus-lookback 168h > report.json
openmeshguard report --input report.json --output report.html
```

For client certificate authentication, replace the token-file option with
`--prometheus-client-cert`, `--prometheus-client-key`, and optionally
`--prometheus-ca-file`. System CA roots remain trusted. The scanner never disables
TLS verification, follows redirects, or routes requests through an environment
proxy. CLI credentials require HTTPS except for local loopback fixtures. Tokens
are read from files rather than command arguments and never enter JSON or logs.
URL user information, query parameters, and fragments are rejected.

## Evidence and controls

`MG-MTLS-101` fails when positive destination plaintext events are observed.
Declared strict plus plaintext is always critical, including when a scan-config
severity override requests a lower severity. A runtime finding carries observed
confidence and a Prometheus resolution step; a strict contradiction also retains
the declared mTLS resolution chain.

`MG-MTLS-102` fails when the known observed event share using mutual TLS is below
100%. Its denominator combines **HTTP request events** and **TCP connection-opening
events**, not bytes or HTTP requests alone. This is a mixed event measure and
cannot compare traffic volume across protocols. Persistent TCP connections opened
before the window are not measured by the opening counter.

The scanner queries `istio_requests_total` and
`istio_tcp_connections_opened_total`, with `reporter="destination"`. Each query
applies `increase` to individual counters before summing by exactly
`destination_workload_namespace`, `destination_workload`, and
`connection_security_policy`. Prometheus corrects counter resets and extrapolates
observed increases; these are estimates from available scrapes, not packet-level
proof or a guarantee of full retention and scrape coverage.

`connection_security_policy="none"` identifies observed plaintext, and
`"mutual_tls"` identifies observed mutual TLS. Other or missing values cannot
establish security posture. The destination-only implementation is validated for
sidecars. Ambient, mixed, unknown enrollment, and ambiguous namespace/name matches
remain unknown rather than claiming equivalent verification.

No source labels are retained. The optional `plaintextSources` field is omitted
or empty; the report makes no source-attribution claim. There are no per-pod,
source-principal, or source-namespace queries.

A valid zero event increase yields `no-traffic-observed`, with both control inputs
unavailable so that no traffic cannot pass either control. Empty vectors yield
unknown, because absent traffic series cannot establish telemetry availability.
An access error produces unknown findings and an optional Prometheus permission
entry. With partial query failures, confirmed positive plaintext remains evidence
for 101, while the share required by 102 is unavailable. Invalid, oversized,
nonfinite, duplicate, or warning-bearing API responses cannot pass controls.

## Windows and cost limits

Defaults are a 168h lookback, 1h API step, and 30s per-query timeout. Lookbacks use
whole seconds between 1s and 8760h. `--prometheus-step` must be between 1s and the
lookback; `--prometheus-timeout` must be between 1ms and 5m.

The API uses a **one-point range query**: `start` and `end` are the same fixed scan
evaluation timestamp. The PromQL range selector covers the complete requested
lookback. Step is supplied to the API but does not partition or sum totals. This
avoids summing overlapping rolling windows and covers partial final hours exactly.
It also avoids evaluating the same seven-day total at 169 historical timestamps.

Namespaces are sorted and queried in chunks of at most 20, with sequential
requests, at most 10,000 namespaces, 20,000 returned series per response, and an
8MiB response limit. The complete telemetry phase has a five-minute deadline.
On timeout or sample/response cost limits, a chunk retries both metric families
once using a 24h lookback when the original window is longer. It discards original
window totals before the retry. Caller cancellation, authentication failures,
malformed responses, and unknown security labels do not widen scope or trigger
cost retries.

Degradation is recorded in `scan.dataSources.prometheus.degradedTo`; HTML shows a
warning, and every workload has its actual window. Successful chunks may retain
the requested window while another chunk degrades. Telemetry failures never abort
the cluster scan.

## Compatibility and operational limits

M7 adds runtime findings even without a Prometheus endpoint. Existing declared
workload fields and configuration findings retain their meaning and values.
Additional runtime unknown/not-applicable findings, score unknown counts,
Prometheus permission evidence, and built-in pack provenance are intentional.
The JSON schema and existing exported resolver/output interfaces are unchanged.

A seven-day lookback can include traffic from before the currently declared policy
was installed. A contradiction means current strict posture and plaintext observed
within the stated window; investigate policy changes and metric coverage before
attributing an active bypass.

For very large meshes, operators can create workload-aggregated recording rules
outside the scanner. Prefer counters/rates that retain destination namespace,
workload, security policy, and destination reporter semantics; apply reset-aware
rates before aggregating. M7 does not discover, configure, or select recording
rules automatically. Its default queries continue to use the standard raw Istio
counter families.

References: [Istio standard metrics](https://istio.io/latest/docs/reference/config/metrics/),
[Prometheus counter increase](https://prometheus.io/docs/prometheus/latest/querying/functions/#increase),
and [Prometheus HTTP API](https://prometheus.io/docs/prometheus/latest/querying/api/).

## Local live demonstration

Run `make kind-up` followed by `make e2e-runtime` to install a bounded sidecar
traffic fixture and scan it with the published read-only role. This optional
harness performs administrator setup separately from scanning and requires
both plaintext and mutual TLS HTTP/TCP observations. Do not run it concurrently
with `make e2e`. See the [fixture instructions](../../test/fixtures/runtime-verification/README.md).
