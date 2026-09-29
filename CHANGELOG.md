# Changelog

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
