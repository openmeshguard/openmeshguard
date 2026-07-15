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
- Define finding supersession or grouping with the scoring model so a globally disabled workload can retain MG-MTLS-003's critical guardrail without permanently double-counting its simultaneous MG-MTLS-001 baseline failure. M3 preserves both independent findings because dynamic severity and supersession are not part of the frozen control format.
- Wire the scan-config file, parameter/severity overrides, classification, and environment baselines with the M6 governance-context work. M3 keeps the engine input hook but does not invent the config format or CLI surface ahead of its owning milestone.
- Produce normalized resource-scope inputs in the milestone that collects Gateway/ServiceEntry resources. Until that producer exists, `scan` rejects resource-scoped packs explicitly instead of silently evaluating zero targets.
- Add MG-GW-001, MG-EGRESS-001, and MG-VER-002 when normalized Gateway, ServiceEntry, and control-plane-version inventory exists; M3 does not invent unsupported resource views.
- Apply production-only environment baselines when M6 supplies resolved classification. The initial mTLS pack evaluates all classified or unclassified workloads so the M1 end-to-end path remains evaluable before M6.
- Add numeric category weighting and an overall numeric score after the v1 weighting data is defined; M3 ships category pass rates and grades only.

## Summary

### Decisions
- Built-ins and validation use one `.yaml` pack pattern end to end: `controls/embed.go` and the disk validation test intentionally match the same extension set. Remediation templates are embedded separately under `controls/templates/*.tmpl` and resolved while the pack is loaded.
- Evidence used by `applicability` is checked before applicability CEL executes. A known `not-in-mesh` workload therefore resolves `not-applicable`, while an unknown data-plane mode or other unavailable applicability input resolves `unknown` and cannot escape scoring as not-applicable.
- Every control must declare at least one exact `requires` evidence path, and every non-structural path independently read by its expression must be declared exactly. Fixed fields use dotted segments; approved literal bracket segments preserve native map keys containing dots or slashes. Dependencies come from the checked CEL AST, so dot access, literal bracket access, macros, shadowed comprehension variables, and strings are distinguished correctly; a parent map cannot hide an unavailable child.
- CEL lexer errors are captured before the namespace compatibility rewrite, so invalid characters cannot be recovered into a different valid policy. A dynamic result is accepted as boolean only when the checked AST is a direct read of a contract field known to be boolean; embedding that field inside a string/list/index expression is rejected at pack load.
- CEL uses dynamic contract objects with scope-specific root variables plus the strings extension. CEL reserves `namespace`, so the compiler transparently rewrites only that root token to an equal-length internal alias; user pack syntax and reported CEL positions remain those of the frozen contract.
- Pack parameters merge through the engine's explicit input hook; the scan-config file and its overrides remain owned by M6. Pass rates count only pass/fail evaluations; `unknown` is reported separately and `not-applicable` is excluded. Grades use A ≥ 90%, B ≥ 80%, C ≥ 70%, D ≥ 60%, otherwise F; categories with no pass/fail evaluations are `unknown`.
- The M1 provisional finding function was deleted rather than wrapped. Engine-generated output now covers every `MTLSEffective` value through `requires` and `applicability`; end-to-end tables prove mixed-by-port produces explicit high findings, not-in-mesh produces only not-applicable findings, and confirmed disabled posture retains the critical MG-MTLS-003 finding.
- The initial pack ships MG-MTLS-001/002/003 only. Authz controls wait for M5, and Gateway/egress/version controls were not shipped because current normalized inventory cannot evaluate them honestly.
- MG-MTLS-003 is deliberately the independent critical check for a known `mtls.effective == disabled` result. Per-port plaintext remains MG-MTLS-002; DestinationRule contradiction detection is not folded into MG-MTLS-003 until its missing producer is assigned, so unavailable secondary evidence cannot downgrade a confirmed critical result.
- MG-MTLS-001/002/003 use the exact NIST CSF 2.0 PR.DS-02 data-in-transit outcome and OWASP Kubernetes Top Ten 2025 K05 related-risk tags. Framework values are traceability tags, never compliance claims; the obsolete OWASP K01 workload-hardening mapping was removed. A globally disabled workload intentionally raises both the strict-baseline MG-MTLS-001 finding and the critical MG-MTLS-003 guardrail until the deferred scoring model defines supersession or grouping.
- Namespace targets come from the complete collected namespace set, including namespaces with no workload. Resource-scoped custom packs are rejected by the real scan path until normalized resource production exists; the standalone engine continues to support and test resource scope.
- Resolution chains are selected from AST-derived requires/applicability/expression dependencies and are globally renumbered when mTLS and authorization evidence is combined. The CEL workload object now carries every field in the frozen resolver result.
- Live-scan inventory availability is derived from each denied Kubernetes/Istio list permission and merged into workload, namespace, and resource targets. Incomplete counts, aggregate data-plane mode, and multi-cluster signals therefore become `unknown` through `requires` instead of evaluating normalized zero/false defaults.
- Permission `affectedControls` values are derived after pack loading from each control's AST-backed dependencies and target scope. The collector no longer embeds built-in IDs, so degraded reports include relevant custom controls and cannot drift when the built-in catalog changes.
- CEL dependency analysis rejects iteration or other dynamic access over root contract maps because it cannot be represented by an exact dotted `requires` path. Resolution-chain steps are projected to canonical map values for CEL, while the literal string `unknown` is treated as unavailable only on contract fields whose enums define it as a sentinel; labels, parameters, and resource-native strings remain ordinary known values.
- User remediation template paths are opened through an OS-enforced rooted filesystem beneath the control-pack directory, so symlinks cannot escape and path validation cannot race the file open. Message and remediation templates validate static selectors against the supported template shape during pack load, while dynamic `Params` and `Inventory` keys remain permitted. Suggested YAML is rendered and emitted only for an open finding; unavailable parameters cannot abort unknown or not-applicable results.
- Template selector validation tracks statically typed local variables, including selectors such as `{{$p := .Posture}}{{$p.Mtls.Effective}}`; a misspelled variable field is rejected during pack validation rather than aborting a later scan.
- Namespace mesh enrollment preserves namespace-label evidence and aggregates both namespace enrollment hints and the resolver's observed sidecar/ambient/mixed/not-in-mesh modes without order-dependent last-write behavior. An unobserved workload cannot overwrite a conclusive workload or namespace-label observation with `unknown`.
- Namespace-scope controls receive only known enrolled namespaces plus namespaces whose enrollment is unknown; namespaces conclusively outside the mesh are excluded. A denied Pod list is treated as possible workload-target loss as well as missing data-plane evidence, so every loaded workload control is named in that permission's `affectedControls`.
- Resource controls deliberately remain source-native rather than translating Istio traffic APIs into Kubernetes Gateway API or an OpenMeshGuard-specific common model. `match.apiGroups` and `match.kinds` are both required; values within a list are ORed and the two dimensions are ANDed. Equivalent objectives with different source schemas use distinct control IDs, native CEL paths, remediation, and parity tests.
- Finding evidence sources are target-specific: Kubernetes Gateway resources identify `gateway-api`, source-native Istio resources identify `istio-crd`, and workload context-only controls do not claim Istio evidence they did not use. Resource finding IDs use API group rather than served version, so Kubernetes API version upgrades do not churn stable IDs. Optional resolution-step fields are absent from CEL maps when absent from the canonical result rather than appearing as known empty strings.
- Runtime findings identify `prometheus` only when verified telemetry reached an open evaluation; unknown and not-applicable results do not claim unavailable telemetry. The approved DestinationRule availability/unknown-propagation change advances resolver provenance from `mtls/v1` to `mtls/v2`.

