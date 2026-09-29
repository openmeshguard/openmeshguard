# M6c — Consumable Outputs (HTML, SARIF, score, CI exit codes)

Branch: `m6c-outputs`

## Goal
The scan becomes consumable by humans and CI: a static HTML report, SARIF export, a `score` command, and a CI exit-code contract — all projections of the canonical JSON, which remains the single internal model. **This milestone earns the v0 release.**

## Context
SPEC.md §6 (report design), §12 (SARIF stance), §16 (scoring), §17. Canonical schema is authoritative for everything rendered — outputs project from it, never diverge. Builds on M6a + M6b merged.

## Deliverables
- [x] Static, self-contained, server-less HTML report rendered from canonical JSON: Declared/Verified/Unknown summary table (SPEC §6), category grades, permission/evidence summary, findings with expandable resolution chains, classification-coverage metric. Single file, no external assets, no network.
- [x] SARIF 2.1.0 export: findings→results, controls→rules, severity mapping, resource locations. Explicitly a projection — the internal model stays canonical JSON. Validated against the SARIF schema in tests.
- [x] `score` command per SPEC §16: numeric weighted score + category grades + critical-cap rule, reading canonical JSON.
- [x] CI exit codes: 0 clean, 1 findings ≥ `--fail-on <sev>`, 2 scan error. **Unknowns never affect exit code by default**; `--fail-on-unknown` is opt-in.

## Definition of Done
- [x] **First-run zero-config scan produces the SPEC §6-shaped summary with honest unknowns** — this is the v0 release criterion.
- [x] HTML renders from a golden JSON in a headless check (well-formedness + key sections present); SARIF validates against its schema in tests.
- [x] All fixtures (sidecar-basic, sidecar-authz, ambient-basic, mixed, governance) green in e2e including RBAC proofs; determinism re-run.
- [x] `make build test lint schema-test` + e2e green.

## Human review gate
Exit-code contract and SARIF projection fidelity — these are the CI-integration and code-scanning surfaces users wire into pipelines; a wrong exit code or a dropped finding in SARIF is a silent trust failure.

## Out of scope
Prometheus / runtime verification (M7), scan --local, drift, multi-cluster evaluation. Release engineering is M6.5 (see plan/M6.5-release.md), triggered on this milestone's completion.

## Deferred

- Per-control-area clean summaries for public-gateway and broad-egress controls
  need canonical aggregates or a canonical control catalog. Absence of a
  finding is not proof of a pass, so M6c does not infer those rows in HTML.
- The frozen schema contains namespace numeric scores but not namespace
  category grades. `score --namespace` labels the available grades as cluster
  grades; adding namespace grades requires a separately approved contract
  change.
- A pinned real-browser smoke can supplement the deterministic
  `golang.org/x/net/html` headless parser when the test environment supplies a
  browser binary. M6c does not add an unpinned runtime download to the release
  gate.
- The published lifecycle dimension remains explicit `unknown` until its
  control-owning milestone supplies lifecycle rules.

## Summary

### Decisions

- Canonical JSON remains the sole model. `report`, `export`, `score`, and the
  scan/score threshold decisions read that model and do not call collectors,
  resolvers, or the control engine. Before projection, input is validated
  against a build-embedded copy of the frozen schema; a parity test requires
  the embedded copy to equal
  `docs/contracts/canonical-json-schema.json` after compaction.
- With explicit human approval, every integer that represents a canonical
  count now has `minimum: 0`: inventory resource counts, ztunnel nodes,
  waypoints, classification totals/by-environment, and category
  `evaluated`/`unknown`. Negative counters fail complete schema validation
  before HTML, SARIF, score, or threshold evaluation.
- HTML is one static file with inline CSS, no scripts, no external assets, a
  network-denying content-security policy, and the OpenMeshGuard evidence-grade
  visual system. Unknown state is a top-level status, not an empty state.
  Authorization coverage treats `waypoint-policy-unenforced` as uncovered.
  Runtime `corroborated`, `contradicted`, `no-traffic-observed`, `unknown`, and
  unavailable states remain distinct with their canonical evidence fields.
  Every canonical workload mTLS and authorization conclusion has its ordered
  resolution chain in a dedicated report section.
- SARIF uses one result for every canonical finding and one rule for every
  referenced control ID. Open, excepted, unknown, and not-applicable findings
  all remain present with deliberate kind/level/suppression mappings and their
  original canonical status and severity in properties. Unknowns use
  `kind: review`, `level: none` as required by SARIF 2.1.0 section 3.27.10.
  The checked-in OASIS SARIF 2.1.0 Errata 01 schema validates the result, and
  tests assert exact finding count/ID parity.
