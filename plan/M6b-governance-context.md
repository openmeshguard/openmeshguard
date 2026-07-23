# M6b — Governance Context (classification, ownership, exceptions)

Branch: `m6b-governance-context`

## Goal
Governance inputs land as first-class, unknown-first context: environment classification, ownership, and exception records — enabling environment-scoped control evaluation without ever silently suppressing a real finding.

## Context
SPEC.md §9 (classification + ownership precedence), §10 (exceptions), §15 (context controls). Canonical schema `workloadPostures[].environment`/`environmentConfidence`/`owner`, `findings[].status` (`excepted`), `findings[].exception`, and `inventory.classification` are authoritative. Builds on M6a (ambient) being merged.

## Deliverables
- [ ] Classification per SPEC §9 precedence: scan-config mapping → `openmeshguard.io/environment` label → fallback labels → `--infer-environments` heuristics → unclassified. Heuristics are OFF by default, produce `inferred` confidence, and are disclosed in the report. Unclassified is explicit, never a default. MG-ENV-001.
- [ ] Ownership per SPEC §9 precedence: labels/annotations → scan-config → import file. MG-OWN-001/002.
- [ ] Exception records + annotation matching per SPEC §10: a matched finding becomes `status: excepted` and is **never removed** from output; an expired exception restores the original severity and raises MG-EXC-002; MG-EXC-001 validates exception record fields. Exceptions are engine-applied after evaluation — controls never reference them.
- [ ] Environment-scoped control evaluation activates: production-only controls (e.g. MG-MTLS-001) evaluate only classified-production workloads; unclassified is covered by MG-ENV-001, not by silent pass.
- [ ] scan-config file format for classification/ownership/exception inputs. If this needs a new frozen-contract surface, STOP and propose it for human approval before writing.
- [ ] Context controls ship as YAML/CEL data; e2e fixtures exercise classified/unclassified, owned/unowned, and active/expired-exception cases with goldens; RBAC proofs + determinism re-run.

## Definition of Done
- Classification, ownership, and exception state all carry explicit unknowns; no path silently passes, fails, or drops a finding.
- An excepted finding is present-but-marked in output; an expired exception is visibly restored with MG-EXC-002; proven by fixture goldens.
- `--infer-environments` stays opt-in and its confidence is disclosed. `make build test lint schema-test` + e2e green.

## Human review gate
**Exception matching** — a mis-scoped exception silently suppresses a real finding, so the matching rules and the never-removed / expired-restored behavior are the highest-stakes semantics in this milestone. Review the exception fixture goldens deliberately.

## Out of scope
Ambient (M6a), HTML/SARIF/score/exit-codes (M6c), Prometheus (M7).

## Deferred
(record follow-ups here)