### Flags raised
- DestinationRule collection remains unassigned and was not added in M3; likely ownership is M5. `resolver.WorkloadInput.DestinationRulesKnown` now distinguishes a collected empty set from unavailable evidence. The approved canonical output change makes `clientTLSContradiction` an optional boolean: real M3 scans omit it, while a future producer emits explicit `false` or `true`. Any custom control requiring the unavailable field is verified to become `unknown`; no built-in currently claims that contradiction is evaluable.
- Workload ports likewise have no real-scan producer yet. MG-MTLS-002 remains `unknown` through `requires` rather than assuming an empty port map.
- MG-MTLS-007 now owns future client/server TLS contradiction detection in `SPEC.md`; it is not shipped until DestinationRule collection exists. MG-MTLS-003 remains independently evaluable from server-side effective posture alone.
- M5's example composite resolver version must use the then-current mTLS tag (`mtls/v2` after M3), not the historical `mtls/v1` example. M3 does not edit the future milestone's implementation.
- Scan-config parameters and severity/environment overrides were not added because the config/classification surface is owned by M6 and no frozen scan-config format exists yet. The prior summary claim that the CLI already supplied overrides was removed.
- Real-scan resource inputs remain unimplemented with their resource collectors. M3 now fails those packs explicitly instead of silently producing no targets.
- The human explicitly approved three frozen-interface corrections during M3 review: native API-group matching, literal bracket keys in exact `requires` paths, and optional `clientTLSContradiction` output tied to explicit DestinationRule availability. `docs/contracts/control-format.md` documents the first two; the canonical schema and resolver JSON tags implement the third. No other frozen-interface change was made.

### Verification
- `make build`, `make test`, `make lint`, and `make schema-test` pass.
- The generated-output schema test covers engine-generated IDs, `unknown` plus `unknownReason`, `not-applicable`, and a real mTLS category grade/pass rate.
- `internal/engine/testdata/malformed.yaml` exercises the human review gate with file-, control-, and CEL-position diagnostics; the validation table also covers every rejection required by the contract.
- Additional adversarial validation fixtures prove lexer recovery cannot alter CEL, nested dynamic non-booleans cannot reach scan time, and selectors through typed template variables are checked during pack load.
- Resource tests prove API group and Kind are matched together, a same-Kind resource from another API is never evaluated, paired Gateway objectives retain parity, and findings preserve the native source API version.
- Generated-output tests prove unavailable DestinationRule evidence omits `clientTLSContradiction`, while completed collection emits an explicit boolean. Permission tests prove affected control IDs come from the built-in and user packs actually loaded for the scan.
