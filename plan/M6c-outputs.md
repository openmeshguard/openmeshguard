# M6c — Consumable Outputs (HTML, SARIF, score, CI exit codes)

Branch: `m6c-outputs`

## Goal
The scan becomes consumable by humans and CI: a static HTML report, SARIF export, a `score` command, and a CI exit-code contract — all projections of the canonical JSON, which remains the single internal model. **This milestone earns the v0 release.**

## Context
SPEC.md §6 (report design), §12 (SARIF stance), §16 (scoring), §17. Canonical schema is authoritative for everything rendered — outputs project from it, never diverge. Builds on M6a + M6b merged.

## Deliverables
- [ ] Static, self-contained, server-less HTML report rendered from canonical JSON: Declared/Verified/Unknown summary table (SPEC §6), category grades, permission/evidence summary, findings with expandable resolution chains, classification-coverage metric. Single file, no external assets, no network.
- [ ] SARIF 2.1.0 export: findings→results, controls→rules, severity mapping, resource locations. Explicitly a projection — the internal model stays canonical JSON. Validated against the SARIF schema in tests.
- [ ] `score` command per SPEC §16: numeric weighted score + category grades + critical-cap rule, reading canonical JSON.
- [ ] CI exit codes: 0 clean, 1 findings ≥ `--fail-on <sev>`, 2 scan error. **Unknowns never affect exit code by default**; `--fail-on-unknown` is opt-in.

## Definition of Done
- **First-run zero-config scan produces the SPEC §6-shaped summary with honest unknowns** — this is the v0 release criterion.
- HTML renders from a golden JSON in a headless check (well-formedness + key sections present); SARIF validates against its schema in tests.
- All fixtures (sidecar-basic, sidecar-authz, ambient-basic, mixed, governance) green in e2e including RBAC proofs; determinism re-run.
- `make build test lint schema-test` + e2e green.

## Human review gate
Exit-code contract and SARIF projection fidelity — these are the CI-integration and code-scanning surfaces users wire into pipelines; a wrong exit code or a dropped finding in SARIF is a silent trust failure.

## Out of scope
Prometheus / runtime verification (M7), scan --local, drift, multi-cluster evaluation. Release engineering is M6.5 (see plan/M6.5-release.md), triggered on this milestone's completion.

## Deferred
(record follow-ups here)
