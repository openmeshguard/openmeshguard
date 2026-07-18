# M5 — Effective Authorization Resolver

Branch: `m5-authz-resolver`

## Goal
Complete `ResolveAuthz` per contract: AuthorizationPolicy evaluation order, attachment-scope resolution, L4-vs-L7 enforceability, waypoint attachment, chains — same pure-function, table-driven discipline as M2.

## Context
SPEC.md §7 (authorization effective posture, scope resolution). Contract: resolver_types.go. Upstream references to cite: Istio authorization policy docs (CUSTOM → DENY → ALLOW evaluation, implicit behavior when no ALLOW matches, root-namespace policies) and ambient waypoint policy attachment (targetRefs, L7 attribute enforcement location).

## Deliverables
- [ ] Evaluation-order model: CUSTOM → DENY → ALLOW; "no ALLOW policy in scope" vs "ALLOW present but workload unmatched" distinguished correctly.
- [ ] Scope resolution: root-namespace (mesh-wide) policies, namespace policies, workload-selector policies; effective classification into the AuthzEffective enum per contract.
- [ ] Default-deny detection: recognize the empty-ALLOW-policy default-deny idiom at namespace and mesh scope.
- [ ] Ambient split: RequiresL7 policies with no ready waypoint in the enforcement path ⇒ `l7-policy-unenforced`, policy names in L7Unenforced, and the missing/unready waypoint recorded in the chain.
- [ ] Broad-allow refinement: resolver confirms/refines the normalizer's BroadAllow hint (empty rules, wildcard principals/namespaces).
- [ ] Normalizer upgrades this milestone requires: AuthorizationPolicy collection + views, waypoint discovery (istio.io/use-waypoint labels, Gateway API waypoints), Sidecar-resource and exportTo scoping into WorkloadInput (the normalizer scopes, the resolver decides — per contract comment).
- [ ] Deferred M3 producer: collect DestinationRules with the typed Istio client and scope normalized views into WorkloadInput; a collected empty set must remain distinguishable from unavailable evidence. Published RBAC already covers this resource and must not change.
- [ ] Deferred M3 producer: normalize workload ports into WorkloadInput with explicit availability, distinguishing an observed-empty port set from uncollected evidence so port-level mTLS resolution is evaluable without zero-value fallthrough.
- [ ] Table-driven tests covering AT MINIMUM: no policy anywhere; root-ns default-deny + local allow; DENY overriding ALLOW; CUSTOM present; allow-only namespace; selector mismatch; ambient L7 policy without waypoint; ambient L7 policy with ready waypoint; mixed L4/L7 rules in one policy; root-ns policy + namespace override interplay.
- [ ] Built-in authz control pack: MG-AUTHZ-001..007 per SPEC §15, using `requires: [authorization.effective]`.
- [ ] Built-in mTLS completion: make MG-MTLS-002 evaluable from produced ports and ship MG-MTLS-007 for DestinationRule client/server TLS contradictions.
- [ ] E2E producer cutover: transition `port-level-override` and `dr-contradiction` from golden-unknown to golden-resolved, update their guards and notes, add the cases.tsv-driven `sidecar-authz/` fixture group with goldens, and prove deterministic output with two consecutive `make e2e` runs including both RBAC proofs.
- [ ] Chain assertions in every table case. `Version()` becomes a composite (current tags: `mtls/v2,authz/v1`) — document the scheme in the resolver package doc.

## Definition of Done
- All tables pass; flags-raised summary as in M2 (expected-output uncertainty goes to the human with Istio doc links first).
- Kind fixture `sidecar-authz/` added with golden findings; e2e green.
- Purity check green.

## Out of scope
Ambient ztunnel/waypoint HEALTH controls (M6) — this milestone models attachment/enforceability only.
