# M1 — Walking Skeleton (one control, end to end)

Branch: `m1-walking-skeleton`

## Goal
`openmeshguard scan` connects to a cluster read-only, collects a minimal resource set, resolves a PROVISIONAL effective mTLS posture (mesh + namespace-level PeerAuthentication only), and emits canonical JSON with inventory, workloadPostures, permissionSummary, and findings from one hardwired provisional check. Thin but real, top to bottom.

## Context
SPEC.md §6, §7 (partial), §12, §13. Contracts: canonical schema (authoritative), resolver types.

## Deliverables
- [ ] `internal/collect`: typed read-only collectors for namespaces, pods, deployments/replicasets/statefulsets/daemonsets, services, PeerAuthentications. List-based, bounded concurrency, namespace-scoped or all-namespaces.
- [ ] Fake-client unit tests including an **action audit test**: assert the only verbs ever issued are get/list (this test is permanent and grows with every collector).
- [ ] Graceful permission degradation: forbidden/notfound per resource recorded into permissionSummary; scan continues.
- [ ] `internal/normalize`: raw objects → normalized inventory + WorkloadInput (subset: no ports/DR/authz yet).
- [ ] Provisional resolver implementation covering ONLY mesh-wide + namespace PA precedence, returning chains; port-level, workload-selector, and DR interplay explicitly return `unknown` with UnknownReason "not yet implemented (M2)".
- [ ] `internal/output`: canonical JSON writer; every scan output validates against the schema (extend schema-test to run on real output).
- [ ] One provisional finding path (hardcoded, replaced in M3): emit a finding when a namespace's resolved posture is permissive. Mark clearly `// PROVISIONAL: replaced by CEL engine in M3`.
- [ ] Sidecar data-plane detection (istio-proxy container / injection labels); ambient detection stub returns unknown.

## Definition of Done
- `make test lint schema-test` green; action-audit test in place.
- Manual verification against a Kind cluster with Istio + a namespace PA (document exact commands in the task file when done).
- Scan output validates against the canonical schema with zero warnings.

## Out of scope
Full resolver semantics (M2), CEL (M3), HTML/SARIF (M6), Prometheus (M7).
