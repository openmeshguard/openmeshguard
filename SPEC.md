# OpenMeshGuard Product Spec

Status: v1 refinement draft  
Date: 2026-07-01  
Audience: founder, early contributors, platform/security design partners

## 1. Core Thesis

OpenMeshGuard helps enterprises move from assumed mesh security to verified Istio posture.

Istio can create a false sense of security when teams assume that adopting a service mesh automatically means zero trust. mTLS, identity, policy, and telemetry are only valuable when they are configured correctly, adopted consistently, tied to ownership, and backed by evidence.

OpenMeshGuard exists to answer:

> Is our service mesh actually secure, compliant, owned, and operating according to enterprise controls?

## 2. Positioning

Primary positioning:

> Continuous governance and risk posture management for enterprise Istio environments.

Supporting message:

> Use Kyverno, OPA, or admission controls to block violations. Use OpenMeshGuard to understand whether your mesh is actually secure, governed, and compliant across every cluster.

Category:

> Service Mesh Governance and Risk Posture Management

Initial beachhead:

> Istio Governance Posture Management

Long-term expansion can include vendor-validated Istio distributions, Envoy-based service connectivity, Gateway API posture, Cilium service mesh posture, Linkerd, Kuma, and broader service-to-service security governance. The wedge stays focused on posture, evidence, ownership, exception, drift, and lifecycle management.

## 3. Buyers and Users

Primary buyer:

- Platform Engineering / Kubernetes Platform Team

Secondary buyers:

- Cloud Security Architecture
- Enterprise Architecture
- Technology Risk / Compliance
- Network Security
- SRE / Production Engineering
- Application Modernization Leadership

Primary users:

- Platform engineers who own mesh enablement.
- Security engineers who need zero-trust and segmentation evidence.
- SREs who manage version skew, drift, and upgrade readiness.
- Risk and compliance teams who need audit-ready control evidence.

Initial ICP:

- Banks
- Insurance
- Healthcare
- Large retailers
- Telcos
- Government contractors

## 4. Product Principles

1. Least-privilege, read-only first.
2. Findings must be high-signal and explainable.
3. OpenMeshGuard must not become a generic dashboard.
4. Enterprise context matters as much as raw YAML.
5. Evidence should be exportable and audit-friendly.
6. Remediation should use existing enterprise workflows before direct mutation.
7. The OSS version must be useful without the enterprise control plane.

## 5. Non-Goals

OpenMeshGuard should not start by:

- Replacing Kiali.
- Running or installing Istio.
- Becoming a full traffic management plane.
- Replacing Solo, Tetrate, OpenShift Service Mesh, or Kong Mesh.
- Replacing Kyverno, OPA, Gatekeeper, or admission control.
- Replacing Prometheus, Grafana, Datadog, or OpenTelemetry.
- Claiming full compliance with NIST, PCI, HIPAA, or other frameworks.
- Mutating production clusters by default.

## 6. First Product Experience

The first report should answer:

> Are you actually protected by the mesh?

Example summary:

| Control area | Status |
| --- | --- |
| Strict mTLS enforced | 67% of production namespaces |
| Explicit authorization | 54% of mesh-enabled production namespaces |
| Default-deny posture | 22% of production namespaces |
| Gateway exposure reviewed | 81% of public routes |
| Broad egress controlled | 73% of ServiceEntries |
| Owner metadata present | 69% of Istio resources |
| Exceptions valid | 58% valid, 42% missing or expired |

This report should lead with verified posture, not generic resource counts.

## 7. MVP Scope

MVP name:

> OpenMeshGuard Community

MVP goal:

> A platform engineer can run OpenMeshGuard against an Istio cluster and immediately get a useful governance posture report.

Planned commands:

```bash
openmeshguard scan --context prod-cluster --all-namespaces
openmeshguard scan --kubeconfig ./kubeconfig --namespace payments-prod
openmeshguard report --format html --output openmeshguard-report.html
openmeshguard export --format json --output findings.json
openmeshguard export --format sarif --output openmeshguard.sarif
openmeshguard score --namespace payments-prod
```

MVP inputs:

- kubeconfig or in-cluster service account
- Kubernetes API
- Istio CRDs
- Gateway API CRDs where used by Istio ambient, waypoints, or Kubernetes Gateway
- Optional Prometheus
- Optional app ownership CSV or YAML mapping

MVP outputs:

- Mesh inventory
- Findings
- Posture score
- Evidence records
- Remediation suggestions
- JSON export
- SARIF export
- Local HTML report

