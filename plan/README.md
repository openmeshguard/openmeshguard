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
| M6 | Ambient posture, classification/ownership/exceptions, HTML report, SARIF, score | e2e fixtures |
| M7 | Prometheus runtime verification (MG-MTLS-101/102) | e2e best-effort |

Human review gates (do not merge past these without maintainer sign-off):
- M0: contract placement and CI shape.
- M2 and M5: **expected outputs of the resolver test tables** — this is the product's correctness core.
- M3: control-format validation behavior.
- M4: RBAC manifests vs SPEC §13.