- The score producer uses published control-pack data for the 100-point
  weights and critical cap. Unknown categories are excluded from the numeric
  denominator but stay explicit. The cluster score rolls up evaluable
  environment-weighted namespace scores and a separate global-weighted
  cluster-scoped component; each contributes its evaluable category weights,
  so neither environment overrides nor cluster-scoped exception controls
  disappear or become overweighted. Open critical findings cap the affected
  component; excepted, unknown, and not-applicable findings do not.
- Severity order is `critical > high > medium > low > info`, with inclusive
  boundaries. Only canonical open findings affect `--fail-on`. Unknown
  findings affect exit status only under `--fail-on-unknown`; malformed input
  and all operational/projection errors, including broken stdout pipes, exit
  2. Canonical validation asserts schema formats and rejects non-UTF-8 input
  before any projection or threshold evaluation.

### Review findings

1. **Fixed — ambient unenforced authorization looked covered.** The first HTML
   projection counted `waypoint-policy-unenforced` in the covered total even
   while the same canonical report contained an open high finding. The switch
   is now exhaustive and the ambient-missing golden proves `0/1` coverage.
2. **Fixed — runtime states were blended into verified.** Contradicted and
   no-traffic-observed workloads previously incremented a green verified
   counter. The summary and workload evidence table now preserve all canonical
   runtime states, including window, mTLS share, plaintext observation, and
   sources.
3. **Fixed — malformed canonical input could exit clean.** Top-level-only
   decoding allowed invalid statuses, severities, chains, and score shapes to
   reach CI evaluation. Every projection now performs complete frozen-schema
   validation, with command-level exit-2 and renderer regressions.
4. **Fixed — cluster-scoped exception hygiene could miss the cluster score.**
   The old average used namespace scores whenever any existed. The cluster
   score now combines environment-weighted namespace components with a
   separate cluster-scoped component, while namespace scores remain separate
   canonical rollups.
5. **Fixed — the lifecycle score dimension disappeared.** Categories with
   published weight but no current controls are seeded as explicit unknowns.
   All 22 score goldens record lifecycle with `passRate: null`.
6. **Fixed — valid optional SARIF titles could fail export.** A missing first
   title no longer conflicts with a later nonempty title for the same control;
   only genuinely conflicting nonempty titles are rejected.
7. **Fixed — README advertised M7 functionality in v0.** The unsupported
   `--prometheus-url` command and current-runtime-verification claims were
   removed. The current surface now states that runtime verification is
   unavailable until M7.
8. **Fixed — acceptance documentation undercounted reports.** The harness is
   documented as 21 fixture cases, one namespace-degradation golden, and one
   all-namespaces report: 23 schema-valid reports, 22 checked-in goldens.
9. **Structured autoreview unavailable; local adversarial review used.** The
   prescribed helper could not open its read-only Codex state database and
   stopped before analysis. Three dedicated read-only agents audited
   projection fidelity, exit/score boundaries, and release proof. Their
   evidenced blockers are items 1–8; both the projection-fidelity and final
   score/exit re-reviews returned ACCEPT after remediation.

### Codex review thread remediation ledger

Review thread `019f99c6-9a5f-7433-bb65-6460dfe24b31` froze
`ad21e33a8a71351f679abb4131c3f37fbe1fbbb5`. The current `HEAD` matched that
SHA exactly; only the pre-existing `docs/dev.md` and this plan were dirty.

| # | Original severity and anchor | Disposition | Reproduction and scoped response |
|---:|---|---|---|
| 1 | HIGH — `internal/output/canonical.go:21` | CONFIRMED | Invalid RFC 3339 dates and raw `0xff` both projected with exit 0. Format assertions and UTF-8 rejection now precede every projection; hostile renderer and command regressions require exit 2. |
| 2 | HIGH — `internal/output/sarif.go:200` | CONFIRMED | Unknown emitted `review/warning`. It now emits normative `review/none` while retaining canonical severity in properties; schema and exact ID parity tests remain green. |
| 3 | HIGH — `internal/output/html.go:164` | CONFIRMED | A canonical contradiction was hidden in the headline when the Prometheus source flag was false. Explicit canonical runtime states now win over the unavailable fallback. |
| 4 | HIGH — `internal/output/json.go:381` | CONFIRMED | A one-namespace production override yielded different cluster and namespace scores. Overall now rolls up environment-weighted namespace components plus cluster-scoped controls, with component-specific critical caps. |
| 5 | HIGH — `internal/output/html.go:474` | CONFIRMED | Unique workload mTLS/authz chain markers were absent while finding chains rendered. A dedicated workload-posture section now renders both ordered chains. |
| 6 | MEDIUM — `cmd/openmeshguard/output_commands.go:152` | CONFIRMED | Large stdout projections piped to a closed reader terminated with 141. The CLI ignores SIGPIPE on supported POSIX targets so the write error reaches the normal exit-2 path; a subprocess regression proves it. |
| 7 | MEDIUM — `docs/contracts/canonical-json-schema.json:176` | CONFIRMED | Explicit human approval authorized the contract correction. Every canonical counter now has `minimum: 0`; direct contract, embedded-schema, four-consumer, and three-command tables reject negative values with exit 2 and no output. |
| 8 | MEDIUM — `HEAD:plan/M6c-outputs.md:12-30` | ALREADY FIXED | The completion record and corrected 23-report acceptance documentation already existed in the pre-review dirty worktree. Those unrelated user changes were preserved; no commit was made. |

