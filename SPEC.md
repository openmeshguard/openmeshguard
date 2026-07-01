# OpenMeshGuard Product Spec

Status: foundation draft  
Date: 2026-06-30  
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

Long-term expansion can include Envoy-based service connectivity, Gateway API posture, Cilium service mesh posture, Linkerd, Kuma, and broader service-to-service security governance. The wedge stays focused on posture, evidence, ownership, exception, drift, and lifecycle management.

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

1. Read-only first.
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
- Optional Prometheus
- Optional Git repo metadata
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

## 8. Architecture Map

Community architecture:

```text
Kubernetes / Istio APIs
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

## 9. Core Modules

### 9.1 Mesh Inventory

Discovers:

- Clusters
- Namespaces
- Workloads
- Services
- Gateways
- VirtualServices
- DestinationRules
- PeerAuthentications
- AuthorizationPolicies
- ServiceEntries
- EnvoyFilters
- Sidecar proxy versions
- Istio control-plane versions
- Ambient mesh labels and configuration
- Owner, app, environment, and repo metadata

Goal:

> Know what exists, where it exists, and who owns it.

### 9.2 Security Posture Assessment

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
- Ambient mesh-specific risks

Goal:

> Validate that Istio security is real, not assumed.

### 9.3 Governance Posture

Evaluates:

- App owner mapping
- Business unit mapping
- Environment classification
- GitOps source-of-truth metadata
- Onboarding status
- Approved pattern usage
- Exception presence
- Exception expiration
- Change ticket linkage
- Ownership gaps

Goal:

> Make mesh risk accountable.

### 9.4 Compliance and Control Mapping

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

### 9.5 Exception Management

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

### 9.6 Drift Detection

Detects:

- Istio resources not traceable to Git
- In-cluster config differing from Git source
- Manual changes
- Missing GitOps metadata
- Resources deployed outside approved pipelines
- Policy changes without change records

Goal:

> Prove mesh configuration follows change-control expectations.

### 9.7 Evidence and Reporting

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

### 9.8 Remediation Workflow

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

## 10. Initial Control Library

### Category A: mTLS Assurance

| Control ID | Control |
| --- | --- |
| OMG-MTLS-001 | Production namespaces must enforce strict mTLS. |
| OMG-MTLS-002 | Mesh-wide policy must not permit plaintext in production without exception. |
| OMG-MTLS-003 | Workloads must not disable mTLS without approved exception. |
| OMG-MTLS-004 | Namespaces transitioning to strict mTLS must have migration status and owner. |
| OMG-MTLS-005 | Ambient mesh namespaces must explicitly validate L4 mTLS posture. |

### Category B: Authorization / Zero Trust

| Control ID | Control |
| --- | --- |
| OMG-AUTHZ-001 | Mesh-enabled production namespaces must define AuthorizationPolicy. |
| OMG-AUTHZ-002 | Production namespaces should use default-deny plus explicit allow policies. |
| OMG-AUTHZ-003 | AuthorizationPolicy must not allow broad access without exception. |
| OMG-AUTHZ-004 | AuthorizationPolicy must scope access to approved principals, namespaces, or workloads. |
| OMG-AUTHZ-005 | Policy coverage must be evaluated at workload level, not only namespace level. |

### Category C: Exposure and Boundary Control

| Control ID | Control |
| --- | --- |
| OMG-GW-001 | Public Gateways must not use wildcard hosts. |
| OMG-GW-002 | Gateway routes must map to approved applications and owners. |
| OMG-GW-003 | VirtualServices must not route production traffic to non-production services. |
| OMG-EGRESS-001 | ServiceEntries must not allow broad external egress without exception. |
| OMG-EGRESS-002 | External service access must map to owner and business justification. |

### Category D: Governance and Ownership

| Control ID | Control |
| --- | --- |
| OMG-OWN-001 | Mesh resources must map to an application owner. |
| OMG-OWN-002 | Production mesh resources must include app ID, environment, owner, and repo metadata. |
| OMG-EXC-001 | Exceptions must include approver, justification, ticket, and expiration. |
| OMG-EXC-002 | Expired exceptions must be escalated. |
| OMG-GIT-001 | Mesh configuration must be traceable to approved GitOps source. |

### Category E: Lifecycle and Drift

| Control ID | Control |
| --- | --- |
| OMG-VER-001 | Sidecar proxy versions must match approved baseline. |
| OMG-VER-002 | Control plane versions must be supported. |
| OMG-DRIFT-001 | In-cluster Istio resources must match approved GitOps source. |
| OMG-EF-001 | EnvoyFilter usage requires approval and exception. |
| OMG-UPG-001 | Upgrade blockers must be identified by app/team. |

## 11. Scoring Model Draft

Scores should be understandable and defensible. Start with a weighted score by namespace or application, then roll up to cluster and fleet views.

Initial scoring dimensions:

| Dimension | Weight |
| --- | ---: |
| mTLS enforced | 20 |
| AuthorizationPolicy coverage | 20 |
| Exposure controls | 15 |
| Egress controls | 10 |
| GitOps traceability | 10 |
| Owner metadata | 10 |
| Exception hygiene | 10 |
| Supported lifecycle baseline | 5 |

Rules:

- Critical findings should cap the affected resource score.
- Expired exceptions should count as active risk.
- Missing owner metadata should lower accountability, not only data quality.
- Scores must always link to the evidence that produced them.

## 12. Build-Vs-Buy Argument

Enterprises can build scanners. OpenMeshGuard is not selling the idea that it can parse YAML and they cannot.

The value is the maintained governance foundation:

- Control library
- Normalized multi-cluster inventory
- Ownership mapping
- Exception lifecycle
- Drift detection
- Audit evidence
- Historical posture
- Enterprise workflow integrations

AI makes it easier to build a first scanner. It does not remove the need for a trusted governance system of record.

## 13. Roadmap

### Phase 0: Foundation

- Private GitHub repo
- Product README
- Product spec
- Private landing page repo
- Initial positioning page

### Phase 1: OSS Scanner Prototype

- Language and package layout decision
- Kubernetes discovery
- Istio CRD discovery
- Normalized inventory model
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

### Phase 3: Ownership and Drift

- Ownership mapping file
- GitOps metadata detection
- Basic drift model
- Exception file format
- Control mapping export

### Phase 4: Enterprise Design Partner Surface

- Multi-cluster ingestion design
- Auth/RBAC design
- Exception lifecycle workflow
- Ticket/PR integration design
- Evidence pack export

## 14. Landing Page Direction

The landing page should be compact like Agent Executor: early-preview label, direct value statement, GitHub link, and product pillars. It should borrow tone and structure from Istio's site/docs: service mesh, security, traffic, observability, reliability, ambient readiness, and Kubernetes-native vocabulary.

First hero:

> OpenMeshGuard  
> Move from assumed mesh security to verified Istio posture.

Subtext:

> Istio gives enterprises powerful mTLS, identity, and policy controls, but misconfiguration can leave services permissive, overexposed, or unauditable. OpenMeshGuard continuously validates Istio security posture, ownership, exceptions, drift, and compliance evidence across every cluster.

## 15. Open Questions

- Should the initial implementation be Go or TypeScript?
- Should the scanner use Kubernetes dynamic clients first or typed Istio clients?
- Should the HTML report be generated statically by the CLI or served by a local UI?
- What is the first supported Istio version range?
- How much framework mapping belongs in OSS before it becomes compliance theater?
- What is the right custom rule format: YAML, Rego, CEL, or purpose-built JSON/YAML?
- Should the first GitOps drift check compare live resources to annotations only, or inspect repositories directly?
- What design partner environment can provide realistic sanitized Istio resources?
