# OpenMeshGuard Product Spec

Status: v2.1 design draft — §21 open questions resolved
Date: 2026-07-02
License: Apache 2.0
Audience: founder, early contributors, platform/security design partners

## Changes from v1

This revision incorporates a structured design review:

1. **"Verified" now means verified.** Runtime evidence from Istio telemetry is promoted into the MVP. Config-only analysis is labeled declared posture; telemetry-backed analysis is labeled verified posture.
2. **Effective posture resolution is now a named, first-class architectural component.** The product's core differentiation is computing what a workload's posture *actually is* after Istio's layered policy semantics resolve — not per-resource YAML linting.
3. **An honest comparison against existing OSS tooling is included.** The spec now answers "why not istioctl analyze + Kiali + Kyverno audit mode?"
4. **Multi-cluster correlation is cut from OSS v1.** v1 ships single-cluster scanning with multi-cluster *awareness*. Full cross-cluster posture correlation moves to a later phase.
5. **Environment classification and ownership are label/annotation-first**, with config files as overrides. Bespoke CSV formats are demoted. Exceptions become annotation-referenced records that live in Git.
6. **Controls are data, not Go code.** The rule engine evaluates CEL expressions over the normalized model. The control library becomes a community-contributable artifact.
7. **Workflow-state metrics are removed from the v1 report.** The first-run report only shows what the scanner can actually know.
8. **A community and OSS strategy section is added**: Apache 2.0, build-in-public timeline, contribution model, CNCF sandbox ambition, and automated Istio version-matrix maintenance.
9. **Control ID prefix changed from `OMG-` to `MG-`.**
10. Offline manifest scanning (`scan --local`) is added as an explicit, scoped roadmap item with degraded-confidence semantics.

## 1. Core Thesis

OpenMeshGuard helps enterprises move from assumed mesh security to verified Istio posture.

Istio can create a false sense of security when teams assume that adopting a service mesh automatically means zero trust. mTLS, identity, policy, and telemetry are only valuable when they are configured correctly, adopted consistently, tied to ownership, and backed by evidence.

The thesis has two halves, and both must be in the product from v1:

- **Declared posture**: what the resolved, effective Istio configuration says should be true.
- **Verified posture**: what mesh telemetry proves is actually happening on the wire.

A config-only scanner still produces assumed posture — just better-inspected. The claim "no plaintext is possible" comes from configuration. The claim "no plaintext was observed in the last 7 days" comes from telemetry. OpenMeshGuard reports both, side by side, and never conflates them.

OpenMeshGuard exists to answer:

> Is our service mesh actually secure, governed, owned, and producing evidence that enterprise control teams can trust?

## 2. Positioning

Primary positioning:

> Continuous governance and risk posture management for enterprise Istio environments.

Supporting message:

> Use Kyverno, OPA, or admission controls to block violations. Use OpenMeshGuard to understand whether your mesh is actually secure, governed, and backed by trustworthy evidence.

Category:

> Service Mesh Governance and Risk Posture Management

Initial beachhead:

> Istio Governance Posture Management

### 2.1 Why not existing tools

This is the first question every platform team will ask, so the answer belongs in the spec and the README.

| Tool | What it does well | What it does not do |
| --- | --- | --- |
| `istioctl analyze` | Config validity and common misconfiguration checks against one cluster or local files. | No effective-posture model, no governance context (ownership, exceptions, environments), no scoring, no evidence trail, no runtime verification, point-in-time only. |
| Kiali validations | Live visualization plus per-resource validation rules; good operator UX. | Validation is resource-oriented, not control-oriented; no audit evidence model, no exception lifecycle, no framework-aligned reporting; assumes Kiali/Prometheus deployment. |
| Kyverno / OPA Gatekeeper (audit mode) | Enforce or report on resource shape at admission or in background scans. | Rules see one resource at a time; they cannot resolve Istio's layered policy semantics (effective mTLS after mesh + namespace + workload + port-level PeerAuthentication and DestinationRule interplay, AuthorizationPolicy CUSTOM→DENY→ALLOW evaluation, Sidecar scoping, ambient L4/L7 split). No mesh-semantic model, no telemetry verification. |
| kubescape / kube-bench / checkov | Kubernetes and IaC hardening baselines. | Not mesh-aware beyond superficial checks; no Istio policy semantics. |
| Prometheus/Grafana Istio dashboards | Raw runtime signal. | No control model, no posture conclusion, no evidence packaging. |

