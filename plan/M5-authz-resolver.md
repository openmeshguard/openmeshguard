# M5 — Effective Authorization Resolver

Branch: `m5-authz-resolver`

## Goal
Complete `ResolveAuthz` per contract: AuthorizationPolicy evaluation order, attachment-scope resolution, L4-vs-L7 enforceability, waypoint attachment, chains — same pure-function, table-driven discipline as M2.

## Context
SPEC.md §7 (authorization effective posture, scope resolution). Contract: resolver_types.go. Upstream references to cite: Istio authorization policy docs (CUSTOM → DENY → ALLOW evaluation, implicit behavior when no ALLOW matches, root-namespace policies) and ambient waypoint policy attachment (targetRefs, L7 attribute enforcement location).

## Deliverables
- [x] Evaluation-order model: CUSTOM → DENY → ALLOW; "no ALLOW policy in scope" vs "ALLOW present but workload unmatched" distinguished correctly.
- [x] Scope resolution: root-namespace (mesh-wide) policies, namespace policies, workload-selector policies; effective classification into the AuthzEffective enum per contract.
- [x] Default-deny detection: recognize the empty-ALLOW-policy default-deny idiom at namespace and mesh scope.
- [x] Ambient split: RequiresL7 policies with no ready waypoint in the enforcement path ⇒ `l7-policy-unenforced`, policy names in L7Unenforced, and the missing/unready waypoint recorded in the chain.
- [x] Broad-allow refinement: resolver confirms/refines the normalizer's BroadAllow hint (empty rules, wildcard principals/namespaces).
- [x] Normalizer upgrades this milestone requires: AuthorizationPolicy collection + views, waypoint discovery (istio.io/use-waypoint labels, Gateway API waypoints), Sidecar-resource and exportTo scoping into WorkloadInput (the normalizer scopes, the resolver decides — per contract comment).
- [x] Deferred M3 producer: collect DestinationRules with the typed Istio client and scope normalized views into WorkloadInput; a collected empty set must remain distinguishable from unavailable evidence. Published RBAC already covers this resource and must not change.
- [x] Deferred M3 producer: normalize workload ports into WorkloadInput with explicit availability, distinguishing an observed-empty port set from uncollected evidence so port-level mTLS resolution is evaluable without zero-value fallthrough.
- [x] Table-driven tests covering AT MINIMUM: no policy anywhere; root-ns default-deny + local allow; DENY overriding ALLOW; CUSTOM present; allow-only namespace; selector mismatch; ambient L7 policy without waypoint; ambient L7 policy with ready waypoint; mixed L4/L7 rules in one policy; root-ns policy + namespace override interplay.
- [x] Built-in authz control pack: MG-AUTHZ-001..007 per SPEC §15, using `requires: [authorization.effective]`.
- [x] Built-in mTLS completion: make MG-MTLS-002 evaluable from produced ports and ship MG-MTLS-007 for DestinationRule client/server TLS contradictions.
- [x] E2E producer cutover: transition `port-level-override` and `dr-contradiction` from golden-unknown to golden-resolved, update their guards and notes, add the cases.tsv-driven `sidecar-authz/` fixture group with goldens, and prove deterministic output with two consecutive `make e2e` runs including both RBAC proofs.
- [x] Chain assertions in every table case. `Version()` becomes a composite (current tags: `mtls/v2,authz/v1`) — document the scheme in the resolver package doc.

## Definition of Done
- [x] All tables pass; flags-raised summary as in M2 (expected-output uncertainty goes to the human with Istio doc links first).
- [x] Kind fixture `sidecar-authz/` added with golden findings; e2e green.
- [x] Purity check green.

## Out of scope
Ambient ztunnel/waypoint HEALTH controls (M6) — this milestone models attachment/enforceability only.

## Summary

### Decisions

- Authorization resolution uses Istio's additive policy model: applicable root-namespace and namespace policies form one union, ordered in the evidence chain as CUSTOM, DENY, then ALLOW. Selector mismatches remain explicit exclusion steps rather than disappearing as if no policy resource existed.
- An ALLOW policy with no `rules` is the default-deny idiom; `rules: [{}]` is an explicit broad allow. A default-deny ALLOW plus explicit ALLOW rules resolves to `default-deny-explicit-allow`; a namespace ALLOW without that idiom remains `allow-only`; any applicable DENY resolves to `deny-present` without discarding additive ALLOW evidence.
- CUSTOM is recorded ahead of native policies. CUSTOM plus a native posture uses the native effective enum and retains the CUSTOM chain step; CUSTOM-only remains explicit `unknown` because the external provider's result cannot be represented by the frozen enum.
- Ambient handling models attachment and enforceability only. A targetRef L7 policy needs a selected, observed, Programmed waypoint; missing or unready attachment resolves to `l7-policy-unenforced`, unavailable Gateway evidence resolves to `unknown`, and selector-based L7 policy at ztunnel is recorded as Istio's fail-safe DENY. Ambient membership detection remains the M6 stub and is not inferred in M5.
- The typed collector now gathers Services, DestinationRules, Sidecars, AuthorizationPolicies, and Gateway API Gateways with bounded list calls and per-scope availability. The normalizer produces Service-bound ports, DestinationRule views, exportTo and Sidecar scoping, authorization views, and waypoint attachment without client-go leakage into the resolver. Observed-empty ports and policy sets remain distinct from unavailable evidence.
- Review corrections preserve a DestinationRule port-level override even when that entry omits TLS, apply the root-namespace default Sidecar only when no namespace-local Sidecar applies, and split controller output to pod-level workloads when observed pods have different service/policy inputs.
- The resolver version is the stable ordered composite `mtls/v2,authz/v1`; only the subsystem tag whose semantics change is bumped. MG-AUTHZ-001..007 and MG-MTLS-007 ship as YAML/CEL data, and produced ports make MG-MTLS-002 evaluable.
- `port-level-override` and `dr-contradiction` now golden resolved evidence, their transitional notes are retired, and the new `sidecar-authz` cases cover root/local union, broad allow, DENY precedence, allow-only, and selector mismatch.
- The branch was rebased cleanly onto current `origin/main` at `abd6a49`; `a931d73` remains the first M5 commit and contains only the required Deliverables expansion.

