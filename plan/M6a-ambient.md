# M6a — Ambient Posture (first-class)

Branch: `m6a-ambient`

## Goal
Ambient becomes a real, detected data-plane mode end to end — replacing the M1 `ambientDetectionStub` — so the ambient L4/L7 posture the M5 authorization resolver was built to model actually receives live inputs during a scan.

## Context
SPEC.md §7 (ambient posture), §15 (ambient controls). Canonical schema `inventory.dataPlane` (mode, ztunnel, waypoints) and `workloadPostures[].dataPlaneMode` are authoritative. Read the M5 Deferred/Summary notes on ambient attachment: M5 modeled enforced/unenforced/unavailable L7 states in resolver tables but never received a live ambient input — this milestone provides it.

## Deliverables
- [ ] Replace `ambientDetectionStub` with real detection: `istio.io/dataplane-mode` namespace/pod enrollment labels, ztunnel DaemonSet discovery, waypoint inventory + readiness.
- [ ] ztunnel node coverage: `nodesCovered` / `nodesTotal`, with `nodesTotal` null when the nodes permission is absent (optional add-on RBAC). Never guess coverage from missing data.
- [ ] Mixed-mode detection feeds `dataPlaneMode` everywhere (inventory + per-workload); sidecar and ambient workloads in one mesh resolve correctly.
- [ ] Waypoint discovery (`istio.io/use-waypoint` labels, Gateway API waypoints) wired into `WorkloadInput.Waypoint` so the M5 resolver's `l7-policy-unenforced` / enforced / unavailable states occur in a real scan, not only in unit tables.
- [ ] Ambient controls MG-MTLS-005 (ambient namespaces have validated L4 mTLS posture) and MG-MTLS-006 (healthy ztunnel coverage on every scheduled node); MG-GW-005 as inventory supports it. Ship as YAML/CEL data.
- [ ] Collectors for any new resource (nodes via optional add-on, ztunnel/waypoint resources) are typed, bounded, get/list only, no secrets/watch; the action-audit test is extended to cover them.
- [ ] Kind `ambient-basic/` fixture + a mixed-mode fixture with goldens; pin ambient-capable Istio in `versions.yaml`; e2e green including both RBAC proofs.

## Definition of Done
- Ambient and mixed-mode fixtures green in e2e; determinism double-run re-recorded.
- A real ambient scan produces the enforced / unenforced / unavailable L7 authorization states end to end (prove with a fixture whose golden shows an ambient L7 policy with and without a ready waypoint).
- Resolver purity green; action audit covers every new resource; `make build test lint schema-test` green.

## Human review gate
Ambient detection semantics — enrollment precedence, ztunnel-coverage unknowns, and waypoint readiness — vs. Istio docs. Cite upstream for any ambiguous edge and flag before encoding.

## Out of scope
Governance context (M6b), HTML/SARIF/score/exit-codes (M6c), Prometheus (M7). No ambient ztunnel/waypoint runtime health beyond declared readiness.

## Deferred
(record follow-ups here)