**OpenMeshGuard's differentiation is the combination that none of these provide:**

1. An **effective posture resolver** that computes per-workload declared posture from Istio's real evaluation semantics.
2. **Runtime verification** of that posture from Istio telemetry.
3. A **governance layer** (ownership, environments, exceptions, evidence, scoring) that turns findings into control evidence enterprises can defend.

If a check can be trivially replicated by a per-resource Kyverno rule, it is not the core of this product. It may still exist in the control library, but the moat is resolution + verification + governance.

## 3. Users

Because this is an open source product first, the primary framing is users, not buyers.

Primary users (OSS):

- Platform engineers who own mesh enablement and want proof their rollout is real.
- Security engineers who need zero-trust and segmentation evidence.
- SREs who manage version skew, lifecycle risk, and upgrade readiness.

Secondary audiences (enterprise adoption path):

- Cloud Security Architecture, Enterprise Architecture, Technology Risk / Compliance, Network Security, Application Modernization leadership.

Natural early-adopter environments:

- Banks, insurance, healthcare, large retail, telco, government contractors — regulated environments where "prove it" is the default posture and where Istio adoption at scale is common.

## 4. Product Principles

1. Least-privilege, read-only first.
2. Findings must be high-signal and explainable. **False positives are the fastest way to lose app teams; effective-posture resolution exists to prevent them.**
3. Declared posture and verified posture are always distinguished, never blended.
4. Unknown is not failing. Missing evidence, missing permissions, and missing classification are reported as their own states.
5. Controls are data. The control library is a community artifact, not compiled logic.
6. Enterprise context (ownership, environment, exceptions) matters as much as raw YAML — but the tool must be useful with zero context files on first run.
7. Evidence should be exportable and audit-friendly.
8. Remediation should use existing enterprise workflows before direct mutation.
9. OpenMeshGuard must not become a generic dashboard.
10. The OSS version must be genuinely useful standalone, forever.

## 5. Non-Goals

OpenMeshGuard should not:

- Replace Kiali, istioctl, or mesh observability tooling.
- Run, install, or upgrade Istio.
- Become a traffic management plane.
- Replace Solo, Tetrate, OpenShift Service Mesh, or Kong Mesh.
- Replace Kyverno, OPA, Gatekeeper, or admission control (it complements enforcement with posture and evidence).
- Replace Prometheus, Grafana, Datadog, or OpenTelemetry (it consumes telemetry, it does not collect it).
- Claim compliance with NIST, PCI, HIPAA, or other frameworks. It produces framework-aligned evidence only.
- Mutate production clusters by default.

## 6. First Product Experience

The first report answers:

> Are you actually protected by the mesh?

The v1 report only contains metrics the scanner can actually know from cluster state, telemetry, and any supplied context. Workflow states (e.g., "reviewed by security") are enterprise-tier concepts and do not appear in OSS v1 output.

Example first-run summary (no context files supplied):

| Control area | Declared | Verified | Unknown |
| --- | --- | --- | --- |
| Strict mTLS (effective, per workload) | 71% of mesh workloads | 64% — no plaintext observed in 7d | 7% — no telemetry access |
| Explicit authorization coverage | 54% of mesh workloads | — (config-only control) | — |
| Default-deny posture | 22% of mesh namespaces | — | — |
| Public gateway wildcard hosts | 3 findings | — | — |
| Broad external egress | 11 ServiceEntries | — | — |
| Environment classification coverage | 61% of namespaces classified | — | 39% unclassified |
| Ownership metadata coverage | 69% of mesh resources | — | — |

Notes on this design:

- **Declared vs Verified vs Unknown are separate columns.** This is the product thesis made visible on the first screen.
- Environment classification coverage is itself a reported metric, because production-scoped controls only apply to namespaces the tool can classify (see §9).
- With no Prometheus access, the Verified column degrades to "unknown — no telemetry access" rather than silently disappearing.