## 8. Implementation Decisions

OpenMeshGuard Community v1 should be built as a lean Kubernetes-native CLI.

Decisions:

- Scanner core and CLI: Go.
- OSS v1 report: static HTML generated from OpenMeshGuard JSON findings.
- Future richer local or enterprise UI: TypeScript.
- Custom control configuration: YAML.
- Framework mapping: defensible control tags only, not compliance claims.
- Drift/source posture for v1: Level 0 deployed-state posture only. Do not require GitOps, source traceability, repository access, or manifest comparison for OSS v1.

Client strategy:

- Use typed Kubernetes, Istio, and Gateway API clients for supported upstream resources where practical.
- Normalize typed API objects into OpenMeshGuard-owned inventory and finding models.
- Use discovery to detect installed API groups, versions, missing CRDs, and provider-specific resource availability.
- Keep dynamic/unstructured access as a compatibility fallback for unknown, optional, or vendor-specific resources, not as the main scanner path.
- Avoid long-running watches, broad polling, or high-cardinality per-object API calls in the CLI scan path.

Performance expectations:

- Prefer list-based collection with bounded concurrency and client-side normalization.
- Reuse informers/caches only if they reduce API pressure for the CLI execution model.
- Support namespace-scoped scans to reduce blast radius and API load.
- Report partial evidence when permissions or APIs are unavailable instead of retrying aggressively or requiring broader access.

Canonical output:

- OpenMeshGuard JSON is the canonical product schema for inventory, findings, evidence, scores, and reports.
- SARIF 2.1.0 is a compatibility export for CI/code-scanning consumers, not the internal findings model.
- The JSON schema must remain stable enough for static HTML reports, future TypeScript UI, enterprise ingestion, and third-party tooling.

Validation strategy:

- OSS v1 validation must use disposable Kind clusters with real upstream Istio deployments.
- Static YAML samples can be used for focused unit tests, but they are not sufficient proof that OpenMeshGuard supports a mesh mode.
- Acceptance fixtures should install and scan real sidecar, ambient, mixed sidecar/ambient, and multi-cluster topologies.
- The required upstream multicluster fixture is a two-cluster Kind multi-primary topology.
- The multicluster fixture should model `cluster1` and `cluster2` as separate primary clusters, each with its own Istio control plane, cluster name, network identity, east-west gateway, and remote-secret based endpoint discovery.
- The default multicluster fixture should use the multi-primary multi-network pattern because it validates cross-cluster gateway posture and matches the upstream ambient multicluster installation path.
- Validation should assert both scanner behavior and API impact: expected findings, expected unknown/missing-evidence behavior, bounded API calls, and no write operations.
- Distribution validation for OpenShift Service Mesh, Tetrate, and Solo should follow the same standard when feasible: real disposable or lab deployments first, sanitized exports only as supplemental regression cases.

## 9. OSS v1 Support Contract

OpenMeshGuard Community v1 supports upstream Istio first.

Support means:

- The scanner can discover and normalize Istio resources from supported upstream Istio releases.
- The scanner can evaluate both sidecar and ambient data plane modes.
- The scanner can report single-cluster and multi-cluster posture where the relevant Istio resources and cluster context are available.
- The scanner can distinguish supported, unsupported, unknown, and partially observable posture instead of collapsing every gap into a security failure.
- The scanner does not require vendor APIs, GitOps adoption, or the OpenMeshGuard Enterprise control plane.

Istio version policy:

- OpenMeshGuard Community v1 tracks the upstream Istio support window.
- The supported minor version range must be derived from Istio's current supported releases before every OpenMeshGuard release.
- Unsupported Istio versions should still be scanned on a best-effort basis, but lifecycle findings must clearly mark them as outside the supported upstream Istio window.
- Control-plane/data-plane skew must be reported according to upstream Istio rules.

Data plane mode policy:

| Mode | v1 expectation |
| --- | --- |
| Sidecar | Fully supported for upstream Istio versions in the supported release window. |
| Ambient | Fully supported for upstream Istio versions in the supported release window, including ztunnel, waypoint, and L4/L7 policy distinction. |
| Mixed sidecar and ambient | Supported where upstream Istio supports the topology; findings must identify unsupported interoperability or partial observability. |
| Multi-cluster sidecar | Supported where upstream Istio resources and cluster relationships can be observed or supplied by configuration. |
| Multi-cluster ambient | Supported where upstream Istio supports the topology; findings must preserve upstream feature-stage and limitation context. |

