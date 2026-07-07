# M2 — Effective mTLS Resolver (pure function, full semantics)

Branch: `m2-mtls-resolver`

## Goal
Complete `ResolveMTLS` per contract: correct Istio precedence, port-level overrides, DestinationRule client-TLS contradiction detection, data-plane awareness, full chains. THE test tables are the specification.

## Context
SPEC.md §7 (mTLS effective posture). Contract: resolver_types.go. Upstream references to cite in code comments: Istio docs on PeerAuthentication precedence (mesh → namespace → workload → port-level) and automatic mTLS / DestinationRule TLS interaction.

## Deliverables
- [x] Precedence implementation: mesh-wide PA → namespace PA → workload-selector PA → portLevelMtls, UNSET inheritance handled correctly at each level.
- [x] `mixed-by-port` computation and ByPort map population.
- [x] DestinationRule interplay: client TLSMode DISABLE/SIMPLE against server STRICT ⇒ ClientTLSContradiction=true with both resources in the chain.
- [x] Data-plane awareness: not-in-mesh workloads ⇒ MTLSNotInMesh; ambient handled to the extent inputs allow (ztunnel Tristate; full ambient controls in M6).
- [x] Unknown propagation: MeshDefaults.Known=false or Unobserved inputs ⇒ MTLSUnknown with UnknownReason; never a guessed default.
- [x] Table-driven tests covering AT MINIMUM: mesh STRICT + ns PERMISSIVE; ns STRICT + workload DISABLE; port-level override in both directions; UNSET inheritance chains; DR contradiction; DR ISTIO_MUTUAL non-contradiction; no PA anywhere (mesh default); not-in-mesh; unknown mesh config; multiple selector PAs (Istio's documented tie-breaking).
- [x] Chain assertions in every table case: order, kinds, and effects — not just the final enum.
- [x] `Version()` returns `mtls/v1` semantics tag; bump rule documented in package doc.

## Definition of Done
- All table cases pass; provisional M1 resolver deleted; M1 outputs now flow from the real resolver.
- Any expected-output uncertainty was flagged to the human with an Istio docs link BEFORE being encoded in a table (list flags raised in the summary — "none" is a valid answer only with citations in comments).
- Purity check still green.

## Out of scope
Authorization resolution (M5), Sidecar-resource scoping (arrives with normalizer work in M5).
