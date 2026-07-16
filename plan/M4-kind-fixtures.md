# M4 — Kind Acceptance Harness and RBAC Proof

Branch: `m4-kind-fixtures`

## Goal
Real-cluster acceptance tests: disposable Kind + pinned upstream Istio (sidecar mode), fixture workloads/policies, golden expected findings, and machine-proof of the read-only guarantee.

## Context
SPEC.md §13, §19 Phase 2, §20 (validation bullets). AGENTS.md hard constraints 1–4.

## Deliverables
- [x] `make kind-up`: Kind cluster + istioctl install of a pinned Istio version from a single `versions.yaml` (the future version-matrix automation hook).
- [x] Fixture set `test/fixtures/sidecar-basic/`: namespaces exercising the M2 resolver table in a live cluster — strict ns, permissive ns, port-level override, DR contradiction, not-in-mesh ns, unlabeled/unclassified ns.
- [x] Golden files: expected canonical-JSON findings/postures per fixture (stable ordering; scrub timestamps). `make e2e` diffs actual vs golden with a readable failure report.
- [x] Published RBAC manifests in `deploy/rbac/`: namespace Role, cluster ClusterRole, optional add-ons — matching SPEC §13 exactly, with per-rule "why" comments.
- [x] **RBAC proof test**: e2e run executes the scan under a ServiceAccount bound ONLY to the published ClusterRole; a second run under the namespace Role verifies degradation to permissionSummary entries + unknown findings instead of failure.
- [x] **No-write proof**: e2e asserts via audit (kube-apiserver audit log in Kind, or an impersonating audit proxy — choose and document) that only get/list verbs were received.
- [x] CI: e2e as a separate workflow (PR-triggered but non-required initially; flip to required when stable).

## Definition of Done
- [x] `make kind-up e2e kind-down` green from a clean machine, documented in docs/dev.md.
- [x] Golden diffs are deterministic across repeated runs.
- [x] Both RBAC proofs pass and are wired into e2e permanently.

## Out of scope
Ambient fixtures (M6), multi-cluster fixtures (post-v1 per SPEC §11), managed-cloud harnesses (Phase 5).

## Deferred

- Use the pinned Istio minor as the input to M2's version-specific root-namespace selector behavior. M4 establishes the version source but does not change resolver semantics.
- Decide whether explicit sidecar-injection disablement can become conclusive `not-in-mesh` only after M6 owns ambient detection. M4 preserves the current honest unknown rather than inferring that ambient is absent.
- Flip the separate pull-request E2E workflow to required after it is stable in hosted CI.

## Summary

### Decisions

- `versions.yaml` is the sole source for Kind v0.31.0, the digest-pinned Kubernetes 1.35.0 node image, and Istio 1.30.2. Local matching tools are reused and otherwise downloaded to `.e2e/bin` from those values.
- Kind kube-apiserver audit logging is the no-write proof because it observes authenticated requests at the API server. The metadata-only policy records only the two scanner ServiceAccounts; setup activity is excluded and the log is truncated immediately before scanning.
- The fixture-manager ServiceAccount is the only harness identity that creates resources. The two scanner ServiceAccounts are separately named, receive short-lived finished kubeconfigs, and are asserted to have exactly one explicit published-profile binding each.
- Goldens retain the complete canonical report. Only `generatedAt` and `scan.clusterContext` are scrubbed. Diff failures show unified diffs and retain actual reports under `.e2e/results`.
- Istio 1.30 on Kubernetes 1.35 uses native sidecars, so the live proof accepts `istio-proxy` as either a regular container or a restartable init container. Scanner normalization already recognizes both representations.
- The port-level override golden resolves mTLS to unknown because normalized workload ports have no producer. Its paired note points to the M3 deferral.
- The DestinationRule contradiction golden retains server-side strict posture and omits `clientTLSContradiction`, the frozen canonical unavailable-evidence representation. Its paired note points to the M3 deferral; M4 does not collect DestinationRules.
- The explicitly injection-disabled fixture retains unknown data-plane membership because ambient detection is deferred. The golden records actual current behavior instead of forcing the pure resolver's not-in-mesh table result.
- The namespace Role scan denies cluster-scoped Namespace evidence and root-namespace PeerAuthentication evidence, then completes with permission-summary entries and three unknown mTLS findings.
- The E2E workflow is pull-request-triggered, separate from the main workflow, non-required by name, and uses the same concurrency/cancel-in-progress pattern.

### Flags raised

- No frozen contract or exported resolver/output shape changed.
- No RBAC API group, resource, or verb outside SPEC §13 was added. Published rules grant exactly `get` and `list`; none grants Secrets, `watch`, token creation, impersonation, or a write verb.
- M3's workload-port and DestinationRule collection deferrals remain intact. The corresponding live fixture evidence is unknown/unavailable, not synthesized from manifests.
- The pinned version matrix unblocks M2's deferred version-specific root-namespace selector analysis, but implementing that behavior remains deferred and was not silently wired in M4.
- The first generated unlabeled-workload golden included a stale ReplicaSet from an in-place fixture correction. A fresh-cluster run exposed the count mismatch; the golden was reset to the clean-cluster value and the full lifecycle was rerun successfully.

### Verification

- Exact clean lifecycle: `make kind-up e2e kind-down` green with durations 43s, 19s, and 1s respectively.
- Exact versions: Kind v0.31.0; `kindest/node:v1.35.0@sha256:452d707d4862f52530247495d180205e029056831160e22870e37e3f6c1ac31f`; Istio 1.30.2.
- Two consecutive ordinary `make e2e` runs matched every golden in 15s each. The final fresh-cluster run also matched all goldens.
- Each E2E run schema-validated all seven reports. The audit proof contained 63 scanner events, all `get`/`list`, including the namespace scanner's expected `403` on root-namespace PeerAuthentications.
- Final local `make build`, `make test`, `make lint`, and `make schema-test`: green in one combined closeout run; lint reported zero issues.