### Flags raised

- The explicitly approved `docs/contracts/canonical-json-schema.json` change
  only adds non-negative minima to count-valued integers, with a byte-parity
  embedded runtime copy. No file in `internal/resolver`, `internal/collect`,
  or `deploy/rbac/` changed. No exported `internal/output` or
  `internal/resolver` type changed. Resolver provenance remains
  `mtls/v5,authz/v8`.
- No Kubernetes resource, verb, or client action was added. Both clean E2E
  runs retained exactly 456 scanner calls, all approved `list` operations,
  with no Secret or subresource access.
- The browser automation wrapper required an unpinned npm executable download,
  which the environment rejected. The release gate instead uses the
  deterministic in-process headless HTML parser plus live E2E key-section and
  no-network assertions; no test was weakened or skipped.
- The final disposable-cluster cleanup attempt could not obtain Docker approval
  after the environment reached its approval-usage limit. The harness had
  already removed its administrator kubeconfig and every scanner credential;
  the local `openmeshguard-e2e` cluster may remain until `make kind-down` can
  be authorized.
- SPEC §6 examples whose pass state is absent from canonical JSON remain
  deferred rather than being recomputed or invented in the HTML view.

### Verification

- The exact combined `make build test lint schema-test` invocation is green on
  the final Go, golden, and documentation state. The complete unit suite and
  shell harness checks passed, lint reported `0 issues`, and schema tests cover
  canonical output, the runtime schema-copy parity guard, and SARIF against the
  official OASIS schema.
- The 5×5 severity/threshold matrix covers all 25 inclusive boundaries.
  Status tables cover clean, open, excepted, unknown default, unknown opt-in,
  not-applicable, and expired/open behavior. Command tests prove output is
  still produced for exit 1 and malformed canonical input exits 2.
- The schema gate directly table-tests every count-valued integer against a
  negative value. Runtime projection tables repeat the same cases across HTML,
  SARIF, score, and threshold evaluation; command tables require exit 2 with
  no output for representative score, inventory, and classification counters.
- Headless HTML tests parse a checked-in golden, require all key sections,
  prohibit scripts and external-capable attributes, assert every canonical
  finding ID is present, and double-render for determinism. Projection tables
  cover waypoint-unenforced authorization and every runtime status. SARIF
  tests assert official-schema validity, every-finding count/ID parity,
  explicit special-status mappings, optional titles, and deterministic output.
- `make kind-up` completed in 37 seconds with Kind v0.31.0, digest-pinned
  Kubernetes 1.35.0, Istio 1.30.2 ambient, and Gateway API v1.5.1. The guarded
  `UPDATE_GOLDEN=1 make e2e` completed in 64 seconds and wrote 22 reviewed
  score-only goldens; all 23 reports and every output/RBAC/audit proof passed.
- Two clean `make e2e` runs completed in 75 and 72 seconds. Both matched all
  goldens and recorded exactly 456 approved scanner calls: 416 cluster, 20
  waypoint-limited, and 20 namespace lists, plus the separate denied
  audit-probe write positive control.
- The zero-config cluster report had all context-file flags false, Prometheus
  disabled, 27 unclassified namespaces, 33 workloads without verified posture,
  and lifecycle explicitly unknown. HTML exposed every required section;
  SARIF contained all 272 canonical findings with exact ID parity; score output
  was nonempty.
- The Codex review remediation re-recorded exactly three inspected
  governance-score golden changes, then completed two clean 70-second E2E
  runs. Both produced 456 approved list calls and identical normalized
  canonical/HTML/SARIF/score SHA-256 values. The live report retained 272/272
  canonical/SARIF finding ID parity and the zero-config honest-unknown shape.
- After explicit approval of the non-negative counter contract, the exact
  combined gate passed again and a further clean `make e2e` completed in 73
  seconds. All 22 goldens matched without regeneration, all 23 reports
  remained schema-valid, the action audit remained exactly 456 approved
  `list` calls, and the normalized canonical/HTML/SARIF/score hashes remained
  unchanged. The live SARIF retained exact 272/272 canonical finding ID parity.
