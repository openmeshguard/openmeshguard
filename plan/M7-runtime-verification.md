# M7 — Runtime Verification (Prometheus)

Branch: `m7-runtime-verification`

## Goal
The thesis lands: MG-MTLS-101/102 verify declared posture against observed traffic, with the guardrails and degradation semantics from SPEC §8.

## Context
SPEC.md §8 (controls, rules, decided guardrails), §21 decisions. Canonical schema `verified` object. Metric family: istio_requests_total / istio_tcp_connections_opened_total with connection_security_policy.

## Deliverables
- [ ] `internal/telemetry`: Prometheus HTTP API client (bearer token + mTLS client auth), no other backends.
- [ ] Queries aggregate server-side at workload granularity ONLY (`sum by (destination_workload_namespace, destination_workload, connection_security_policy)`); HTTP and TCP metric families both covered.
- [ ] Guardrails per SPEC §8: default 168h lookback (configurable), default step 1h, 30s per-query timeout, per-namespace chunking on large meshes, degradation to 24h with report warning (`scan.dataSources.prometheus.degradedTo`), never fail the scan on telemetry cost.
- [ ] Verified posture assembly per schema: status corroborated/contradicted/no-traffic-observed/unknown; mtlsTrafficShare; plaintextObserved; plaintextSources (source workload identities where cardinality-safe, else namespaces).
- [ ] Contradiction rule: declared strict + plaintext observed ⇒ status contradicted AND the MG-MTLS-101 finding is critical regardless of pack severity (engine rule per SPEC §16).
- [ ] MG-MTLS-101/102 activated in the built-in pack (runtime evidenceType, requires verified.*); absent Prometheus ⇒ mechanical unknowns via requires.
- [ ] Unit tests against a fake Prometheus (recorded responses): corroborated, contradicted, no-traffic, timeout-degradation, chunking, auth failure ⇒ permissionSummary entry.
- [ ] E2E (best-effort): Kind fixture with Prometheus + traffic generator producing at least one plaintext and one mTLS flow; if flaky, keep as nightly-only and document.
- [ ] Report/HTML: Verified column populated; "no telemetry access" rendering verified against golden.

## Definition of Done
- Declared and verified never blended anywhere in output (grep-able field-level check in schema-test).
- All guardrail behaviors covered by tests; scan of a telemetry-less cluster is byte-identical to pre-M7 output except schema-required nulls.
- The demo sentence works end to end: a fixture report names workloads that actually received plaintext in the window.

## Out of scope
Other telemetry backends, recording-rule management (docs only), runtime authz verification (post-v1).