## 7. Effective Posture Resolver

This is the named core of the product. Per-resource checks are commodity; resolution is not.

The resolver computes, per workload (and per port where relevant):

**mTLS effective posture**

- Merge mesh-wide, namespace, and workload-level PeerAuthentication, including port-level overrides, using Istio's precedence rules.
- Intersect with DestinationRule TLS settings that affect client-side behavior (e.g., `DISABLE`, `SIMPLE`, `ISTIO_MUTUAL`) to detect declared-strict-but-client-plaintext contradictions.
- Account for data plane mode: sidecar mTLS vs ambient ztunnel L4 mTLS, including workloads not enrolled in either.
- Output: one of `strict`, `permissive`, `disabled`, `mixed-by-port`, `not-in-mesh`, with the resolution chain attached as evidence.

**Authorization effective posture**

- Model AuthorizationPolicy evaluation order (CUSTOM → DENY → ALLOW) and the semantics of "no ALLOW policy present."
- Resolve policy attachment scope: mesh root namespace, namespace, workload selector, and (ambient) waypoint attachment via `targetRefs`.
- Distinguish L4-enforceable rules (ztunnel) from L7 rules that require a waypoint, and flag L7-requiring policies with no waypoint in the enforcement path as **not enforced** rather than "present."
- Output per workload: `default-deny + explicit allow`, `allow-only`, `no-policy`, `deny-present`, `waypoint-policy-unenforced`, with the evaluation chain as evidence.

**Scope resolution**

- Apply `Sidecar` resource scoping and `exportTo` semantics when computing reachability and policy visibility.
- Resolve Gateway and Kubernetes Gateway API exposure paths, including waypoint enrollment (`istio.io/use-waypoint`) and cross-namespace `ReferenceGrant`s.

**Resolver requirements**

- Every resolved conclusion carries its input chain (which resources, in which order, produced the result). This feeds the evidence model directly.
- The resolver is versioned against Istio minor releases; semantic changes in Istio's evaluation rules are tracked as resolver changes with tests.
- Resolver outputs are exposed as structured inputs to the rule engine so individual controls stay simple (see §12).

Kind acceptance fixtures must include resolution edge cases: port-level PeerAuthentication overrides, DestinationRule TLS contradictions, namespace-vs-workload policy conflicts, ambient L7 policies without waypoints, and Sidecar-scoped visibility.

## 8. Runtime Verification

Runtime evidence is in the MVP, not the roadmap. The thesis requires it.

v1 runtime-verified controls (Prometheus optional input, standard Istio proxy metrics):

| Control ID | Verification |
| --- | --- |
| MG-MTLS-101 | No plaintext requests observed to mesh workloads within the lookback window (via `istio_requests_total` / `istio_tcp_connections_opened_total` with `connection_security_policy="none"`). Primary use: proving whether PERMISSIVE-mode workloads are actually receiving plaintext before a strict cutover — or that declared-strict is corroborated by observed traffic. |
| MG-MTLS-102 | Share of observed inbound traffic per workload that was mutual TLS over the lookback window, reported as verified mTLS coverage. |

Rules:

- Runtime controls never overwrite declared posture; they corroborate or contradict it. A contradiction (declared strict, plaintext observed) is a critical finding of its own.
- Absence of traffic is reported as `no-traffic-observed`, not as a pass.
- Absence of Prometheus access degrades these controls to `unknown — no telemetry access` in the report and the permission summary.
- Lookback window is configurable; default 7 days.
- v1 supports Prometheus HTTP API with bearer token or mTLS client auth. Other telemetry backends are post-v1.

Query cost guardrails (decided):

- All queries aggregate server-side at workload granularity (`sum by (destination_workload_namespace, destination_workload, connection_security_policy)`) — never per-pod or per-source-principal cardinality.
- Range queries are bounded: default step 1h, per-query client timeout (default 30s), and per-namespace chunking on large meshes instead of one mesh-wide query.
- On query timeout or sample-limit failure, degrade to a 24h window with an explicit warning in the report; never fail the whole scan on telemetry cost.
- Document recommended recording rules for very large meshes; the scanner prefers them when present.

