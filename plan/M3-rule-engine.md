# M3 — CEL Rule Engine and Built-in Control Packs

Branch: `m3-rule-engine`

## Goal
Controls become data. The engine loads packs, validates them per the contract, evaluates CEL against resolver outputs + inventory, and produces findings — including mechanical unknown/not-applicable handling.

## Context
Contract: docs/contracts/control-format.md (authoritative, including its Semantics and Validation sections). SPEC.md §12, §15, §17.

## Deliverables
- [x] Pack loader: embedded built-ins (go:embed) + `--control-pack` (repeatable); duplicate-ID rejection across packs.
- [x] Pack validation exactly per contract §Validation, with file/ID/CEL-position error messages; expose as `openmeshguard controls validate`.
- [x] CEL environments per scope (workload/namespace/resource) exposing exactly the contract's variables; compile-time rejection of out-of-scope variable references.
- [x] `requires` mechanism: dotted-path availability check ⇒ `unknown` finding with unknownReason, CEL never evaluated.
- [x] `applicability` ⇒ `not-applicable`, excluded from pass rates.
- [x] Finding assembly: deterministic IDs, severity, message templating, resolution chain attachment from resolver output, evidenceSources.
- [x] Built-in packs (initial, config-type only): MG-MTLS-001/002/003, MG-AUTHZ placeholders SKIPPED (resolver lands M5) — instead ship MG-GW-001, MG-EGRESS-001, MG-VER-002 if inventory supports it; keep the pack honest about what's evaluable now.
- [x] Replace the M1 provisional finding path entirely.
- [x] Score summary v1: category pass rates + letter grades per SPEC §16 (numeric weighting can be a follow-up; grades required now).

## Definition of Done
- `make test lint schema-test` green; engine unit tests cover pass/fail/unknown/not-applicable/exception-absent paths and every validation rejection case.
- A deliberately malformed pack in testdata produces the contract-specified errors.
- End-to-end: scan of the M1 manual setup now produces engine-generated findings validating against the schema.

## Out of scope
Exception matching (M6 with context inputs), runtime controls (M7), authz controls (M5).

## Deferred
- Assign and implement DestinationRule collection in a later milestone (likely M5). No current milestone owns it, so real scans deliberately mark `mtls.clientTLSContradiction` unavailable and controls that require it `unknown`.
- Wire normalized workload-port collection in a later milestone. M3 deliberately marks `mtls.byPort` unavailable in real scans until a producer can distinguish an observed empty port set from uncollected evidence.
- Add MG-GW-001, MG-EGRESS-001, and MG-VER-002 when normalized Gateway, ServiceEntry, and control-plane-version inventory exists; M3 does not invent unsupported resource views.
- Apply production-only environment baselines when M6 supplies resolved classification. The initial mTLS pack evaluates all classified or unclassified workloads so the M1 end-to-end path remains evaluable before M6.
- Add numeric category weighting and an overall numeric score after the v1 weighting data is defined; M3 ships category pass rates and grades only.

## Summary

### Decisions
- Built-ins and validation use one `.yaml` pattern end to end: `controls/embed.go` and the disk validation test intentionally match the same extension set.
- `applicability` runs before `requires`. This makes `not-in-mesh` workloads mechanically `not-applicable` even though port and DestinationRule evidence is absent, while every other unavailable required path becomes `unknown` before the control expression can run.
- Every control must declare at least one dotted `requires` path, and every non-structural path referenced by its expression must be covered by `requires`. This prevents a control from silently evaluating unavailable evidence as pass or fail.
- CEL uses dynamic contract objects with scope-specific root variables plus the strings extension. CEL reserves `namespace`, so the compiler transparently rewrites only that root token to an equal-length internal alias; user pack syntax and reported CEL positions remain those of the frozen contract.
- Scan-config parameters override same-named pack defaults. Pass rates count only pass/fail evaluations; `unknown` is reported separately and `not-applicable` is excluded. Grades use A ≥ 90%, B ≥ 80%, C ≥ 70%, D ≥ 60%, otherwise F; categories with no pass/fail evaluations are `unknown`.
- The M1 provisional finding function was deleted rather than wrapped. Engine-generated output now covers every `MTLSEffective` value through `requires` and `applicability`; mixed-by-port and not-in-mesh regression tables prove the two known provisional defects do not survive.
- The initial pack ships MG-MTLS-001/002/003 only. Authz controls wait for M5, and Gateway/egress/version controls were not shipped because current normalized inventory cannot evaluate them honestly.

### Flags raised
- DestinationRule collection remains unassigned and was not added in M3. Until an owner milestone supplies it, MG-MTLS-003 is `unknown` in real scans whenever it needs `clientTLSContradiction`; likely ownership is M5.
- Workload ports likewise have no real-scan producer yet. MG-MTLS-002 and MG-MTLS-003 remain `unknown` through `requires` rather than assuming an empty port map.
- No frozen contract files or exported resolver/output JSON shapes were changed. No contract change is proposed by M3.

### Verification
- `make build`, `make test`, `make lint`, and `make schema-test` pass.
- The generated-output schema test covers engine-generated IDs, `unknown` plus `unknownReason`, `not-applicable`, and a real mTLS category grade/pass rate.
- `internal/engine/testdata/malformed.yaml` exercises the human review gate with file-, control-, and CEL-position diagnostics; the validation table also covers every rejection required by the contract.
