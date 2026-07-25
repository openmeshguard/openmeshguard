# M6b — Governance Context (classification, ownership, exceptions)

Branch: `m6b-governance-context`

## Goal
Governance inputs land as first-class, unknown-first context: environment classification, ownership, and exception records — enabling environment-scoped control evaluation without ever silently suppressing a real finding.

## Context
SPEC.md §9 (classification + ownership precedence), §10 (exceptions), §15 (context controls). Canonical schema `workloadPostures[].environment`/`environmentConfidence`/`owner`, `findings[].status` (`excepted`), `findings[].exception`, and `inventory.classification` are authoritative. Builds on M6a (ambient) being merged.

## Deliverables
- [x] Classification per SPEC §9 precedence: scan-config mapping → `openmeshguard.io/environment` label → fallback labels → `--infer-environments` heuristics → unclassified. Heuristics are OFF by default, produce `inferred` confidence, and are disclosed in the report. Unclassified is explicit, never a default. MG-ENV-001.
- [x] Ownership per SPEC §9 precedence: labels/annotations → scan-config → import file. MG-OWN-001/002.
- [x] Exception records + annotation matching per SPEC §10: a matched finding becomes `status: excepted` and is **never removed** from output; an expired exception restores the original severity and raises MG-EXC-002; MG-EXC-001 validates exception record fields. Exceptions are engine-applied after evaluation — controls never reference them.
- [x] Environment-scoped control evaluation activates: production-only controls (e.g. MG-MTLS-001) evaluate only classified-production workloads; unclassified is covered by MG-ENV-001, not by silent pass.
- [x] scan-config file format for classification/ownership/exception inputs. If this needs a new frozen-contract surface, STOP and propose it for human approval before writing.
- [x] Context controls ship as YAML/CEL data; e2e fixtures exercise classified/unclassified, owned/unowned, and active/expired-exception cases with goldens; RBAC proofs + determinism re-run.

## Definition of Done
- [x] Classification, ownership, and exception state all carry explicit unknowns; no path silently passes, fails, or drops a finding.
- [x] An excepted finding is present-but-marked in output; an expired exception is visibly restored with MG-EXC-002; proven by fixture goldens.
- [x] `--infer-environments` stays opt-in and its confidence is disclosed. `make build test lint schema-test` + e2e green.

## Human review gate
**Exception matching** — a mis-scoped exception silently suppresses a real finding, so the matching rules and the never-removed / expired-restored behavior are the highest-stakes semantics in this milestone. Review the exception fixture goldens deliberately.

## Out of scope
Ambient (M6a), HTML/SARIF/score/exit-codes (M6c), Prometheus (M7).

## Deferred

- Add cluster-, namespace-, selector-, and exact-resource exception scopes after
  real users demonstrate that annotation-only references are insufficient.
  M6b deliberately consumes enterprise exception decisions rather than
  implementing a second exception workflow.
- Add exception owner, justification, source-file, and annotation-binding
  provenance to canonical output only after the frozen
  `findings[].exception` and resolution-chain contracts receive explicit human
  approval. M6b keeps the approved ID, expiry, approver, ticket, evidence
  source, and unchanged posture chain while enforcing owner-bound matching.
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

### Expected review-remediation delta before regeneration

- No existing golden changes are expected. The governance fixtures already use
  string-normalized environments, loaded control IDs, matching exception and
  workload owners, stable controller metadata, and unambiguous enrollment.
- One new `governance-owner-mismatch` golden is expected. It reuses the active
  workload with a record owned by another team; MG-MTLS-001 must remain open
  without exception evidence and MG-EXC-001 must report the invalid binding.
- The active and expired exception projections remain contract-identical. The
  E2E semantic guard now additionally requires the existing `expiresAt`,
  `approver`, `ticket`, and `exception-record` evidence fields, so regeneration
  cannot silently discard them.

### Decisions

- The human-approved user-facing formats are documented in [governance
  context](../docs/context.md). Classification uses an ordered source list:
  exact namespace mapping, configured namespace labels, namespace-name regular
  expressions, a whole-cluster environment, and cluster-context mappings are
  available building blocks. Configured sources replace defaults; the default
  label order is `openmeshguard.io/environment`, `environment`, then `env`.
  An unavailable higher-precedence source stops as unknown. Complete unmatched
  evidence becomes explicit `unclassified`; opt-in inference runs last and
  reports `inferred` confidence.
- Ownership applies configurable application-ID and owner keys to workload
  labels, workload annotations, namespace labels, and namespace annotations in
  that order. Config mappings then ownership imports resolve
  application-ID-to-owner. One imported application ID therefore spans any
  number of namespaces without namespace mappings. Configured key lists replace
  the documented defaults.
- Exception scope is deliberately annotation-only:
  `openmeshguard.io/exception: <id>` on the exact workload resource references
  one record. Records contain control IDs, owner, approver, justification,
  HTTPS ticket, and RFC3339 expiration; they contain no scope selector, status,
  revocation, or workflow state. Those broader capabilities remain in
  Deferred.