### Flags raised

- CUSTOM → DENY → ALLOW ordering and root/local interaction were held for human review before tables were encoded. The approved result follows Istio's documented layered order and additive merge model, not PeerAuthentication's oldest-policy tie-break: [Istio authorization implicit enablement](https://istio.io/latest/docs/concepts/security/#implicit-enablement).
- Empty `{}` ALLOW versus `rules: [{}]` was held before encoding. Istio documents missing rules as never matching and an empty rule as always matching; the approved tables and `broadAllow` output preserve that distinction: [AuthorizationPolicy reference](https://istio.io/latest/docs/reference/config/security/authorization-policy/) and [allow-nothing/allow-all examples](https://istio.io/latest/docs/concepts/security/#allow-nothing-deny-all-and-allow-all-policy).
- Root-namespace policy interaction was held before encoding. Istio documents root policies as mesh-wide and selectors there as matching across namespaces; the approved resolver unions matching root and local policies and records selector exclusions: [AuthorizationPolicy scope](https://istio.io/latest/docs/reference/config/security/authorization-policy/).
- Ambient L7 boundaries were held before encoding. Istio documents targetRefs attachment to waypoints, ztunnel's inability to enforce L7, and selector-based L7 fail-safe DENY; the approved resolver distinguishes enforced, unenforced, and unavailable evidence: [Istio ambient L7 features](https://istio.io/latest/docs/ambient/usage/l7-features/).
- CUSTOM-only was held because Istio delegates the decision to the named external provider. The human approved `unknown` when no native effective class is available: [AuthorizationPolicy CUSTOM action](https://istio.io/latest/docs/reference/config/security/authorization-policy/).
- `broadAllow` was initially withheld, explained, and then explicitly approved as a structural signal for an applicable ALLOW containing an empty rule or wildcard principal/namespace. It is not a request-level simulation and remains omitted when authorization evidence is unavailable.
- The human explicitly approved two frozen-contract edits: additive `AuthorizationPolicyView.HasSelector` and `HasRules`, `WaypointView.Known`, and optional canonical `authorization.broadAllow`; and removal of the unused `MeshDefaults.MeshMTLSMode`. No other exported resolver/output or `docs/contracts` shape changed.
- DestinationRule port overrides do not inherit omitted destination-level fields; review found and fixed the producer so an omitted port TLS block is retained as explicit unset/default evidence: [DestinationRule port-level settings](https://istio.io/latest/docs/reference/config/networking/destination-rule/).
- Root Sidecar defaults were added after review found that scoped scans collected only the workload namespace. Istio documents the root Sidecar as the fallback only when a namespace has no applicable local configuration: [Sidecar reference](https://istio.io/latest/docs/reference/config/networking/sidecar/).
- No RBAC profile was edited. Collection remains typed, bounded, get/list-only, and Secret-free; the action audit covers every new resource.
- Structured autoreview was attempted. Its sandboxed run could not initialize the read-only Codex state database, and policy rejected exporting the branch bundle outside the sandbox. No workaround was used; local adversarial review found and fixed the three producer/scoping issues listed above, and the final local diff review has no actionable finding.

### Verification

- `make build`, `make test`, `make lint`, and `make schema-test`: green on the rebased tree. Lint reported zero issues and includes the resolver dependency/purity guard. The successful build emitted only Go's sandbox stat-cache warning.
- Focused collector, normalizer, resolver, and CLI tests pass. Resolver tables assert exact chain order, kinds, fields, and effects in every case; built-in control tables cover pass, fail, unknown, and not-applicable outcomes.
- Golden-update proof changed only `namespace-role-degraded.json`: root `Sidecar` access is now correctly denied for `namespace/istio-system`. That run passed both RBAC proofs with 220 approved list events and no other calls in 48 seconds.
- A fresh `make kind-up` completed in 41 seconds. Two consecutive non-update `make e2e` runs on the final fixture table both matched every `sidecar-basic` and `sidecar-authz` golden, passed the ClusterRole and namespace Role proofs, recorded exactly 220 approved scanner list events and no other calls, and completed in 36 and 50 seconds.
- `git diff -- deploy/rbac` and `git diff --check`: clean. The retired note files are absent, golden case bijection remains enforced, and no scanner credential survived the harness cleanup.
