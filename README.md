# OpenMeshGuard

Move from assumed mesh security to verified Istio posture.

OpenMeshGuard is an open-core service mesh governance platform, starting with Istio. It helps platform, security, and risk teams continuously verify mTLS, authorization, exposure, ownership, exceptions, drift, lifecycle, and compliance evidence across clusters.

The first product surface is a read-only posture scanner and report generator. It is not another Kiali, not a mesh distribution, and not a generic observability dashboard. OpenMeshGuard sits above the mesh as a governance and evidence layer.

## Why This Exists

Istio gives enterprises powerful security primitives: workload identity, mTLS, authorization policy, ingress and egress controls, traffic policy, and telemetry. Those controls only create real zero-trust outcomes when they are configured correctly, consistently adopted, continuously validated, and mapped to enterprise ownership and controls.

Large organizations need to answer questions like:

- Are production namespaces enforcing strict mTLS?
- Which mesh-enabled apps have no AuthorizationPolicy?
- Which Gateways or VirtualServices create exposure risk?
- Which Istio resources are missing owner, app, environment, or repo metadata?
- Which policies changed in-cluster without a GitOps source of truth?
- Which teams have active, expired, or missing exceptions?
- Which workloads block an Istio or ambient mesh migration?
- Can platform and security teams export audit-ready evidence without spreadsheets?

## Product Thesis

OpenMeshGuard helps enterprises govern service mesh adoption and risk posture without taking over mesh operations.

The initial category is:

> Service Mesh Governance and Risk Posture Management

The initial beachhead is:

> Istio Governance Posture Management

## First Scope

OpenMeshGuard Community should let a platform engineer run a read-only scan against an Istio cluster and immediately receive a useful governance posture report.

Planned CLI shape:

```bash
openmeshguard scan --context prod-cluster --all-namespaces
openmeshguard report --format html
openmeshguard export --format sarif
openmeshguard score --namespace payments-prod
```

Initial outputs:

- Mesh inventory across clusters, namespaces, workloads, services, Gateways, VirtualServices, DestinationRules, PeerAuthentications, AuthorizationPolicies, ServiceEntries, EnvoyFilters, proxy versions, and ambient labels.
- Findings with severity, affected resources, evidence, and remediation guidance.
- Posture scores by cluster, namespace, workload, application, and control area.
- JSON, SARIF, and local HTML reports.
- Framework-aligned evidence mapping where defensible.

## What OpenMeshGuard Is Not

OpenMeshGuard is not trying to replace:

- Istio
- Kiali
- Solo or Tetrate platforms
- Kyverno, OPA, or Gatekeeper
- Grafana, Prometheus, Datadog, or OpenTelemetry
- CNAPP or KSPM tools

The first principle is read-only intelligence. Remediation should flow through GitOps, pull requests, tickets, policy engines, or existing enterprise change workflows.

## Initial Control Areas

OpenMeshGuard starts with a small, high-signal control library:

- mTLS assurance
- Authorization and zero-trust coverage
- Gateway and egress exposure
- Ownership and metadata completeness
- Exception lifecycle
- GitOps traceability and drift
- Proxy and control-plane lifecycle
- EnvoyFilter and advanced customization review
- Ambient migration readiness

## Open Core Split

Community edition focuses on practitioner trust and local value:

- CLI scanner
- Istio and Kubernetes discovery
- Built-in OpenMeshGuard baseline controls
- Basic posture score
- Local HTML report
- JSON and SARIF export
- Custom rule files
- CI mode

Enterprise edition should monetize scale and workflow:

- Multi-cluster fleet inventory
- SSO and enterprise RBAC
- Historical posture trends
- Exception lifecycle management
- Audit evidence packs
- ServiceNow and Jira workflows
- Backstage, CMDB, and GitOps ownership mapping
- Remediation pull requests
- Premium control packs
- Support and SLA

## Repository Status

This repository is the private product foundation for OpenMeshGuard. The current work is intentionally documentation-first: README, product spec, and landing-page positioning before implementation starts.

See [SPEC.md](./SPEC.md) for the first product specification.
