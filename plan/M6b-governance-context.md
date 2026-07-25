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

- Add cluster-, namespace-, selector-, and exact-resource exception scopes after
  real users demonstrate that annotation-only references are insufficient.
  M6b deliberately consumes enterprise exception decisions rather than
  implementing a second exception workflow.
- Add exception status/revocation, multiple-exception resolution, approval
  workflows, and direct ticket-system integrations after the owning external
  system contract is defined. In M6b, removing a Git-native record revokes it
  and Git preserves the audit history.

## Summary

### Expected golden delta before regeneration

- MG-MTLS-001 and MG-AUTHZ-001/002 become production-scoped. Existing fixture
  workloads have no governance classification, so those three controls must no
  longer emit open/unknown/not-applicable findings for them.
- Every mesh namespace without classification must instead receive
  MG-ENV-001, and every mesh workload without resolved ownership must receive
  MG-OWN-001. This replacement is the coverage guard: a production-only
  finding may disappear only when the same target is visibly covered by the
  environment governance finding.
- MG-OWN-002 evaluates only classified-production workloads and therefore must
  not appear in existing unclassified fixtures.
- MG-EXC-001/002 must appear only in the new exception fixtures. An active
  exception keeps its original finding with `status: excepted`; an expired
  exception keeps the original finding `open` at its original severity and
  adds MG-EXC-002.
- Built-in pack provenance changes from `builtin-mtls`/`builtin-authz` 0.2.0 to
  0.3.0 and adds `builtin-context` 0.3.0. Resolver provenance remains
  `mtls/v5,authz/v8`.
- The limited-permission ambient and namespace-Role goldens' ownership unknown
  reason names both namespace label and annotation evidence after the review
  fix that degrades application-ID and owner fields independently. Their
  control/status sets do not change.