Provider and distribution policy:

| Environment | v1 stance |
| --- | --- |
| Upstream Istio | First-class OSS v1 target. |
| Red Hat OpenShift Service Mesh | Validate compatibility through Kubernetes, Istio, and Gateway API resources. Add OpenShift-specific checks only when the normalized model is insufficient. |
| Tetrate | Validate compatibility through Kubernetes, Istio, and Gateway API resources. Add Tetrate APIs only when needed for ownership, lifecycle, or policy context that is not visible through OSS APIs. |
| Solo / Gloo Mesh / Solo Enterprise for Istio | Validate compatibility through Kubernetes, Istio, and Gateway API resources. Add Solo APIs only when needed for ownership, lifecycle, or policy context that is not visible through OSS APIs. |

Provider support is not a promise to replace those platforms. The goal is to prove whether OpenMeshGuard can govern the Istio posture they deploy. Vendor-specific APIs are roadmap extensions, not OSS v1 prerequisites.

## 10. Cluster Access and Least Privilege

Least privilege is a product requirement, not only an implementation detail.

OpenMeshGuard Community v1 must publish a clear RBAC profile before implementation is considered complete. Enterprises should be able to answer:

- What permissions does the scanner need?
- Why does it need each permission?
- Which findings are unavailable when a permission is not granted?
- Which permissions are optional?
- Which permissions are explicitly not required?

Access principles:

- Default to read-only Kubernetes API access.
- Support namespace-scoped scans with a Role where cluster-wide visibility is not approved.
- Support all-namespace or multi-namespace scans with a narrowly scoped ClusterRole.
- Never require write verbs for OSS v1 scanning.
- Never require access to Kubernetes Secret values for baseline posture.
- Degrade findings to `unknown` or `missing evidence` when permissions are absent instead of requesting broader access silently.
- Include RBAC manifests and a permission explanation in published documentation.

Minimum baseline permissions for useful OSS v1 scans:

| API area | Resources | Verbs | Why |
| --- | --- | --- | --- |
| Kubernetes core | namespaces, pods, services, endpoints, endpointslices | get, list | Map namespaces, workloads, services, service discovery, and selected pods to mesh posture. |
| Kubernetes apps | deployments, replicasets, statefulsets, daemonsets | get, list | Resolve workload ownership and detect sidecars, ztunnel coverage, waypoint deployments, and rollout context. |
| Istio networking | gateways, virtualservices, destinationrules, serviceentries, sidecars, envoyfilters, workloadentries, workloadgroups | get, list | Evaluate traffic routing, ingress, egress, sidecar scoping, VM/workload entries, and advanced Envoy customization risk. |
| Istio security | peerauthentications, authorizationpolicies, requestauthentications | get, list | Evaluate mTLS posture, authorization coverage, default-deny, and identity policy. |
| Istio telemetry/extensions | telemetry, wasmplugins | get, list | Identify telemetry and extension posture where installed. |
| Gateway API | gatewayclasses, gateways, httproutes, grpcroutes, tcproutes, tlsroutes, referencegrants | get, list | Evaluate Kubernetes Gateway API exposure, waypoint attachment, route ownership, and cross-namespace references. |

Optional permissions:

| Permission | Why it is optional |
| --- | --- |
| nodes get/list | Improves ambient ztunnel node coverage and node-label context, but ztunnel posture can still be partially inferred from DaemonSets and pods. |
| events get/list | Improves troubleshooting evidence for failing ztunnel, waypoint, and control-plane resources. |
| configmaps get/list in Istio control-plane namespaces | Improves meshConfig and revision analysis, but should be scoped to known control-plane namespaces where possible. |
| pods/log get | Useful only for diagnostics; not required for baseline posture. |
| Prometheus read access | Improves traffic and policy-use evidence; not required for static configuration posture. |
| Vendor API read access | Enables richer OpenShift, Tetrate, or Solo context only when normalized Kubernetes/Istio/Gateway resources are insufficient. |

Explicitly not required for OSS v1 baseline scans:

- create, update, patch, delete, deletecollection, bind, escalate, or impersonate verbs
- secrets get/list/watch
- serviceaccounts/token create
- pods/exec, pods/attach, pods/portforward
- admission webhook mutation privileges
- cluster-admin
- watch for one-shot OSS v1 CLI scans

Published RBAC should come in three profiles:

