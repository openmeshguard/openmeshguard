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
- [x] **RBAC proof test**: e2e run executes the scan under a ServiceAccount whose only resource-authorizing binding is the published ClusterRole; pinned Kubernetes non-resource `get` defaults are isolated and verified separately. A second run under the namespace Role verifies degradation to permissionSummary entries + unknown findings instead of failure.
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
- `make e2e` always rebuilds the configured scanner binary and passes its absolute path to the harness, including when `BINARY` is overridden. The harness accepts relative state-directory overrides, resets all fixture namespaces concurrently, centralizes the case table in `cases.tsv`, and creates token kubeconfigs as 0600 files in a random 0700 directory removed on exit.
- STRICT, PERMISSIVE, and namespace-STRICT/workload-DISABLE precedence are resolved live. The port, DestinationRule, and injection-disabled fixtures verify their live inputs but intentionally golden the scanner's current unavailable/unknown evidence, preserving the M3 and M6 deferrals.
- The static RBAC proof enumerates every published manifest and exact SPEC §13 rule, rejects extra YAML documents/files, aggregation, wildcards, subresources, non-resource URLs, resource names, and any verb beyond `get`/`list`. The live proof also compares the referenced Role/ClusterRole proof shape to the published manifest before scanning and detects direct, User, and service-account-group bindings. The disposable cluster removes `system:basic-user`; the remaining authenticated-group exceptions are live-verified as non-resource `get`-only discovery roles.
- Kind kube-apiserver audit logging runs in `blocking-strict` mode. Kind writes its administrator credential only to a protected harness kubeconfig, never the user's default kubeconfig. Successful fixture-manager and Kind-admin reads prove privileged requests are recorded before both credentials are removed. Separate cluster- and namespace-scanner proof phases ensure only the active scanner kubeconfig exists; privileged setup between phases is excluded by a second truncation. A separate unbound audit-probe must produce a recorded denied ConfigMap create; combined scanner events pass a strict list-only API group, resource, and no-subresource allowlist, and any post-boundary fixture-manager or Kind-admin event fails the proof.
- Scanner children run with an empty environment, a fresh empty `HOME`, and only their explicit scanner kubeconfig. ServiceAccount tokens are passed to `jq` by protected temporary file rather than argv and deleted immediately after kubeconfig generation. Harness administrator credentials are removed on every E2E exit and Kind teardown; failure diagnostics use a separate temporary credential.
- Eight small reports are golden-compared and a ninth all-namespaces report exercises the ClusterRole across namespaces and global workload ordering. Checked-in goldens are schema-tested outside the optional Kind job; an exact case/golden bijection, raw generated fields, exact expected control/status sets, and non-empty resolved posture/finding chains must pass before update mode can copy.
- CI owns failure diagnostics collection before its always-run teardown. Local failures retain the cluster for inspection. The separate PR workflow remains non-required, uses repository Go version metadata, a 30-minute timeout, concurrency cancellation, and failure artifacts.

### Flags raised

- No frozen contract under `docs/contracts/` and no exported resolver/output shape changed.
- No RBAC API group, resource, or verb outside SPEC §13 was added. Published profiles contain no Secrets, `watch`, token creation, impersonation, subresource, or write grant.
- M3's workload-port and DestinationRule producers remain unwired, and injection-disabled membership remains unknown pending M6. Review requests to make those fixtures fully resolvable were not implemented because they conflict with the explicit M4 scope boundary.
- The pinned version matrix unblocks M2's deferred version-specific root-namespace selector analysis. That behavior was not implemented in M4.
- Kubernetes automatically binds authenticated/service-account identities to built-in roles. The review found that `system:basic-user` includes `create` on self-review resources, so `kind-up` now removes that binding and E2E requires it to remain absent. Only three live-verified non-resource `get` discovery roles are excepted; the scanner audit remains the independent proof that the scanner issued only approved reads.
- The review round exposed and fixed clean-build/stale-binary execution, world-readable persistent tokens, multi-document/wildcard/aggregation RBAC gaps, group binding gaps, vacuous audit/schema/golden guards, missing cluster-scan/conflict coverage, relative state paths, destructive diagnostic overwrite, unchecked downloads, and stale closeout evidence.
- A failed local E2E run now keeps its cluster for investigation; CI has the single collect-then-delete owner. This is intentional and documented.
- The first hosted E2E run exposed a Linux-only fallback bug: POSIX shell function assignments are process-global, so checksum verification overwrote the requested Kind version and cleanup then treated the checksum as a download version. Function-specific variable names now isolate every download helper, and `make test` forces both Kind and istioctl fallback paths with synthetic local artifacts.
- A later review exposed an unsafe Kubernetes-default RBAC exception, incomplete golden-update semantics, and unquoted Kind host paths. The proof cluster now removes `system:basic-user`, verifies every remaining default exception as non-resource `get` only, requires exact finding sets plus non-empty resolved chains before any golden copy, and round-trips special-character paths through parsed YAML.
- The final security review exposed an alternate-credential audit blind spot, bearer-token argv exposure, overridden-binary drift, an overly broad named-object GET allowance, and the lack of a case/golden bijection. The harness now positively proves and then excludes both privileged credential classes, isolates each scanner process, consumes tokens through protected files, propagates the built binary path, accepts scanner `list` calls only, and rejects stale or missing goldens.
- Structured autoreview was attempted with `autoreview --mode local --thinking xhigh --stream-engine-output`. The sandboxed run could not initialize the read-only Codex state database, and policy rejected exporting the local bundle when retried outside the sandbox. No workaround was used; the supplied adversarial reviews plus local static/live verification drove the closeout.

### Verification

- Exact final-branch `make kind-up e2e kind-down` from no Kind clusters: 45s, 25s, and 0s. A second fresh cluster took 41s to set up, then produced consecutive E2E runs in 25s and 37s; the first used `BINARY=/tmp/openmeshguard-m4-determinism/openmeshguard`, the second used the default binary, and all eight normalized report SHA-256 values were identical. Teardown took 0s.
- Exact versions: Kind v0.31.0; `kindest/node:v1.35.0@sha256:452d707d4862f52530247495d180205e029056831160e22870e37e3f6c1ac31f`; Istio 1.30.2. All eight upstream Kind/istioctl platform checksums were pinned from their official release assets.
- Both E2E runs matched all eight goldens and schema-validated nine reports. Each final audit contained 71 cluster-scanner list events, nine namespace-scanner list events, one separate denied audit-probe create event, and zero fixture-manager or `kubernetes-admin` events after the proof boundary.
- The namespace Role report contained the expected Namespace and root-PeerAuthentication denials plus three unknown findings. The workload-conflict report retained the workload policy chain and open critical MG-MTLS-003 finding.
- The live binding proof required `system:basic-user` to remain absent and verified all three allowed Kubernetes-default roles as non-resource `get` only before either scanner ran.
- Final `make build test lint schema-test`: green; lint reported zero issues. Synthetic audit mutations for scanner GET/write, fixture-manager write, and Kind-admin write are rejected. The custom-binary propagation, argv-free token contract, scanner environment isolation, stale/missing golden bijection, Kind/istioctl fallback, missing-finding/missing-chain, and special-character Kind path regressions all pass. `go test -race ./deploy/rbac ./internal/output ./test/e2e`, shell syntax, actionlint, relative/special-character-state-path regression, and `git diff --check`: green.
- No token-bearing scanner, fixture-manager, diagnostic, or Kind-administrator kubeconfig remained after the final runs, and no Kind cluster remained after teardown.
