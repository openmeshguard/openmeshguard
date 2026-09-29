# M7 — Runtime Verification (Prometheus)

Branch: `m7-runtime-verification`

## Goal
The thesis lands: MG-MTLS-101/102 verify declared posture against observed traffic, with the guardrails and degradation semantics from SPEC §8.

## Context
SPEC.md §8 (controls, rules, decided guardrails), §21 decisions. Canonical schema `verified` object. Metric family: istio_requests_total / istio_tcp_connections_opened_total with connection_security_policy.

## Deliverables
- [x] `internal/telemetry`: Prometheus HTTP API client (bearer token + mTLS client auth), no other backends.
- [x] Queries aggregate server-side at workload granularity ONLY (`sum by (destination_workload_namespace, destination_workload, connection_security_policy)`); HTTP and TCP metric families both covered.
- [x] Guardrails per SPEC §8: default 168h lookback (configurable), default step 1h, 30s per-query timeout, per-namespace chunking on large meshes, degradation to 24h with report warning (`scan.dataSources.prometheus.degradedTo`), never fail the scan on telemetry cost.
- [x] Verified posture assembly per schema: status corroborated/contradicted/no-traffic-observed/unknown; mtlsTrafficShare; plaintextObserved; plaintextSources (optional empty/omitted: destination-only queries do not retain source attribution, human-approved 2026-09-29).
- [x] Contradiction rule: declared strict + plaintext observed ⇒ status contradicted AND the MG-MTLS-101 finding is critical regardless of pack severity (engine rule per SPEC §16).
- [x] MG-MTLS-101/102 activated in the built-in pack (runtime evidenceType, requires verified.*); absent Prometheus ⇒ mechanical unknowns via requires.
- [x] Unit tests against a fake Prometheus (recorded responses): corroborated, contradicted, no-traffic, timeout-degradation, chunking, auth failure ⇒ permissionSummary entry.
- [x] E2E (best-effort): Kind fixture with Prometheus + traffic generator producing at least one plaintext and one mTLS flow; if flaky, keep as nightly-only and document.
- [x] Report/HTML: Verified column populated; "no telemetry access" rendering verified against golden.

## Definition of Done
- [x] Declared and verified never blended anywhere in output (grep-able field-level check in schema-test).
- [x] All guardrail behaviors covered by tests; a telemetry-less scan preserves pre-M7 declared workload fields and configuration findings. Human-approved 2026-09-29: additive runtime unknown/not-applicable findings, related score unknown counts, optional Prometheus permission evidence, and updated built-in pack provenance are expected. The frozen JSON schema remains unchanged.
- [x] The demo sentence works end to end: a fixture report names workloads that actually received plaintext in the window.

## Out of scope
Other telemetry backends, recording-rule management (docs only), runtime authz verification (post-v1).

## Approved interpretation and implementation decisions

- Human-approved 2026-09-29: activate both runtime controls without an endpoint,
  allowing their additive unavailable outcomes rather than requiring impossible
  byte-identical full reports. Existing declared results remain unchanged.
- Human-approved 2026-09-29: retain destination-only grouping; omit/empty optional
  plaintext sources and make no source attribution claim.
- One-point `query_range` uses a fixed evaluation timestamp and reset-aware
  `increase(metric[lookback])` before aggregation. Default step 1h is transmitted
  but never used to sum overlapping window totals.
- Valid zero increases mean no traffic observed; missing series, unsupported
  destination-reporter semantics, and unknown security labels remain unknown.
- mTLS share combines HTTP request events and TCP connection-opening events.
- Recording rules are documented recommendations only, consistent with this
  milestone's recording-rule-management exclusion.
- Details and limitations: [Prometheus runtime verification](../docs/telemetry/prometheus.md).

## Validation evidence — 2026-09-29

- `make test lint schema-test` passed; lint reported zero issues. Runtime schema
  tests preserve declared fields, and compatibility guards reject drift in
  declared findings, permissions, and prior mTLS unknown counts.
- Guarded `UPDATE_GOLDEN=1 make e2e` passed in 69s with 456 approved scanner list
  audit events. All 22 migrated goldens separately matched their pre-M7 declared
  projections and exact prior unknown counts.
- `make e2e-runtime` passed against real Kind/Istio traffic. Two bounded Jobs
  completed; receiver metrics contained HTTP and TCP events with both `none`
  and `mutual_tls` security labels (40 events in each raw aggregate).
- The canonical report names `omg-runtime/runtime-receiver` as receiving
  plaintext within `5m0s`; its declared posture remains `permissive`, both runtime
  controls produce observed open findings, and its mixed mTLS event share is
  approximately 0.5005. This corroborates the permissive declaration while
  still reporting the observed plaintext risk.
- Live artifacts: `.e2e/runtime-results/report.json`, `metrics.json`, and
  `audit.jsonl`. The report passed schema validation and the scanner audit
  contained 20 API events with only permitted reads. Harness administrator
  fixture setup and forwarding remain separate from scanner actions.
- Implementation acceptance is complete. v0.2.0 release publication remains
  pending the reviewed commit/tag and release workflow verification.

## Deferred

- Source attribution requires an explicit future query-cardinality decision.
- Ambient destination reporting, federated endpoint cluster attribution, and
  recording-rule selection require separate validation/contracts.