| Profile | Purpose |
| --- | --- |
| Namespace scan Role | Lowest-friction app/team scan for one namespace, with reduced cluster-scope evidence. |
| Cluster scan ClusterRole | Read-only all-namespace scan for platform/security teams. |
| Optional evidence ClusterRole add-ons | Narrow opt-ins for nodes, events, control-plane ConfigMaps, Prometheus, or vendor evidence. |

Every report must include a permission/evidence summary showing which permissions were present, which evidence was unavailable, and which findings were affected.

## 11. Architecture Map

Community architecture:

```text
Kubernetes / Istio / Gateway APIs
        |
OpenMeshGuard Scanner
        |
Normalizer
        |
Rule Engine
        |
Findings + Scores + Evidence
        |
CLI / Local Dashboard / JSON / SARIF / HTML Report
```

Enterprise architecture:

```text
Cluster Agents / Scheduled Scans
        |
OpenMeshGuard Enterprise Control Plane
        |
Inventory + Findings + Ownership + Exceptions + Evidence
        |
Dashboards + Reports + Tickets + Pull Requests + GRC Exports
```

## 12. Core Modules

### 12.1 Mesh Inventory

Discovers:

- Clusters
- Namespaces
- Workloads
- Services
- Gateways
- Kubernetes Gateway API resources where used by Istio
- VirtualServices
- DestinationRules
- PeerAuthentications
- AuthorizationPolicies
- ServiceEntries
- EnvoyFilters
- Sidecar proxy versions
- Istio control-plane versions
- Istio revisions and control-plane/data-plane skew
- Ambient mesh labels and configuration
- ztunnel DaemonSets, versions, readiness, and node coverage
- Waypoint proxies, GatewayClass usage, enrollment labels, and target scope
- Multi-cluster topology hints, trust domains, networks, east-west gateways, and remote secrets where observable
- Owner, app, environment, and repo metadata

Goal:

> Know what exists, where it exists, and who owns it.

### 12.2 Security Posture Assessment

Evaluates:

- mTLS mode
- Strict vs permissive posture
- Plaintext acceptance risk
- AuthorizationPolicy coverage
- Default-deny posture
- Overly broad allow policies
- Gateway exposure
- Wildcard hosts
- Broad egress
- Cross-namespace routing
- Cross-environment routing
- EnvoyFilter risk
- ztunnel health and coverage
- Waypoint presence, scope, and enrollment
- Ambient L4 vs L7 policy enforcement semantics
- Ambient and Kubernetes NetworkPolicy interactions
- Sidecar/ambient interoperability risks
- Multi-cluster trust, network, and gateway posture where observable

Goal:

> Validate that Istio security is real, not assumed.

### 12.3 Governance Posture

Evaluates:

- App owner mapping
- Business unit mapping
- Environment classification
- Onboarding status
- Approved pattern usage
- Exception presence
- Exception expiration
- Change ticket linkage
- Ownership gaps

Goal:

> Make mesh risk accountable.

### 12.4 Compliance and Control Mapping

Maps technical findings to:

- OpenMeshGuard Enterprise Mesh Security Baseline
- OWASP Kubernetes Top 10
- NIST Cybersecurity Framework 2.0
- NIST SP 800-53 families
- CIS Kubernetes Benchmark where relevant
- Internal enterprise controls

Constraint:

> OpenMeshGuard provides framework-aligned evidence. It does not make an enterprise compliant by itself.

Goal:

> Convert technical mesh findings into control evidence.

### 12.5 Exception Management

Tracks:

- Exception owner
- Approver
- Business justification
- Affected resources
- Linked control
- Linked finding
- Expiration date
- Linked ticket or change record
- Current status
- Historical evidence

Goal:

> Make risk exceptions visible, time-bound, and auditable.

### 12.6 Source Traceability and Drift Detection

Deferred beyond OSS v1 deployed-state posture.

Detects:

- Istio resources with no declared source of record
- In-cluster config differing from a provided source of record
- Manual changes
- Missing GitOps, Helm, CI/CD, platform, or ownership metadata where the enterprise expects it
- Resources deployed outside approved pipelines where the enterprise declares those pipelines
- Policy changes without change records
- Field ownership changes from Kubernetes managedFields where available
- Drift between live resources and supplied local manifests or repositories when the user opts into repository inspection

Goal:

> Prove whether mesh configuration has accountable source, ownership, and change evidence without assuming every enterprise uses GitOps.

Constraint:

> OpenMeshGuard should not reinvent GitOps. It should consume source-of-record evidence when provided, detect missing or inconsistent traceability, and make drift explainable. A resource with no GitOps metadata is not automatically noncompliant unless the organization's baseline requires GitOps.