This is deliberately narrow: two controls, one metric family, huge demo value. "These 14 services actually received plaintext traffic last week" is the sentence that sells the thesis.

## 9. Environment Classification and Ownership

Production-scoped controls require knowing what production is. v1 defines an explicit strategy instead of assuming context files exist.

Classification precedence (highest wins):

1. Explicit mapping in the scan config file.
2. Documented label convention: `openmeshguard.io/environment` on the namespace, falling back to widely used conventions (`environment`, `env`) when enabled.
3. Optional name heuristics (`-prod`, `-production` suffixes) behind an explicit `--infer-environments` flag, disabled by default. Results carry `inferred` confidence and the report discloses that heuristic classification was used. (Decided: ships in v1, off by default.)
4. Otherwise: `unclassified`.

Behavior:

- Production-scoped controls evaluate only classified-production namespaces.
- Unclassified namespaces are never silently treated as production or silently skipped. Classification coverage is itself a reported governance metric, and unclassified namespaces get a dedicated finding.

Ownership precedence:

1. Resource/namespace labels and annotations: `app.kubernetes.io/name`, `app.kubernetes.io/part-of`, plus `openmeshguard.io/owner` and `openmeshguard.io/app-id`.
2. Scan config file mapping (namespace or selector → owner/app/BU).
3. Imported mapping file (CSV/YAML) as a last-resort bulk bridge from CMDB exports.

Rationale: labels and annotations already live in the manifests and flow through GitOps review. Bespoke files are overrides and bridges, not the primary model.

## 10. Exceptions

Exceptions are Git-native records, designed to flow through the same review process as the resources they cover.