- Exceptions are applied only after CEL evaluation and only to an underlying
  `open` finding. Active exceptions mutate that finding to `excepted` without
  changing severity, reasoning, resources, or its chain. Expired exceptions
  attach evidence but leave the finding `open` at the already-evaluated
  severity and independently raise MG-EXC-002. Unknown, not-applicable, and
  MG-EXC-001/002 findings cannot be excepted.
- Scan-config control data uses the M3 engine hooks: pack parameters merge
  below scan defaults and environment parameters; an override replaces a
  control's environment list and can set severity by environment. Invalid
  severities, duplicate overrides, and unknown control IDs fail loading.
  Scan-config provenance is emitted as user pack
  `scan-config:<metadata.name>` at the configured version.
- MG-ENV-001, MG-OWN-001/002, and MG-EXC-001/002 are YAML/CEL data with
  pass/fail/unknown/not-applicable tables. Built-in mTLS/authz/context pack
  metadata is `0.3.0`; resolver provenance remains
  `mtls/v5,authz/v8`.

### Review findings

1. **Fixed — exceptions could replace unknown/not-applicable state.** The
   post-evaluation matcher originally changed any matching finding to
   `excepted`. It now accepts only `open`; table regressions prove unknown and
   not-applicable findings remain unchanged and unannotated.
2. **Fixed — exception hygiene controls were accepted in records.** The engine
   already refused to except MG-EXC-001/002, but such a record still passed
   validation. Record validation now raises MG-EXC-001 for either control while
   the engine skip remains defense in depth.
3. **Fixed — annotation ownership lost evidence provenance.** Annotation-based
   ownership resolved correctly but its evidence projection only recognized
   label source names. Both workload and namespace annotations now contribute
   `kubernetes-api`, with a regression covering scan-config classification plus
   annotation ownership.
4. **Fixed — ownership fields degraded together.** Missing namespace metadata
   made both application ID and owner unknown even when one existed directly
   on the workload. The fields now resolve independently; unavailable
   higher-precedence owner evidence still blocks config/import fallback.
5. **Fixed — golden mutation proof targeted a retired finding.** Once
   MG-MTLS-001 became production-scoped, the old mutation deleted nothing from
   the unclassified permissive golden. It now deletes MG-MTLS-002, and separate
   mutations prove the guard rejects removal of both the active-excepted and
   expired-restored MG-MTLS-001 findings.
6. **Disposed as fixture isolation — an expired record affected every
   governance scan that loaded it.** This is correct exception-hygiene
   behavior, not an engine defect. The classified/owned and
   unclassified/unowned cases no longer load exception files; only the active
   and expired cases do, keeping each golden's purpose explicit.
7. **Structured autoreview unavailable; no workaround used.** The prescribed
   branch review command first failed because the sandbox made the Codex state
   database read-only. Its escalated retry was rejected because it would
   export the branch bundle to an external review service without separate
   authorization. A local read-only adversarial review found and fixed items
   1–5; the final local diff has no remaining actionable finding.
8. **Mitigated within the approved contract — exception IDs were replayable
   across workload owners.** The annotation-only format remains unchanged, but
   a valid record now matches only when `spec.owner` equals the workload's
   resolved owner. Owner mismatch raises MG-EXC-001; unavailable owner evidence
   makes the reference unknown; both leave the original finding open.
   Engine-level owner checks provide defense in depth. A namespace editor who
   can also forge the approved owner metadata could still replay an ID; fully
   preventing that requires the scoped-record format already recorded in
   Deferred and therefore remains an explicit accepted M6b limitation rather
   than an unapproved contract expansion.
9. **Fixed — overrides could filter mandatory governance controls.**
   Environment-list overrides are rejected for MG-ENV-001, MG-OWN-001, and
   MG-EXC-001/002, preventing an override from removing the coverage finding
   for an unclassified, unowned, invalid, or expired target.
10. **Fixed — known non-enrollment dominated unknown enrollment.** Namespace
    aggregation now uses the lattice enrolled > unknown > not-enrolled. An
    unknown workload keeps the namespace in MG-ENV-001 evaluation even when a
    sibling is conclusively outside the mesh; the result is order-independent.
11. **Fixed — governance metadata depended on normalization shape.** Workload
    context now includes controller resource labels/annotations, standalone
    ReplicaSets, and owning controller metadata for per-Pod normalized
    results. Resource metadata supplements templates, and the highest owning
    controller wins on collisions, so rollout-driven normalization does not
    change or discard governance context.
12. **Fixed — environment scalars were coercible and whitespace-sensitive.**
    Classification and control environment inputs require YAML strings and
    reject surrounding whitespace, preventing values such as boolean `true`
    or `" production "` from silently missing production control scopes.
13. **Fixed — headerless exception documents were discarded as blank.** Only
    a structurally empty YAML document is skipped. A document containing
    `spec` without the required header now fails strict loading.