### 12.7 Evidence and Reporting

Produces:

- Executive posture summary
- Control coverage report
- Business-unit rollup
- App/team risk report
- Audit evidence pack
- Exception report
- mTLS posture report
- Authorization coverage report
- Gateway exposure report
- Upgrade readiness report

Goal:

> Give platform, security, risk, and audit teams evidence they can use.

### 12.8 Remediation Workflow

The first versions should not mutate production directly. They should produce:

- Remediation guidance
- Suggested YAML
- GitHub or GitLab pull requests
- Jira tickets
- ServiceNow items
- Slack or Teams alerts
- Kyverno or OPA policy recommendations
- Exception requests

Goal:

> Move from finding to action through approved enterprise workflows.

## 13. Initial Control Library

### Category A: mTLS Assurance

| Control ID | Control |
| --- | --- |
| OMG-MTLS-001 | Production namespaces must enforce strict mTLS. |
| OMG-MTLS-002 | Mesh-wide policy must not permit plaintext in production without exception. |
| OMG-MTLS-003 | Workloads must not disable mTLS without approved exception. |
| OMG-MTLS-004 | Namespaces transitioning to strict mTLS must have migration status and owner. |
| OMG-MTLS-005 | Ambient mesh namespaces must explicitly validate L4 mTLS posture. |
| OMG-MTLS-006 | Ambient workloads must have healthy ztunnel coverage on every scheduled node. |

### Category B: Authorization / Zero Trust

| Control ID | Control |
| --- | --- |
| OMG-AUTHZ-001 | Mesh-enabled production namespaces must define AuthorizationPolicy. |
| OMG-AUTHZ-002 | Production namespaces should use default-deny plus explicit allow policies. |
| OMG-AUTHZ-003 | AuthorizationPolicy must not allow broad access without exception. |
| OMG-AUTHZ-004 | AuthorizationPolicy must scope access to approved principals, namespaces, or workloads. |
| OMG-AUTHZ-005 | Policy coverage must be evaluated at workload level, not only namespace level. |
| OMG-AUTHZ-006 | Ambient L7 authorization requirements must use waypoint-enforced policy attachment. |
| OMG-AUTHZ-007 | Ambient policies that require L7 attributes must not be treated as equivalent to ztunnel-enforced L4 policy. |

### Category C: Exposure and Boundary Control

| Control ID | Control |
| --- | --- |
| OMG-GW-001 | Public Gateways must not use wildcard hosts. |
| OMG-GW-002 | Gateway routes must map to approved applications and owners. |
| OMG-GW-003 | VirtualServices must not route production traffic to non-production services. |
| OMG-GW-004 | Kubernetes Gateway API resources used by Istio must map to approved applications and owners. |
| OMG-GW-005 | Waypoint scope and enrollment must be explicit for services or namespaces that require L7 policy. |
| OMG-EGRESS-001 | ServiceEntries must not allow broad external egress without exception. |
| OMG-EGRESS-002 | External service access must map to owner and business justification. |

### Category D: Governance and Ownership

| Control ID | Control |
| --- | --- |
| OMG-OWN-001 | Mesh resources must map to an application owner. |
| OMG-OWN-002 | Production mesh resources must include app ID, environment, owner, and repo metadata. |
| OMG-EXC-001 | Exceptions must include approver, justification, ticket, and expiration. |
| OMG-EXC-002 | Expired exceptions must be escalated. |

### Category E: Lifecycle and Drift

| Control ID | Control |
| --- | --- |
| OMG-VER-001 | Sidecar proxy versions must match approved baseline. |
| OMG-VER-002 | Control plane versions must be supported. |
| OMG-VER-003 | ztunnel and waypoint versions must match approved baseline where ambient is enabled. |
| OMG-VER-004 | Istio control-plane/data-plane skew must follow upstream Istio support rules. |
| OMG-EF-001 | EnvoyFilter usage requires approval and exception. |
| OMG-UPG-001 | Upgrade blockers must be identified by app/team. |

## 14. Scoring Model Draft

Scores should be understandable and defensible. Start with a weighted score by namespace or application, then roll up to cluster and fleet views.

Initial scoring dimensions:

| Dimension | Weight |
| --- | ---: |
| mTLS enforced | 25 |
| AuthorizationPolicy coverage | 25 |
| Exposure controls | 15 |
| Egress controls | 10 |
| Owner metadata | 10 |
| Exception hygiene | 10 |
| Supported lifecycle baseline | 5 |