- An exception is a YAML record: ID, control ID(s), scope (cluster/namespace/selector/resource), owner, approver, justification, ticket link, expiration, status.
- Resources reference exceptions via annotation: `openmeshguard.io/exception: <id>` (or the exception's scope selector matches them).
- Exception records live in the user's repo and are passed to the scanner (`--exceptions ./exceptions/`).
- Expired exceptions count as active risk (finding severity restored, plus an exception-hygiene finding).
- The scanner never treats an exception as deletion of a finding — the finding remains in output, marked excepted, with the exception attached as evidence.

Enterprise-tier ideas (approval workflows, ticket integration, expiry escalation routing) stay out of OSS v1, but the record format is designed so those layers consume the same schema later.

## 11. MVP Scope

MVP name:

> OpenMeshGuard Community

MVP goal:

> A platform engineer can run OpenMeshGuard against an Istio cluster and within five minutes see resolved, per-workload mesh security posture — declared and, where telemetry exists, verified — with zero context files.

Planned commands:

```bash
openmeshguard scan --context prod-cluster --all-namespaces
openmeshguard scan --kubeconfig ./kubeconfig --namespace payments-prod
openmeshguard scan --context prod-cluster --prometheus-url https://prom.example.com
openmeshguard report --format html --output openmeshguard-report.html
openmeshguard export --format json --output findings.json
openmeshguard export --format sarif --output openmeshguard.sarif
openmeshguard score --namespace payments-prod
```

MVP inputs:

- kubeconfig or in-cluster service account (read-only)
- Kubernetes API, Istio CRDs, Gateway API CRDs where used
- Optional Prometheus endpoint (enables verified controls)
- Optional scan config file (classification, ownership, control overrides)
- Optional exception records

MVP outputs:

- Mesh inventory (normalized)
- Resolved effective posture per workload
- Findings with evidence chains
- Declared/verified/unknown posture summary
- Score summary
- Remediation guidance with suggested YAML
- Canonical JSON, SARIF 2.1.0, static HTML report

MVP scope decisions:

- **Single-cluster scanning with multi-cluster awareness.** The scanner detects signals that the cluster participates in a multi-cluster mesh (east-west gateway deployments/services, `topology.istio.io/network` labels, mesh network configuration where readable) and reports: "this cluster appears to participate in a multi-network mesh; cross-cluster posture is not evaluated in this version." No cross-cluster correlation, no remote-secret inspection (which would violate the no-secrets permission stance anyway). Full multi-cluster correlation is a later phase and likely wants the scheduled/agent model rather than a one-shot CLI.
- **Sidecar and ambient are both first-class in v1**, including mixed mode, because ambient adoption is exactly when posture questions (ztunnel coverage, waypoint enrollment, L4/L7 enforcement gaps) are most acute.
- **`scan --local ./manifests` (offline mode) is post-v1 but designed for now.** Offline scanning of rendered manifests broadens adoption (PR-time shift-left, matches the SARIF/CI story) but cannot resolve full effective posture without cluster state. When shipped, offline findings carry `static-only` confidence and the report states which controls were skipped or degraded. The rule engine and normalized model must not assume a live cluster, so this stays cheap to add.

## 12. Implementation Decisions

- Scanner core and CLI: Go.
- **Rule engine: controls are data.** Each control is a YAML document: metadata (ID, title, severity, category, environments, framework tags) plus a CEL expression evaluated against the normalized inventory and resolver outputs. The built-in library ships as embedded data files; users can supply additional control packs and override severity/baselines per environment. This makes the control library community-contributable without touching Go, and makes custom enterprise controls a first-class OSS feature.
- The effective posture resolver is Go (it encodes Istio semantics and needs versioned tests), and its outputs are structured CEL inputs so controls stay one-liners where possible.
- OSS v1 report: static HTML generated from canonical JSON, no server. Future richer UI: TypeScript.
- Canonical output: OpenMeshGuard JSON is the product schema (inventory, resolver outputs, findings, evidence, scores, permission summary). SARIF 2.1.0 is a compatibility export, not the internal model. The JSON schema is published and versioned from v1.
- Framework mapping: defensible control tags only (NIST CSF 2.0, NIST SP 800-53 families, OWASP Kubernetes Top 10, CIS where relevant), never compliance claims.
- Drift/source traceability: deferred beyond v1 (deployed-state posture only). Design unchanged from v1 spec §12.6 semantics: missing source metadata is not automatically noncompliant unless the org baseline says so.

Client strategy:

- Typed Kubernetes, Istio, and Gateway API clients for supported upstream resources; discovery to detect installed API groups/versions/CRDs; dynamic/unstructured access only as compatibility fallback.
- No long-running watches, broad polling, or high-cardinality per-object calls in the CLI scan path. List-based collection, bounded concurrency, client-side normalization.
- Namespace-scoped scans supported to reduce blast radius and API load.
- Partial evidence reported when permissions or APIs are unavailable — never silent retry-with-broader-access.

## 13. Cluster Access and Least Privilege

Unchanged in substance from v1 (this section survived review intact); summarized here as binding requirements:

- Published RBAC profiles before implementation is "done": namespace-scan Role, cluster-scan ClusterRole, and optional evidence add-ons (nodes, events, control-plane ConfigMaps, Prometheus, vendor APIs).
- Baseline permissions: `get`/`list` on core (namespaces, pods, services, endpointslices), apps (deployments, replicasets, statefulsets, daemonsets), Istio networking/security/telemetry CRDs, and Gateway API resources.
- Explicitly never required for OSS v1: any write verbs, secrets access, token creation, exec/attach/portforward, impersonation, cluster-admin, or `watch`.
- Every report includes a permission/evidence summary: which permissions were present, which evidence was unavailable, which findings were affected.
- Documentation answers, per permission: why it is needed, what degrades without it, and whether it is optional.

## 14. Architecture Map

Community architecture:

```text
Kubernetes / Istio / Gateway APIs        Prometheus (optional)
        |                                       |
   Collectors (typed clients, read-only)   Telemetry queries
        |                                       |
        +---------------+-----------------------+
                        |
                  Normalizer
                        |
        Effective Posture Resolver  (Go, versioned vs Istio semantics)
                        |
        Rule Engine  (CEL over normalized model + resolver outputs;
                      controls = YAML data, built-in + user packs)
                        |
     Findings + Scores + Evidence chains + Permission summary
                        |
        Canonical JSON  ->  CLI / SARIF / Static HTML report
```

Enterprise architecture (directional, unchanged):

```text
Cluster Agents / Scheduled Scans
        |
Enterprise Control Plane (multi-cluster correlation, history)
        |
Inventory + Findings + Ownership + Exceptions + Evidence
        |
Dashboards + Reports + Tickets + Pull Requests + GRC Exports
```

## 15. Initial Control Library

Prefix changed to `MG-`. Each control declares its evidence type: `config` (declared posture), `runtime` (verified posture), or `context` (requires classification/ownership/exception input).

### Category A: mTLS Assurance

| Control ID | Type | Control |
| --- | --- | --- |
| MG-MTLS-001 | config | Production mesh-managed workloads must resolve to effective strict mTLS. |
| MG-MTLS-002 | config | Every declared, Service-bound workload port must resolve to strict mTLS. |
| MG-MTLS-003 | config | Workloads must never resolve to globally disabled mTLS. |
| MG-MTLS-004 | context | Namespaces transitioning to strict mTLS must have migration status and owner. |
| MG-MTLS-005 | config | Ambient namespaces must have explicitly validated L4 mTLS posture. |
| MG-MTLS-006 | config | Ambient workloads must have healthy ztunnel coverage on every scheduled node. |
| MG-MTLS-007 | config | Client TLS configuration must not contradict resolved server mTLS. |
| MG-MTLS-101 | runtime | No plaintext traffic observed to mesh workloads within the lookback window. |
| MG-MTLS-102 | runtime | Verified mTLS share of observed inbound traffic per workload. |

### Category B: Authorization / Zero Trust

| Control ID | Type | Control |
| --- | --- | --- |
| MG-AUTHZ-001 | config | Mesh-enabled production workloads must be covered by resolved AuthorizationPolicy. |
| MG-AUTHZ-002 | config | Production namespaces should resolve to default-deny plus explicit allow. |
| MG-AUTHZ-003 | config | Policies must not allow broad access (`{}` rules, wildcard principals) without exception. |
| MG-AUTHZ-004 | config | Access must scope to approved principals, namespaces, or workloads. |
| MG-AUTHZ-005 | config | Coverage is evaluated at resolved workload level, never namespace-resource presence. |
| MG-AUTHZ-006 | config | Ambient L7 authorization requirements must resolve to waypoint-enforced attachment. |
| MG-AUTHZ-007 | config | L7-requiring policies without a waypoint in the enforcement path are reported as not enforced. |

### Category C: Exposure and Boundary Control

| Control ID | Type | Control |
| --- | --- | --- |
| MG-GW-001 | config | Public gateways must not use wildcard hosts. |
| MG-GW-002 | context | Gateway routes must map to owners. |
| MG-GW-003 | config | Production traffic must not route to non-production services (requires classification). |
| MG-GW-004 | context | Gateway API resources used by Istio must map to owners. |
| MG-GW-005 | config | Waypoint scope/enrollment must be explicit where L7 policy is required. |
| MG-EGRESS-001 | config | ServiceEntries must not allow broad external egress without exception. |
| MG-EGRESS-002 | context | External access must map to owner and justification. |

### Category D: Governance and Ownership

| Control ID | Type | Control |
| --- | --- | --- |
| MG-OWN-001 | context | Mesh resources must map to an application owner. |
| MG-OWN-002 | context | Production mesh resources must include app ID, environment, and owner metadata. |
| MG-ENV-001 | context | Namespaces must be classifiable to an environment; unclassified is a governance finding. |
| MG-EXC-001 | context | Exceptions must include approver, justification, ticket, and expiration. |
| MG-EXC-002 | context | Expired exceptions count as active risk and are escalated in the report. |

### Category E: Lifecycle and Advanced Risk

| Control ID | Type | Control |
| --- | --- | --- |
| MG-VER-001 | config | Sidecar proxy versions must match approved baseline. |
| MG-VER-002 | config | Control plane versions must be within the upstream Istio support window. |
| MG-VER-003 | config | ztunnel and waypoint versions must match approved baseline where ambient is enabled. |
| MG-VER-004 | config | Control-plane/data-plane skew must follow upstream Istio support rules. |
| MG-EF-001 | config | EnvoyFilter usage requires approval and exception. |
| MG-UPG-001 | config | Upgrade blockers identified per app/team. |

## 16. Scoring Model

Scores must be understandable, defensible, and hard to argue into uselessness.

- Scoring configuration ships as data (same mechanism as controls): published default weights, overridable per environment.
- Default dimensions and weights (draft): effective mTLS 25, authorization coverage 25, exposure controls 15, egress controls 10, ownership metadata 10, exception hygiene 10, lifecycle baseline 5.
- Alongside the numeric score, report **letter grades per control category** (A–F on pass rates) — categories are harder to game and easier to communicate upward than a blended 0–100.
- Critical findings cap the affected resource's score.
- Expired exceptions count as active risk.
- Runtime-verified contradictions (declared strict, plaintext observed) are always critical.
- Scores always link to the evidence and resolver chains that produced them.
- Rollups: namespace → cluster in OSS. Fleet rollups are an enterprise-tier concern.

## 17. Evidence Model

Every finding includes enough evidence for a platform or security engineer to understand what was observed, what was resolved, and what was inferred.

Minimum finding fields:

- Finding ID, Control ID, severity, evidence type (config/runtime/context)
- Affected cluster, namespace, workload, service, or resource
- Data plane mode: sidecar, ambient, mixed, unknown, not applicable
- Evidence source: Kubernetes API, Istio CRD, Gateway API, Prometheus, config file, exception record
- Evidence confidence: observed, resolved, inferred, user-supplied, unavailable
- **Resolution chain** (for resolver-derived findings): the ordered resources and rules that produced the effective posture
- Resource references, reasoning summary, remediation guidance, exception status

Reports separate: confirmed risk, declared-vs-verified contradictions, out-of-support lifecycle posture, missing evidence, unknown posture (inputs unavailable), and organization-policy gaps (missing owner/classification/exception metadata).

Later (enterprise/GRC track): OSCAL export and signed report attestation (e.g., cosign) for audit-grade evidence handling. Not v1, but the canonical JSON schema should not preclude either.

## 18. Community and OSS Strategy

- **License: Apache 2.0** (required for any future CNCF path; expected by the ICP's OSS review boards).
- **Design privately, build in public.** The repo stays private only until the spec, schema draft, and skeleton are coherent — target opening the repo at the Phase 1 prototype stage, before polish. Early issues and design docs in the open are the contribution funnel.
- **The control library is the community surface.** Because controls are CEL + YAML, contributing a control does not require Go. Contribution docs, a control-pack format, and CI validation of contributed controls are v1 deliverables, not afterthoughts.
- Governance: single-maintainer initially, documented; adopt a lightweight MAINTAINERS/CONTRIBUTING/CoC set at repo opening. CNCF sandbox is a plausible 12–18 month ambition if adoption warrants — Apache 2.0, vendor-neutral naming, and public roadmap keep that door open.
- **Istio version-matrix automation is mandatory, not aspirational.** CI tracks upstream supported Istio releases (scheduled job regenerating the support matrix and running Kind fixtures against new minors). Without automation this becomes an unmaintained-scanner graveyard.
- Name check: no direct "OpenMeshGuard" collision found as of this draft; adjacent names (OpenMesh graphics library, Openmesh Network, Open-Mesh Wi-Fi) are unrelated spaces. Reserve the GitHub org and domain before opening the repo.

## 19. Roadmap

### Phase 0: Foundation (private)

- Product spec (this document), canonical JSON schema draft, control YAML/CEL format draft
- Least-privilege RBAC profiles drafted
- Name/org/domain reservations, Apache 2.0 LICENSE, README

### Phase 1: OSS Scanner Prototype (repo opens during this phase)

- Go scanner core and CLI; typed-first client strategy with discovery fallback
- Normalized inventory model; sidecar and ambient mode detection
- **Effective posture resolver v1: mTLS and authorization resolution with evidence chains**
- Rule engine (CEL) + first control pack as embedded data
- Canonical JSON output; least-privilege RBAC manifests
- Multi-cluster awareness signals (detection + honest "not evaluated" reporting)

### Phase 2: Report and Verification

- Static HTML report with declared/verified/unknown summary
- **Prometheus integration and runtime controls MG-MTLS-101/102**
- Score summary with category grades; SARIF export; CI mode
- Permission/evidence summary in every report
- Kind acceptance fixtures: sidecar, ambient, mixed mode, resolver edge cases

### Phase 3: Governance Context

- Classification and ownership precedence (labels/annotations → config → import file)
- Exception record format + annotation referencing
- Control packs: user-supplied controls, severity/baseline overrides per environment
- Contribution pipeline for community controls

### Phase 4: Shift-Left and Traceability

- `scan --local` offline manifest mode with static-only confidence semantics
- Source metadata detection (GitOps/Helm/CI/managedFields) and drift model (missing source ≠ noncompliant unless baseline requires it)

### Phase 5: Multi-Cluster and Distribution Validation

- Two-cluster Kind multi-primary multi-network fixture; cross-cluster correlation design (likely agent/scheduled model, feeds enterprise architecture)
- **First post-Kind validation track: OpenShift Service Mesh on a disposable ROSA lab** — validates Operator-managed Istio resources, SCC/RBAC effects on scanner permissions, OpenShift networking, and OpenShift Gateway API/ambient caveats, while standing up reusable AWS lab tooling
- Managed Kubernetes harness with upstream Istio as a later conformance track: EKS first (reuses the AWS account/automation built for ROSA), then GKE and AKS
- Solo/Gloo validation when licensing and lab access justify it; Tetrate deferred until demand justifies it

### Phase 6: Enterprise Design Partner Surface

- Multi-cluster ingestion, exception lifecycle workflow, ticket/PR integrations, evidence pack / GRC exports (OSCAL candidate), fleet rollups

## 20. OSS v1 Ready Definition

OpenMeshGuard Community v1 is ready when:

- The Go CLI scans a namespace or cluster using only the documented least-privilege `get/list` RBAC profile, with no write operations attempted (proven in validation) and bounded API usage.
- The effective posture resolver produces per-workload mTLS and authorization posture with attached resolution chains, validated against Kind fixtures covering the documented edge cases (port-level overrides, DestinationRule contradictions, ambient L7-without-waypoint, Sidecar scoping).
- With a Prometheus endpoint supplied, runtime controls MG-MTLS-101/102 produce verified posture; without it, they degrade to explicit unknown states.
- The first-run report is useful with zero context files, and classification/ownership coverage are honestly reported as governance metrics.
- Controls are loaded as data (built-in pack + user packs), and a documented format exists for contributing controls.
- Canonical JSON (published schema), SARIF export, and static server-less HTML render from the same output.
- Kind acceptance fixtures install and scan real upstream Istio sidecar, ambient, and mixed topologies; multi-cluster participation is detected and honestly reported as not evaluated.
- OSS v1 requires no GitOps adoption, no repository access, no vendor APIs, no secrets access, no watch permissions, and no write verbs.
- Documentation covers permissions and degradation, the declared/verified/unknown model, the control format, and the OSS/enterprise boundary.

## 21. Decisions

Resolved 2026-07-02:

- **Rule engine: CEL.** Lighter Go embedding, aligns with Kubernetes direction (ValidatingAdmissionPolicy), keeps control authoring approachable without a policy-language learning curve. Rego interop can be revisited if contributor demand materializes.
- **Prometheus defaults and guardrails: defined in §8.** 7-day default lookback; workload-granularity aggregation only; bounded, chunked range queries; graceful degradation to a 24h window with an explicit report warning.
- **Classification heuristics: ship in v1, off by default.** Behind `--infer-environments`, `inferred` confidence, disclosed in the report (§9).
- **Distribution validation: ROSA first.** The first post-Kind track is OpenShift Service Mesh on a disposable ROSA lab — it exercises the hardest platform deltas (Operator-managed resources, SCC/RBAC), matches the initial adopter profile, and is the simplest OpenShift lab to provision and tear down. Note the scope boundary: ROSA validates the OpenShift/OSSM track, not upstream-Istio-on-managed-Kubernetes conformance. That remains a separate, later track and starts with EKS to reuse the AWS account and automation already built for ROSA (§19 Phase 5).
- **`score` remains a standalone command in v1**, reading the same canonical JSON as `report`, so it adds no separate data path.

No open design questions remain blocking Phase 1.
