# M3 — CEL Rule Engine and Built-in Control Packs

Branch: `m3-rule-engine`

## Goal
Controls become data. The engine loads packs, validates them per the contract, evaluates CEL against resolver outputs + inventory, and produces findings — including mechanical unknown/not-applicable handling.

## Context
Contract: docs/contracts/control-format.md (authoritative, including its Semantics and Validation sections). SPEC.md §12, §15, §17.

## Deliverables
- [ ] Pack loader: embedded built-ins (go:embed) + `--control-pack` (repeatable); duplicate-ID rejection across packs.
- [ ] Pack validation exactly per contract §Validation, with file/ID/CEL-position error messages; expose as `openmeshguard controls validate`.
- [ ] CEL environments per scope (workload/namespace/resource) exposing exactly the contract's variables; compile-time rejection of out-of-scope variable references.
- [ ] `requires` mechanism: dotted-path availability check ⇒ `unknown` finding with unknownReason, CEL never evaluated.
- [ ] `applicability` ⇒ `not-applicable`, excluded from pass rates.
- [ ] Finding assembly: deterministic IDs, severity, message templating, resolution chain attachment from resolver output, evidenceSources.
- [ ] Built-in packs (initial, config-type only): MG-MTLS-001/002/003, MG-AUTHZ placeholders SKIPPED (resolver lands M5) — instead ship MG-GW-001, MG-EGRESS-001, MG-VER-002 if inventory supports it; keep the pack honest about what's evaluable now.
- [ ] Replace the M1 provisional finding path entirely.
- [ ] Score summary v1: category pass rates + letter grades per SPEC §16 (numeric weighting can be a follow-up; grades required now).

## Definition of Done
- `make test lint schema-test` green; engine unit tests cover pass/fail/unknown/not-applicable/exception-absent paths and every validation rejection case.
- A deliberately malformed pack in testdata produces the contract-specified errors.
- End-to-end: scan of the M1 manual setup now produces engine-generated findings validating against the schema.

## Out of scope
Exception matching (M6 with context inputs), runtime controls (M7), authz controls (M5).