Rules:

- Critical findings should cap the affected resource score.
- Expired exceptions should count as active risk.
- Missing owner metadata should lower accountability, not only data quality.
- Scores must always link to the evidence that produced them.

## 15. Evidence Model

Every finding must include enough evidence for a platform or security engineer to understand what was observed and what was inferred.

Minimum finding fields:

- Finding ID
- Control ID
- Severity
- Affected cluster, namespace, workload, service, or mesh resource
- Data plane mode: sidecar, ambient, mixed, unknown, or not applicable
- Evidence source: Kubernetes API, Istio CRD, Gateway API, Prometheus, ownership file, exception file, or vendor API
- Evidence confidence: observed, inferred, user-supplied, or unavailable
- Resource references
- Reasoning summary
- Remediation guidance
- Exception status

Reports must separate:

- Confirmed risk
- Unsupported or out-of-support lifecycle posture
- Missing evidence
- Unknown posture because required inputs were not available
- Organization-policy failure, such as missing required owner or exception metadata

## 16. Build-Vs-Buy Argument

Enterprises can build scanners. OpenMeshGuard is not selling the idea that it can parse YAML and they cannot.

The value is the maintained governance foundation:

- Control library
- Normalized multi-cluster inventory
- Ownership mapping
- Exception lifecycle
- Deployed-state drift detection
- Audit evidence
- Historical posture
- Enterprise workflow integrations

AI makes it easier to build a first scanner. It does not remove the need for a trusted governance system of record.

## 17. Roadmap

### Phase 0: Foundation

- Private GitHub repo
- Product README
- Product spec
- Private landing page repo
- Initial positioning page
- OSS v1 support contract
- Least-privilege access model

### Phase 1: OSS Scanner Prototype

- Go scanner core and CLI
- Static report asset generation from canonical OpenMeshGuard JSON
- Typed-first Kubernetes, Istio, and Gateway API client strategy with dynamic discovery fallback
- Least-privilege RBAC manifests for namespace and cluster scans
- Kubernetes discovery
- Istio CRD discovery
- Gateway API discovery
- Normalized inventory model
- Sidecar and ambient data plane mode detection
- Single-cluster and multi-cluster topology model
- First control library
- CLI scan command
- JSON findings output

### Phase 2: Local Report

- HTML report
- Score summary
- Evidence records
- Remediation guidance
- SARIF export
- CI mode
- Permission and evidence summary in every report
- Kind-based validation fixtures for upstream Istio sidecar, upstream Istio ambient, mixed mode, and multi-cluster scenarios
- Required multicluster fixture: two Kind clusters using upstream Istio multi-primary multi-network with east-west gateways and remote secrets

### Phase 3: Ownership and Deployed-State Governance

- Ownership mapping file
- Exception file format
- Control mapping export
- YAML control configuration for severity, environment baseline, and allowed patterns

### Phase 4: Source Traceability and Drift

- Source metadata detection across GitOps, Helm, CI/CD, Kubernetes managedFields, and explicit mapping files
- Optional repository or manifest comparison
- Drift model that distinguishes missing source metadata, unverified source, and confirmed live-vs-source mismatch

### Phase 5: Distribution Validation

- OpenShift Service Mesh compatibility validation
- OpenShift Service Mesh ambient validation
- Tetrate compatibility validation
- Solo / Gloo Mesh / Solo Enterprise for Istio compatibility validation
- Vendor API gap analysis
- Provider-specific caveat reporting where normalized Istio resources are insufficient

### Phase 6: Enterprise Design Partner Surface

- Multi-cluster ingestion design
- Auth/RBAC design
- Exception lifecycle workflow
- Ticket/PR integration design
- Evidence pack export

## 18. Landing Page Direction

The landing page should be compact like Agent Executor: early-preview label, direct value statement, GitHub link, and product pillars. It should borrow tone and structure from Istio's site/docs: service mesh, security, traffic, observability, reliability, ambient readiness, and Kubernetes-native vocabulary.

First hero:

> OpenMeshGuard  
> Move from assumed mesh security to verified Istio posture.

Subtext:

> Istio gives enterprises powerful mTLS, identity, and policy controls, but misconfiguration can leave services permissive, overexposed, or unauditable. OpenMeshGuard continuously validates Istio security posture, ownership, exceptions, drift, and compliance evidence across every cluster.

## 19. Open Questions

- Which subset of distribution validation can realistically run in disposable or lab environments without vendor licensing friction?
