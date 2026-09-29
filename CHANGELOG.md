# Changelog

## v0.2.0 — 2026-09-29

Optional Prometheus runtime verification alongside unchanged declared posture.

- MG-MTLS-101 identifies observed plaintext; declared strict plus observed
  plaintext is always critical. MG-MTLS-102 evaluates the mutual TLS share of
  observed HTTP request and TCP connection-opening events.
- Authenticated Prometheus HTTP API access with bearer token files or mTLS client
  certificates; verified posture remains separate in canonical JSON and HTML.
- Destination-workload aggregation, bounded namespace chunks and responses,
  seven-day default lookback, thirty-second query timeout, and explicit 24h
  degradation on query cost failures.
- Missing telemetry, unavailable security labels, and no observed traffic never
  pass runtime controls. Telemetry failures degrade evidence without aborting
  the cluster scan.
- Guarded acceptance preserves existing declared findings and prior unknown
  counts; optional Kind traffic fixtures demonstrate real plaintext and mutual
  TLS over HTTP and TCP with read-only scanner audit evidence.

The JSON schema and existing exported resolver/output interfaces are unchanged.
Telemetry-less reports gain runtime unknown/not-applicable findings, related
score unknown counts, and Prometheus permission evidence. Source attribution and
ambient/mixed runtime verification remain unavailable; endpoints must be scoped
to the scanned cluster. mTLS share is a mixed event measure, not HTTP request or
byte coverage.

## v0.1.0

First Community release of declared and resolved Istio security posture.

- Read-only, typed, bounded Kubernetes/Istio/Gateway API collection with explicit
  evidence gaps and permission degradation; no Secrets access.
- Effective per-workload mTLS and authorization resolution with ordered evidence
  chains for sidecar, ambient, and mixed meshes.
- YAML/CEL controls, environment classification, ownership, and owner-bound,
  annotation-referenced exceptions that preserve findings.
- Canonical JSON, self-contained HTML, SARIF 2.1.0, weighted scores, and opt-in CI
  thresholds. Unknown evidence never silently passes or fails.
- Linux/macOS binaries for amd64/arm64, SHA-256 checksums, and keyless Sigstore
  signatures for every archive and the checksum file.

Prometheus runtime verification is not included. Verified posture is explicitly
unavailable in v0; M7 adds traffic-backed verification. Offline scanning,
cross-cluster correlation, and lifecycle controls remain future work.
