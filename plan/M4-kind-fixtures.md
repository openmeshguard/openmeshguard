# M4 — Kind Acceptance Harness and RBAC Proof

Branch: `m4-kind-fixtures`

## Goal
Real-cluster acceptance tests: disposable Kind + pinned upstream Istio (sidecar mode), fixture workloads/policies, golden expected findings, and machine-proof of the read-only guarantee.

## Context
SPEC.md §13, §19 Phase 2, §20 (validation bullets). AGENTS.md hard constraints 1–4.

## Deliverables
- [ ] `make kind-up`: Kind cluster + istioctl install of a pinned Istio version from a single `versions.yaml` (the future version-matrix automation hook).
- [ ] Fixture set `test/fixtures/sidecar-basic/`: namespaces exercising the M2 resolver table in a live cluster — strict ns, permissive ns, port-level override, DR contradiction, not-in-mesh ns, unlabeled/unclassified ns.
- [ ] Golden files: expected canonical-JSON findings/postures per fixture (stable ordering; scrub timestamps). `make e2e` diffs actual vs golden with a readable failure report.
- [ ] Published RBAC manifests in `deploy/rbac/`: namespace Role, cluster ClusterRole, optional add-ons — matching SPEC §13 exactly, with per-rule "why" comments.
- [ ] **RBAC proof test**: e2e run executes the scan under a ServiceAccount bound ONLY to the published ClusterRole; a second run under the namespace Role verifies degradation to permissionSummary entries + unknown findings instead of failure.
- [ ] **No-write proof**: e2e asserts via audit (kube-apiserver audit log in Kind, or an impersonating audit proxy — choose and document) that only get/list verbs were received.
- [ ] CI: e2e as a separate workflow (PR-triggered but non-required initially; flip to required when stable).

## Definition of Done
- `make kind-up e2e kind-down` green from a clean machine, documented in docs/dev.md.
- Golden diffs are deterministic across repeated runs.
- Both RBAC proofs pass and are wired into e2e permanently.

## Out of scope
Ambient fixtures (M6), multi-cluster fixtures (post-v1 per SPEC §11), managed-cloud harnesses (Phase 5).
