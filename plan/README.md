# Milestone Plan

One milestone per branch/session, in order. Each file's **Definition of Done is the only exit criterion** — do not expand scope; record follow-ups under "Deferred" in the task file.

| Milestone | Delivers | Cluster needed? |
| --- | --- | --- |
| M0 | Scaffold, CI, contracts enforced | No |
| M1 | Walking skeleton: read-only scan → canonical JSON, one provisional finding | Manual Kind check |
| M2 | Effective mTLS resolver, full semantics, table-driven | No |
| M3 | CEL rule engine + built-in control packs, grades | No |
| M4 | Kind acceptance harness, golden findings, RBAC/no-write proofs | Yes (e2e) |
| M5 | Effective authorization resolver + authz controls | e2e fixture |
| M6a | Ambient posture first-class (detection, ztunnel, waypoints) | e2e fixtures |
| M6b | Governance context: classification, ownership, exceptions | e2e fixtures |
| M6c | Consumable outputs: HTML report, SARIF, score, CI exit codes — **earns v0** | e2e fixtures |
| M6.5 | v0 release engineering (goreleaser, signed binaries, docs) — tag `v0.1.0` | No |
| M7 | Prometheus runtime verification (MG-MTLS-101/102) | e2e best-effort |

M6 was split into M6a/M6b/M6c (ambient / governance context / outputs) so each ships as a
reviewable branch; they run in order. M6c completing is the v0 release criterion, after which
M6.5 cuts `v0.1.0`.

Human review gates (do not merge past these without maintainer sign-off):
- M0: contract placement and CI shape.
- M2 and M5: **expected outputs of the resolver test tables** — this is the product's correctness core.
- M3: control-format validation behavior.
- M4: RBAC manifests vs SPEC §13.
- M6a: ambient detection semantics vs Istio docs.
- M6b: **exception matching** — a mis-scoped exception silently suppresses a real finding.
- M6c: CI exit-code contract and SARIF projection fidelity.
