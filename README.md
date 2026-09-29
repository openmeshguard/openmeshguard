# OpenMeshGuard

**Move from assumed mesh security to verified Istio posture.**

> ⚠️ **Early preview.** OpenMeshGuard is under active development and not yet ready for production use. The scanner, schemas, and control library are evolving. Feedback and issues are very welcome.

Adopting Istio is not the same as being protected by it. mTLS can be permissive where you believe it's strict, AuthorizationPolicies can silently enforce nothing, ambient L7 policies can sit unenforced without a waypoint, and none of it shows up until an audit — or an incident.

OpenMeshGuard is a read-only CLI scanner that tells you what your mesh security posture *actually is*:

- **Resolved, not linted.** It computes per-workload *effective* posture using Istio's real evaluation semantics — layered PeerAuthentication (mesh/namespace/workload/port), DestinationRule TLS interplay, AuthorizationPolicy evaluation order (CUSTOM → DENY → ALLOW), Sidecar scoping, and the ambient L4/L7 split — instead of checking resources one YAML at a time.
- **Verification is never implied.** The report keeps declared and verified posture separate. Optional Prometheus
  evidence now identifies observed plaintext and mutual TLS event share;
  missing telemetry remains explicitly unknown.
- **Honest about what it doesn't know.** Missing permissions, missing telemetry, and unclassified namespaces are reported as explicit unknowns — never silently passed or failed.
- **Evidence you can hand to a security team.** Every finding carries its resolution chain: which resources, in which order, produced the conclusion.

## Install