14. **Fixed — exception control IDs were only syntax-checked.** Records are
    now checked against the complete loaded built-in and user control set.
    Unknown IDs invalidate the record and produce MG-EXC-001 instead of an
    inert apparently-valid exception.
15. **Partially accepted; contract-gated — exception provenance could be
    stronger.** The E2E guard now protects every existing canonical exception
    field and `exception-record` evidence. Adding owner, justification,
    source-file, or binding-chain fields would change the frozen canonical
    output/resolution-chain contract, so that work is recorded in Deferred
    instead of overloading approved fields.
16. **Disposed as approved behavior — ordered classification sources.** The
    exact-mapping, label, configured name-rule, cluster, and cluster-context
    building blocks execute in the operator-declared order documented in the
    approved format. A configured regex is an explicit resolved source;
    `--infer-environments` still controls only the built-in heuristic fallback.
17. **Disposed as approved behavior — namespace/selector ownership maps.**
    The approved M6b bridge resolves configurable application-ID metadata to an
    owner, allowing the same ID across namespaces. Reintroducing the earlier
    SPEC namespace/selector mapping would reverse the human-approved format
    rather than fix an implementation defect.

### Flags raised

- The canonical report fields were implemented against the unchanged
  [canonical JSON schema](../docs/contracts/canonical-json-schema.json), and
  exception application follows the unchanged [control-format
  boundary](../docs/contracts/control-format.md) that controls never reference
  exceptions. No file in `docs/contracts/` changed.
- No exported type in `internal/resolver` or `internal/output` changed.
  `git diff origin/main...HEAD -- docs/contracts internal/resolver deploy/rbac`
  is empty, satisfying the frozen-contract and RBAC approval gates in
  [AGENTS.md](../AGENTS.md).
- No Kubernetes resource or action was added. Governance derives from already
  collected Namespace/workload metadata and local files. The published RBAC
  remains unchanged, and the live audit remains list-only, Secret-free, and
  watch-free.
- The actual existing-golden control delta exactly matches the pre-regeneration
  forecast: only MG-MTLS-001 and MG-AUTHZ-001/002 were removed, and every mesh
  namespace losing them gained MG-ENV-001. The known outside-mesh fixture lost
  only their not-applicable instances and retains explicit not-applicable
  ownership coverage. No production finding disappeared from a mesh target
  without environment coverage.
- Removing an exception record is the M6b revocation mechanism, with Git
  preserving history. Explicit status/revocation fields, selectors, broader
  scopes, multi-record resolution, and enterprise-system integrations remain
  deferred rather than implied.

### Verification

- Final `make build test lint schema-test` is green. The complete Go and shell
  test suite passed; lint reported `0 issues`; schema tests validated generated,
  fixture, and external reports against the unchanged canonical schema.
- Table-driven tests cover ordered classification sources and confidence,
  opt-in inference, whole-cluster classification, configurable workload and
  namespace label/annotation ownership, config-before-import precedence,
  independent unknown fields, strict file loading, environment parameters and
  overrides, every context-control outcome, and exception never-removal,
  expiry, invalidity, wrong-control, hygiene-control, unknown, and
  not-applicable cases. Review regressions additionally cover all mandatory
  governance-control override rejections, boolean and whitespace environment
  inputs, loaded exception control IDs, nonempty headerless documents,
  cross-owner exception binding, mixed unknown/non-mesh namespace enrollment
  in both orders, controller metadata collisions, standalone ReplicaSets, and
  per-Pod normalization.
- The final guarded `UPDATE_GOLDEN=1 make e2e` completed in 63 seconds. It
  schema-validated all 22 golden reports plus the all-namespaces report, passed
  semantic guards before copying, and recorded 456 approved list calls and no
  other scanner calls. Every existing golden remained unchanged; the only
  addition was `governance-owner-mismatch`. Active and expired exception
  goldens retain exactly one MG-MTLS-001 finding at configured `critical`
  severity; the active status is `excepted`, while expired is `open` with
  exception evidence plus MG-EXC-002. The owner-mismatch golden keeps
  MG-MTLS-001 open without exception evidence and raises MG-EXC-001.
- Two consecutive final non-update `make e2e` runs matched all goldens, passed
  the ClusterRole, namespace Role, and waypoint-limited proofs, and completed
  in 74 and 76 seconds. Both recorded the same 456 approved events: 416
  cluster-scanner lists, 20 waypoint-limited lists, and 20 namespace-scanner
  lists, plus the separate denied audit-probe write positive control. No
  scanner credential survived cleanup.
- `make kind-up` completed in 41 seconds with Kind v0.31.0, digest-pinned
  Kubernetes 1.35.0, Istio 1.30.2 ambient, and Gateway API v1.5.1.
  `make kind-down` removed the disposable cluster in 1 second.
  `git diff --check` and the frozen-contract/RBAC diff checks are clean.
