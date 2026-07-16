# M4 — Kind Acceptance Harness and RBAC Proof

Branch: `m4-kind-fixtures`

## Goal
Real-cluster acceptance tests: disposable Kind + pinned upstream Istio (sidecar mode), fixture workloads/policies, golden expected findings, and machine-proof of the read-only guarantee.

## Context
SPEC.md §13, §19 Phase 2, §20 (validation bullets). AGENTS.md hard constraints 1–4.

## Deliverables
- [x] `make kind-up`: Kind cluster + istioctl install of a pinned Istio version from a single `versions.yaml` (the future version-matrix automation hook).
- [x] Fixture set `test/fixtures/sidecar-basic/`: live M2 resolution for strict, permissive, and namespace-vs-workload precedence; unavailable-evidence boundaries for port-level override, DestinationRule contradiction, injection-disabled membership, and unlabeled/unclassified behavior. M4 does not wire deferred producers.
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
- Move the fixture orchestration from shell into a Go table harness if M5 growth makes the current bounded script materially harder to extend or diagnose.

## Summary

### Decisions

- `versions.yaml` remains the sole source for Kind v0.31.0, its digest-pinned Kubernetes 1.35.0 node image, Istio 1.30.2, and every supported-platform download checksum. Downloads use retries, connection/total timeouts, and SHA-256 verification; cached istioctl downloads are re-extracted from the verified archive before execution.
- `make e2e` always rebuilds the scanner. The harness accepts relative state-directory overrides, resets all fixture namespaces concurrently, centralizes the case table in `cases.tsv`, and creates token kubeconfigs as 0600 files in a random 0700 directory removed on exit.
- STRICT, PERMISSIVE, and namespace-STRICT/workload-DISABLE precedence are resolved live. The port, DestinationRule, and injection-disabled fixtures verify their live inputs but intentionally golden the scanner's current unavailable/unknown evidence, preserving the M3 and M6 deferrals.
- The static RBAC proof enumerates every published manifest and exact SPEC §13 rule, rejects extra YAML documents/files, aggregation, wildcards, subresources, non-resource URLs, resource names, and any verb beyond `get`/`list`. The live proof also compares the referenced Role/ClusterRole proof shape to the published manifest before scanning and detects direct, User, and service-account-group bindings.
- Kind kube-apiserver audit logging runs in `blocking-strict` mode. A separate unbound audit-probe identity must produce a recorded denied ConfigMap create before scanners run; scanner events then pass a strict verb, API group, resource, and no-subresource allowlist.
- Eight small reports are golden-compared and a ninth all-namespaces report exercises the ClusterRole across namespaces and global workload ordering. Checked-in goldens are schema-tested outside the optional Kind job; raw generated fields and semantic/finding guards must pass before update mode can copy.
- CI owns failure diagnostics collection before its always-run teardown. Local failures retain the cluster for inspection. The separate PR workflow remains non-required, uses repository Go version metadata, a 30-minute timeout, concurrency cancellation, and failure artifacts.

### Flags raised

- No frozen contract under `docs/contracts/` and no exported resolver/output shape changed.
- No RBAC API group, resource, or verb outside SPEC §13 was added. Published profiles contain no Secrets, `watch`, token creation, impersonation, subresource, or write grant.
- M3's workload-port and DestinationRule producers remain unwired, and injection-disabled membership remains unknown pending M6. Review requests to make those fixtures fully resolvable were not implemented because they conflict with the explicit M4 scope boundary.
- The pinned version matrix unblocks M2's deferred version-specific root-namespace selector analysis. That behavior was not implemented in M4.
- Kubernetes automatically binds authenticated/service-account identities to fixed built-in discovery roles. The harness excludes only those named defaults while rejecting any additional resource-authorizing binding; the scanner audit remains the proof that the scanner itself issued only approved reads.
- The review round exposed and fixed clean-build/stale-binary execution, world-readable persistent tokens, multi-document/wildcard/aggregation RBAC gaps, group binding gaps, vacuous audit/schema/golden guards, missing cluster-scan/conflict coverage, relative state paths, destructive diagnostic overwrite, unchecked downloads, and stale closeout evidence.
- A failed local E2E run now keeps its cluster for investigation; CI has the single collect-then-delete owner. This is intentional and documented.
- Structured autoreview was attempted with `autoreview --mode local --thinking xhigh --stream-engine-output`. The sandboxed run could not initialize the read-only Codex state database, and policy rejected exporting the local bundle when retried outside the sandbox. No workaround was used; the supplied adversarial reviews plus local static/live verification drove the closeout.

### Verification

- Final clean sequence from no Kind clusters: `make kind-up` 46s; first ordinary `make e2e` 37s; second ordinary `make e2e` 37s; `make kind-down` 0s.
- Exact versions: Kind v0.31.0; `kindest/node:v1.35.0@sha256:452d707d4862f52530247495d180205e029056831160e22870e37e3f6c1ac31f`; Istio 1.30.2. All eight upstream Kind/istioctl platform checksums were pinned from their official release assets.
- Both ordinary E2E runs matched all eight goldens and schema-validated nine reports. The final audit contained 71 cluster-scanner list events, nine namespace-scanner list events, and one separate denied audit-probe create event.
- The namespace Role report contained the expected Namespace and root-PeerAuthentication denials plus three unknown findings. The workload-conflict report retained the workload policy chain and open critical MG-MTLS-003 finding.
- Final `make build test lint schema-test`: green; lint reported zero issues. `go test -race ./deploy/rbac ./internal/output`, shell syntax, actionlint, relative-state-path regression, and `git diff --check`: green.
- No token-bearing kubeconfig remained after the final runs, and no Kind cluster remained after teardown.
