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
- Wire the scan-config file, parameter/severity overrides, classification, and environment baselines with the M6 governance-context work. M3 keeps the engine input hook but does not invent the config format or CLI surface ahead of its owning milestone.
- Produce normalized resource-scope inputs in the milestone that collects Gateway/ServiceEntry resources. Until that producer exists, `scan` rejects resource-scoped packs explicitly instead of silently evaluating zero targets.
- Add MG-GW-001, MG-EGRESS-001, and MG-VER-002 when normalized Gateway, ServiceEntry, and control-plane-version inventory exists; M3 does not invent unsupported resource views.
- Apply production-only environment baselines when M6 supplies resolved classification. The initial mTLS pack evaluates all classified or unclassified workloads so the M1 end-to-end path remains evaluable before M6.
- Add numeric category weighting and an overall numeric score after the v1 weighting data is defined; M3 ships category pass rates and grades only.

## Summary

### Decisions
- Built-ins and validation use one `.yaml` pack pattern end to end: `controls/embed.go` and the disk validation test intentionally match the same extension set. Remediation templates are embedded separately under `controls/templates/*.tmpl` and resolved while the pack is loaded.
- Evidence used by `applicability` is checked before applicability CEL executes. A known `not-in-mesh` workload therefore resolves `not-applicable`, while an unknown data-plane mode or other unavailable applicability input resolves `unknown` and cannot escape scoring as not-applicable.
- Every control must declare at least one dotted `requires` path, and every non-structural leaf path referenced by its expression must be declared exactly. Dependencies come from the checked CEL AST, so dot access, literal bracket access, macros, and strings are distinguished correctly; a parent map cannot hide an unavailable child.
- CEL uses dynamic contract objects with scope-specific root variables plus the strings extension. CEL reserves `namespace`, so the compiler transparently rewrites only that root token to an equal-length internal alias; user pack syntax and reported CEL positions remain those of the frozen contract.
- Pack parameters merge through the engine's explicit input hook; the scan-config file and its overrides remain owned by M6. Pass rates count only pass/fail evaluations; `unknown` is reported separately and `not-applicable` is excluded. Grades use A ≥ 90%, B ≥ 80%, C ≥ 70%, D ≥ 60%, otherwise F; categories with no pass/fail evaluations are `unknown`.
- The M1 provisional finding function was deleted rather than wrapped. Engine-generated output now covers every `MTLSEffective` value through `requires` and `applicability`; end-to-end tables prove mixed-by-port produces explicit high findings, not-in-mesh produces only not-applicable findings, and confirmed disabled posture retains the critical MG-MTLS-003 finding.
- The initial pack ships MG-MTLS-001/002/003 only. Authz controls wait for M5, and Gateway/egress/version controls were not shipped because current normalized inventory cannot evaluate them honestly.
- MG-MTLS-003 is deliberately the independent critical check for a known `mtls.effective == disabled` result. Per-port plaintext remains MG-MTLS-002; DestinationRule contradiction detection is not folded into MG-MTLS-003 until its missing producer is assigned, so unavailable secondary evidence cannot downgrade a confirmed critical result.
- Namespace targets come from the complete collected namespace set, including namespaces with no workload. Resource-scoped custom packs are rejected by the real scan path until normalized resource production exists; the standalone engine continues to support and test resource scope.
- Resolution chains are selected from AST-derived requires/applicability/expression dependencies and are globally renumbered when mTLS and authorization evidence is combined. The CEL workload object now carries every field in the frozen resolver result.

### Flags raised
- DestinationRule collection remains unassigned and was not added in M3; likely ownership is M5. `workload.mtls.clientTLSContradiction` remains unavailable, and any custom control requiring it is verified to become `unknown`. No built-in currently claims that contradiction is evaluable.
- Workload ports likewise have no real-scan producer yet. MG-MTLS-002 remains `unknown` through `requires` rather than assuming an empty port map.
- Scan-config parameters and severity/environment overrides were not added because the config/classification surface is owned by M6 and no frozen scan-config format exists yet. The prior summary claim that the CLI already supplied overrides was removed.
- Real-scan resource inputs remain unimplemented with their resource collectors. M3 now fails those packs explicitly instead of silently producing no targets.
- No frozen contract files or exported resolver/output JSON shapes were changed. No contract change is proposed by M3.

### Verification
- `make build`, `make test`, `make lint`, and `make schema-test` pass.
- The generated-output schema test covers engine-generated IDs, `unknown` plus `unknownReason`, `not-applicable`, and a real mTLS category grade/pass rate.
- `internal/engine/testdata/malformed.yaml` exercises the human review gate with file-, control-, and CEL-position diagnostics; the validation table also covers every rejection required by the contract.
