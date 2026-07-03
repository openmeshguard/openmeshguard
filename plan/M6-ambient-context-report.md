# M6 — Ambient Posture, Governance Context, Report and SARIF

Branch: `m6-ambient-context-report`

## Goal
Ambient becomes first-class, governance context inputs land (classification/ownership/exceptions), and the scan becomes consumable: static HTML report, SARIF export, score command.

## Context
SPEC.md §6 (report design), §9, §10, §12 (SARIF stance), §15 (ambient + context controls), §16, §17. Canonical schema is authoritative for everything rendered.

## Deliverables

### Ambient
- [ ] Ambient detection: dataplane-mode labels/enrollment, ztunnel DaemonSet discovery + node coverage (nodes permission optional ⇒ nodesTotal null), waypoint inventory + readiness.
- [ ] Controls MG-MTLS-005/006, MG-GW-005; mixed-mode detection feeding dataPlaneMode everywhere.
- [ ] Kind ambient fixture (`test/fixtures/ambient-basic/`) + mixed-mode fixture, golden findings, e2e green. Pin ambient-capable Istio in versions.yaml.

### Governance context
- [ ] Classification per SPEC §9 precedence: scan-config mapping → openmeshguard.io/environment label → fallback labels → `--infer-environments` heuristics (off by default, inferred confidence, disclosed in report) → unclassified. MG-ENV-001.
- [ ] Ownership per SPEC §9 precedence (labels/annotations → config → import file). MG-OWN-001/002.
- [ ] Exception records + annotation matching per SPEC §10: findings become `excepted` (never removed), expired ⇒ severity restored + MG-EXC-002. MG-EXC-001 validation of record fields.
- [ ] Environment-scoped control evaluation now active (production-only controls evaluate only classified-production).

### Outputs
- [ ] Static HTML report from canonical JSON, no server: Declared/Verified/Unknown summary table per SPEC §6, category grades, permission/evidence summary, findings with resolution chains expandable, classification coverage metric. Single self-contained file.
- [ ] SARIF 2.1.0 export: findings→results, controls→rules, severity mapping, resource locations; explicitly a projection (schema remains internal model). Validate against SARIF schema in tests.
- [ ] `score` command per SPEC §16: numeric weighted score + category grades + critical-cap rule, reading canonical JSON.
- [ ] Exit codes for CI mode: 0 clean, 1 findings ≥ threshold (`--fail-on high`), 2 scan error; unknowns NEVER affect exit code by default, `--fail-on-unknown` opt-in.

## Definition of Done
- All fixtures (sidecar-basic, sidecar-authz, ambient-basic, mixed) green in e2e including RBAC proofs.
- HTML renders from a golden JSON in a headless check (well-formedness + key sections present).
- First-run zero-config scan produces the SPEC §6-shaped summary with honest unknowns.

## Out of scope
Prometheus (M7), scan --local, drift, multi-cluster evaluation.
