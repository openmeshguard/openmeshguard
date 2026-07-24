# M6a — Ambient Posture (first-class)

Branch: `m6a-ambient`

## Goal
Ambient becomes a real, detected data-plane mode end to end — replacing the M1 `ambientDetectionStub` — so the ambient L4/L7 posture the M5 authorization resolver was built to model actually receives live inputs during a scan.

## Context
SPEC.md §7 (ambient posture), §15 (ambient controls). Canonical schema `inventory.dataPlane` (mode, ztunnel, waypoints) and `workloadPostures[].dataPlaneMode` are authoritative. Read the M5 Deferred/Summary notes on ambient attachment: M5 modeled enforced/unenforced/unavailable L7 states in resolver tables but never received a live ambient input — this milestone provides it.

## Deliverables
- [x] Replace `ambientDetectionStub` with real detection: `istio.io/dataplane-mode` namespace/pod enrollment labels, ztunnel DaemonSet discovery, waypoint inventory + readiness.
- [x] ztunnel node coverage: `nodesCovered` / `nodesTotal`, with `nodesTotal` null when the nodes permission is absent (optional add-on RBAC). Never guess coverage from missing data.
- [x] Mixed-mode detection feeds `dataPlaneMode` everywhere (inventory + per-workload); sidecar and ambient workloads in one mesh resolve correctly.
- [x] Waypoint discovery (`istio.io/use-waypoint` labels, Gateway API waypoints) wired into `WorkloadInput.Waypoint` so the M5 resolver's `l7-policy-unenforced` / enforced / unavailable states occur in a real scan, not only in unit tables.
- [x] Ambient controls MG-MTLS-005 (ambient namespaces have validated L4 mTLS posture) and MG-MTLS-006 (healthy ztunnel coverage on every scheduled node); MG-GW-005 as inventory supports it. Ship as YAML/CEL data.
- [x] Collectors for any new resource (nodes via optional add-on, ztunnel/waypoint resources) are typed, bounded, get/list only, no secrets/watch; the action-audit test is extended to cover them.
- [x] Kind `ambient-basic/` fixture + a mixed-mode fixture with goldens; pin ambient-capable Istio in `versions.yaml`; e2e green including both RBAC proofs.

## Definition of Done
- Ambient and mixed-mode fixtures green in e2e; determinism double-run re-recorded.
- A real ambient scan produces the enforced / unenforced / unavailable L7 authorization states end to end (prove with a fixture whose golden shows an ambient L7 policy with and without a ready waypoint).
- Resolver purity green; action audit covers every new resource; `make build test lint schema-test` green.

## Human review gate
Ambient detection semantics — enrollment precedence, ztunnel-coverage unknowns, and waypoint readiness — vs. Istio docs. Cite upstream for any ambiguous edge and flag before encoding.

## Out of scope
Governance context (M6b), HTML/SARIF/score/exit-codes (M6c), Prometheus (M7). No ambient ztunnel/waypoint runtime health beyond declared readiness.

## Deferred

- Decide whether a namespace-scoped scan with broader credentials may follow an
  `istio.io/use-waypoint-namespace` label outside the requested namespace.
  M6a preserves the existing scope boundary and reports that evidence
  `unknown`; all-namespaces scans resolve cross-namespace waypoints.
- Add a multi-node Kind case for partial ztunnel coverage if the acceptance
  topology expands. The current one-node live proof and table-driven missing
  Ready-pod cases cover the required semantics without widening M6a.
- Project `AuthorizationPolicy` targetRefs for `ServiceEntry` workloads after
  the normalized posture inventory has a ServiceEntry target to attach them
  to. The collector already reads ServiceEntries, but inventing a Kubernetes
  workload attachment in M6a would misstate enforcement and widen the frozen
  posture model.
- Split a controller whose live replicas have heterogeneous data-plane
  enrollment into per-Pod posture targets in a later inventory milestone.
  M6a reports inventory-level `mixed` for healthy sidecar/ambient coexistence;
  an individual workload whose mode cannot be selected remains `unknown`.

## Summary

### Decisions