Community release `v0.2.0` adds optional Prometheus runtime verification alongside
declared and resolved posture. Download the archive for
your platform from [GitHub Releases](https://github.com/openmeshguard/openmeshguard/releases/tag/v0.2.0):

| Platform | Archive |
| --- | --- |
| macOS Apple silicon | `openmeshguard_0.2.0_darwin_arm64.tar.gz` |
| macOS Intel | `openmeshguard_0.2.0_darwin_amd64.tar.gz` |
| Linux arm64 | `openmeshguard_0.2.0_linux_arm64.tar.gz` |
| Linux amd64 | `openmeshguard_0.2.0_linux_amd64.tar.gz` |

Before extraction, [verify the archive and signed checksums](docs/releases/verification.md)
using the accompanying Sigstore bundles. Extract the archive, put `openmeshguard`
on your `PATH`, then run `openmeshguard version`; it reports `version=v0.2.0`.

With a Go 1.24+ toolchain:

```bash
go install github.com/openmeshguard/openmeshguard/cmd/openmeshguard@v0.2.0
openmeshguard version
```

For development, clone the repository and run `make build`. The binary lands in
`bin/openmeshguard` and reports `version=dev`. A release build uses
`VERSION=v0.2.0 make build`. No Istio installation is performed by the scanner.

## Try it today

The scanner resolves **effective mTLS and authorization posture** end to end
for sidecar, ambient, and mixed meshes, then evaluates the built-in security
and governance control packs. Point it at any cluster where your kubeconfig
has read access (or apply the least-privilege profiles in
[`deploy/rbac/`](deploy/rbac/)):

```bash
# Scan one namespace (repeatable flag), or --all-namespaces
openmeshguard scan --context my-cluster --namespace payments > report.json

# Optional runtime evidence from a cluster-scoped Prometheus endpoint
openmeshguard scan --namespace payments \
  --prometheus-url https://prometheus.example.com \
  --prometheus-token-file /path/to/token > verified-report.json

# Effective mTLS per workload, with the policies that produced each conclusion
jq '.workloadPostures[] | {workload, dataPlaneMode, mtls: .mtls.effective}' report.json

# Findings from the built-in control pack, each with its resolution chain
jq '.findings[] | {controlId, status, severity, reasoning}' report.json

# Numeric score, category grades, and namespace rollups
jq '.scores' report.json

# Self-contained human report and SARIF compatibility export
openmeshguard report --input report.json --format html --output report.html
openmeshguard export --input report.json --format sarif --output openmeshguard.sarif

# CI threshold: writes canonical JSON, exits 1 for open high/critical findings
openmeshguard scan \
  --context my-cluster \
  --namespace payments \
  --fail-on high > ci-report.json
```

What to expect in the output:

- **`workloadPostures`** — per-workload effective mTLS (`strict` / `permissive` / `disabled` / `mixed-by-port` / `unknown`) resolved through Istio's real precedence (mesh → namespace → workload → port), each with the ordered `chain` of resources that produced it.
- **`findings`** — engine-evaluated controls (e.g. `MG-MTLS-001` when a workload resolves to permissive), with severity, evidence sources, and remediation. Evidence the scanner cannot obtain yields findings with status `unknown` and an explicit `unknownReason` — never a silent pass.
- **`permissionSummary`** — exactly which access the scanner had, and what degraded without it.
- **`scores`** — numeric weighted cluster/namespace scores plus per-category pass rates and letter grades.

The report is canonical JSON validating against [`docs/contracts/canonical-json-schema.json`](docs/contracts/canonical-json-schema.json). HTML, SARIF, score display, and CI thresholds are projections of that JSON. See [consumable outputs and the CI contract](docs/outputs.md).

## Quickstart

```bash
# Scan a cluster (read-only; see deploy/rbac for the exact permissions)
openmeshguard scan --context my-cluster --all-namespaces > openmeshguard.json

# Generate a local, server-less HTML report
openmeshguard report --input openmeshguard.json --format html --output report.html

# Export for CI / code scanning
openmeshguard export --input openmeshguard.json --format sarif --output openmeshguard.sarif

# Print the canonical weighted score and category grades
openmeshguard score --input openmeshguard.json
```

First run requires **zero configuration files**. Ownership, environment
classification, and exception records are optional inputs that unlock
governance controls — see [governance context](docs/context.md).

### Example summary

| Control area | Declared | Verified | Unknown |
| --- | --- | --- | --- |
| Strict mTLS (effective, per workload) | 71% | Observed mTLS event share | Unknown without telemetry |
| Explicit authorization coverage | 54% | — | — |
| Default-deny posture | 22% of namespaces | — | — |
| Public gateway wildcard hosts | 3 findings | — | — |
| Environment classification coverage | 61% | — | 39% unclassified |

## What it needs

- Read-only access: `get`/`list` on core workload resources, Istio CRDs, and Gateway API resources. Published RBAC profiles (namespace-scoped Role, cluster-scoped ClusterRole, optional add-ons) ship with the project.
- **Never required:** write verbs, Secrets access, `exec`/`attach`/`port-forward`, impersonation, `watch`, or cluster-admin.
- Optional Prometheus endpoint with standard Istio proxy counters: `--prometheus-url`, bearer-token file or mTLS client authentication. Runtime verification currently validates sidecar destination metrics; ambient/mixed verification remains unknown. See [runtime verification](docs/telemetry/prometheus.md).

Every report includes a permission summary showing which evidence was available and which findings were affected by missing access.

## What it is — and isn't

OpenMeshGuard complements the tools you already run. It does not replace them.

| You already use | Keep using it for | OpenMeshGuard adds |
| --- | --- | --- |
| `istioctl analyze` | Config validity checks | Effective per-workload posture, governance context, scoring, and evidence |
| Kiali | Live mesh visualization and ops | Control-oriented posture, audit evidence, exception awareness |
| Kyverno / OPA Gatekeeper | Blocking violations at admission | Resolution of Istio's layered policy semantics that per-resource rules can't see; posture over time |
| Prometheus / Grafana | Metrics and dashboards | Observed plaintext findings, mTLS event share, bounded queries, and separate runtime evidence |

Non-goals: installing or managing Istio, traffic management, replacing mesh vendors or policy engines, claiming NIST/PCI/HIPAA compliance (it produces framework-*aligned* evidence only), or mutating your clusters.

## Controls

Controls are **data, not code**: YAML metadata plus a CEL expression evaluated against the normalized mesh model and resolver outputs. The current built-in library covers effective mTLS posture, authorization/zero-trust coverage, gateway exposure, ownership, and exception hygiene across both **sidecar and ambient** data planes, including ztunnel coverage and waypoint enforcement gaps. The published lifecycle score dimension remains explicitly `unknown` until lifecycle controls ship.

You can ship your own control packs alongside the built-ins, and contributing a control upstream doesn't require writing Go.

## Sidecar, ambient, multi-cluster

- Sidecar and ambient modes (including mixed) are first-class.
- Multi-cluster: v1 scans one cluster at a time and *detects* multi-cluster participation (east-west gateways, network topology labels), reporting honestly that cross-cluster posture is not yet evaluated. Full multi-cluster correlation is on the roadmap.

## What works and what is next

| Capability | v0.2.0 |
| --- | --- |
| Effective sidecar/ambient mTLS and authorization, mixed mesh detection | Shipped |
| CEL controls, ownership, environment classification, Git-native exceptions | Shipped |
| Canonical JSON, HTML, SARIF, scores, opt-in CI thresholds | Shipped |
| Prometheus collection and runtime-verified posture | Shipped for sidecar destination metrics; optional |
| Lifecycle controls | Future; lifecycle score remains unknown |
| Offline manifests and cross-cluster correlation | Future |

## Roadmap (abridged)

1. Scanner core, effective posture resolver, CEL rule engine, canonical JSON
2. Governance context, HTML, SARIF, score, and CI mode
3. Prometheus collection and runtime-verified controls
4. Offline manifest scanning (`scan --local`) and source/drift traceability
5. Multi-cluster correlation and distribution validation (OpenShift Service Mesh on ROSA first, then managed K8s with upstream Istio)

See [SPEC.md](SPEC.md) for the full design. Steps 1–3 are implemented; later steps remain planned.

## Contributing

The project is in its early Community releases. Useful contributions include:

- Try it against a real Istio environment and file honest issues — especially resolver disagreements ("OpenMeshGuard says X, my mesh does Y"). Those are gold.
- Propose or contribute controls (YAML + CEL — no Go required).
- Propose improvements to the frozen contracts for maintainer review; contract changes need explicit approval.

Run `make build test lint schema-test` before submitting a change. See [development and acceptance testing](docs/dev.md) and [AGENTS.md](AGENTS.md) for the architecture and contribution constraints. Report vulnerabilities through [SECURITY.md](SECURITY.md).

## License

Apache License 2.0. See [LICENSE](LICENSE) for the canonical text.