- Ambient enrollment follows Istio's documented selection order. An observed
  `istio-proxy` wins; otherwise a Pod `istio.io/dataplane-mode` value overrides
  the Namespace value, `ambient` opts in, `none` opts out, and unavailable or
  unsupported evidence remains `unknown`: [Istio ambient workload
  enrollment](https://istio.io/latest/docs/ambient/usage/add-workloads/#pod-selection-logic-for-ambient-and-sidecar-modes).
- ztunnel discovery uses typed, paginated list calls only. Candidates require
  the canonical `app=ztunnel` and `app.kubernetes.io/name=ztunnel` chart labels;
  a DaemonSet must repeat those identities in its selector and Pod template,
  and a ztunnel Pod must have a controller owner reference to that collected
  DaemonSet UID. Coverage counts unique nodes with a Ready owned ztunnel Pod,
  and each ambient workload receives `ZtunnelOnNode` from its observed
  scheduled Pod nodes. This follows Istio's ztunnel DaemonSet template:
  [ztunnel chart
  template](https://github.com/istio/istio/blob/master/manifests/charts/ztunnel/templates/daemonset.yaml).
- `nodesTotal` is emitted as JSON null unless the optional Node list succeeded.
  `nodesCovered` and per-workload ztunnel state are never derived from a denied
  ztunnel Pod/DaemonSet list. The existing
  `deploy/rbac/addons/nodes-cluster-role.yaml` is applied only by the acceptance
  harness; no published RBAC manifest changed.
- Waypoints remain typed Gateway API projections. Only the
  `istio-waypoint` GatewayClass is inventoried; the selected workload, Service,
  or Namespace `istio.io/use-waypoint` path is ready only when the Gateway has
  a current-generation `Programmed=True` condition and its
  `istio.io/waypoint-for` scope covers that selection:
  [configure waypoint
  proxies](https://istio.io/latest/docs/ambient/usage/waypoint/).
- The human-reviewed coexistence rule is per workload: an observed sidecar
  takes precedence and ignores ztunnel state; only ambient workloads consume
  `ZtunnelOnNode`. Ambient `true` proceeds to L4/L7 policy evaluation,
  `false` resolves authorization as not in the ambient mesh, and unavailable
  evidence remains `unknown`. This matches Istio's documented exclusive Pod
  selection between sidecar and ambient modes: [ambient and sidecar
  coexistence](https://istio.io/latest/docs/ambient/usage/add-workloads/#pod-selection-logic-for-ambient-and-sidecar-modes).
- Confirmed resolver semantics fixes moved the implementation to
  `mtls/v5,authz/v8`. No frozen resolver/output type or schema changed.
- MG-MTLS-005, MG-MTLS-006, and MG-GW-005 are YAML/CEL controls with explicit
  ambient applicability and pass/fail/unknown/not-applicable tables.
  `builtin-mtls` and `builtin-authz` metadata versions are both `0.2.0`.
- The Kind harness pins Istio 1.30.2's ambient profile and checksum-verifies the
  Gateway API v1.5.1 experimental CRD bundle required for waypoint proxies.
  The live ambient group proves ready, missing, and unavailable waypoint
  evidence; the mixed-mode group proves one ambient and one sidecar workload in
  the same namespace.

### Review findings

1. **Fixed — spoofable ztunnel evidence.** Reproduced with noncanonical
   DaemonSets and unowned Pods that were counted as coverage. The collector now
   authenticates canonical labels, selector/template identity, and controller
   ownership; `TestCollectorRejectsUnownedAndNonCanonicalZtunnelResources`
   fails on the old path and passes on the fix.
2. **Fixed with human sign-off — ambient authorization without ztunnel.**
   Reproduced by resolving an ambient workload with
   `ZtunnelOnNode=False/Unobserved`; both previously evaluated policy as though
   an ambient enforcement path existed. Authorization now gates on ztunnel for
   ambient workloads only, while an observed sidecar retains precedence. The
   behavior follows Istio's separate secure-overlay and waypoint layers:
   [ambient data
   plane](https://istio.io/latest/docs/ambient/architecture/data-plane/).
3. **Fixed — inverse client/server TLS contradiction.** Reproduced with server
   `PeerAuthentication` DISABLE and client `DestinationRule` ISTIO_MUTUAL; the
   contradiction was false. The mTLS resolver now models both strict and
   disabled server expectations and the regression passes.
4. **Fixed — ruleless waypoint DENY/CUSTOM.** Reproduced with targetRef
   policies that had no rules and were nevertheless reported as unenforced.
   Ruleless DENY, CUSTOM, and AUDIT policies no longer create waypoint
   enforcement dependencies; table regressions cover DENY and CUSTOM.
5. **Fixed — denied Namespace evidence collapsed inherited waypoint state.**
   Reproduced with unavailable Namespace labels and no explicit workload or
   Service waypoint selection. Normalization now preserves an unavailable
   Namespace waypoint view, so authorization remains `unknown`.
6. **Partly fixed, partly deferred — supported targetRefs.** Reproduced
   GatewayClass and ServiceEntry projection gaps. Root-namespace
   `GatewayClass/istio-waypoint` policies now project through the selected
   waypoint, consistent with Istio targetRef support:
   [AuthorizationPolicy
   targetRefs](https://istio.io/latest/docs/reference/config/security/authorization-policy/).
   ServiceEntry posture projection is recorded in Deferred because M6a has no
   normalized ServiceEntry workload target to receive the result.
7. **Fixed — stale waypoint readiness.** Reproduced with
   `Programmed=True` whose `observedGeneration` lagged the Gateway generation.
   Stale conditions no longer mark a waypoint ready.
8. **Fixed — PeerAuthentication permission loss masked known data-plane
   conclusions.** Reproduced for a known non-mesh workload and an ambient
   workload without ztunnel. Data-plane mTLS resolution now occurs before
   policy-evidence availability, preserving those known outcomes while other
   workloads still degrade to `unknown`.
9. **Fixed — CEL/output inventory-mode mismatch.** Reproduced with internal
   `not-applicable`, which the output canonicalized to `unknown` after CEL had
   evaluated controls against `not-applicable`. The CEL activation now uses
   the same canonical inventory mode as output.
10. **Fixed — golden guard accepted resolved findings without chains.**
    Reproduced by deleting resolved chains from a degraded golden; the guard
    accepted it. It now rejects every non-unknown finding without a chain, and
    the engine supplies an explicit control-applicability step where a
    not-applicable result has no resolver chain.

### Flags raised

- No file under `docs/contracts/`, no exported `internal/resolver` or
  `internal/output` type, and no file under `deploy/rbac/` changed.
- Namespace-scoped cross-namespace waypoint following remains the conservative
  `unknown` described in Deferred. M6a does not broaden a requested scan scope
  silently.
- Waypoint health is declared readiness only (`Programmed=True` plus attachment
  scope). Runtime dataplane traffic health remains out of scope until its
  owning milestone.
- The first live fixture run exposed that Istio 1.30.2's ztunnel DaemonSet
  metadata lacks `app=ztunnel` even though its selector and Pod template carry
  that label. The collector was corrected to filter the typed DaemonSet list by
  `spec.selector`; no expectation or golden was weakened.
- Istio does not install Gateway API CRDs by default. The harness now installs
  the checksum-pinned bundle documented by Istio before installing the ambient
  profile: [Istio Gateway API
  setup](https://istio.io/latest/docs/tasks/traffic-management/ingress/gateway-api/).

### Verification

- `make build`, `make test`, `make lint`, and `make schema-test` are green on
  the final tree. Lint reports zero issues and includes resolver depguard
  purity.
- Focused collector/normalizer/engine/output tables cover enrollment
  precedence, Ready and missing ztunnel coverage, denied optional Nodes,
  `nodesTotal:null`, all new typed actions, and every new control outcome.
- A guarded review-fix `UPDATE_GOLDEN=1 make e2e` completed in 62 seconds and
  wrote 17
  reviewed goldens; all 18 reports were schema-valid. The ready ambient golden
  records ztunnel coverage `1/1` and authorization `allow-only`; the matching L7
  policy without a ready waypoint records `waypoint-policy-unenforced`; the
  limited-evidence scan records known ztunnel coverage `1/1` and Gateway
  evidence `unknown`.
- A clean `make kind-up` completed in 44 seconds with Kind v0.31.0, digest-pinned
  Kubernetes 1.35.0, Istio 1.30.2 ambient, and Gateway API v1.5.1. Two
  consecutive non-update `make e2e` runs matched all 17 goldens, passed both
  published RBAC proofs plus the waypoint-limited proof, recorded the same 356
  approved list events and no other scanner calls, and completed in 64 and 61
  seconds.
- The final audit split was 316 cluster-scanner lists, 20 waypoint-limited
  lists, and 20 namespace-scanner lists, plus one separate denied audit-probe
  create. `make kind-down` removed the disposable cluster in 0 seconds.
- `git diff --check`, `git diff -- deploy/rbac`, and the frozen-contract diff
  checks are clean. No scanner credential directory remained after either
  consecutive run.
