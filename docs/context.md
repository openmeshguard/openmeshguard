# Governance context

Governance context is optional. Without context files, OpenMeshGuard still
scans the mesh and reports unclassified namespaces and unowned workloads
instead of guessing or treating missing context as compliant.

```bash
openmeshguard scan \
  --context my-cluster \
  --all-namespaces \
  --scan-config ./openmeshguard.yaml
```

The scan config, ownership import, and exception record formats use strict
YAML field validation. Relative `inputs` paths are resolved from the directory
containing the scan config.

## Scan config

```yaml
apiVersion: openmeshguard.io/v1alpha1
kind: ScanConfig
metadata:
  name: platform-governance
  version: 1.0.0

inputs:
  ownershipImport: ownership.yaml
  exceptions:
    - exceptions/

classification:
  sources:
    - type: namespace-mapping
      mappings:
        legacy-payments: production
    - type: namespace-label
      keys:
        - catalog.example.com/environment
        - environment
    - type: namespace-name
      rules:
        - pattern: '-prod$'
          environment: production
        - pattern: '-stage$'
          environment: staging
    - type: cluster-context
      mappings:
        us-east-production: production

ownership:
  labels:
    appId:
      - catalog.example.com/application-id
      - app-id
    owner:
      - catalog.example.com/team
  applications:
    - appId: checkout
      owner: team-checkout

controls:
  parameters:
    defaults:
      approvedIstioVersions: ["1.30"]
    environments:
      production:
        approvedIstioVersions: ["1.30"]
  overrides:
    - controlId: MG-MTLS-001
      environments: [production]
      severityByEnvironment:
        production: critical
```

Configured `classification.sources` are evaluated in order; the first
conclusive source wins. Supported source types are:

- `namespace-mapping`: exact namespace-to-environment mappings.
- `namespace-label`: the first non-empty value from the ordered namespace
  label keys.
- `namespace-name`: the first matching regular-expression rule.
- `cluster`: one `environment` value for every namespace in the cluster.
- `cluster-context`: exact kubeconfig-context-to-environment mappings.

Environment names in mappings, rules, cluster sources, parameter maps, and
control overrides must be YAML strings without surrounding whitespace.
Boolean or numeric scalar coercion is rejected instead of creating an
environment that silently misses an exact control scope.

When `classification.sources` is omitted, the default is one namespace-label
source with the ordered keys `openmeshguard.io/environment`, `environment`,
and `env`. Supplying sources replaces that default. If evidence required by a
higher-precedence source is unavailable, classification is unknown and does
not fall through to a lower source.

If every configured source is available but unmatched, the namespace is
`unclassified`. Passing `--infer-environments` adds the built-in `-prod` and
`-production` namespace suffix rules after configured sources. It is disabled
by default; inferred results use `environmentConfidence: inferred`, and the
report sets `scan.environmentInference: true`.

Control parameters merge in this order: control-pack parameters, scan-config
`defaults`, then parameters for the target environment. An override
`environments` list replaces the control's list. A
`severityByEnvironment` entry changes severity only for that environment.
Unknown control IDs, duplicate overrides, and invalid severities are errors;
overrides cannot disable a control. The mandatory coverage controls
MG-ENV-001, MG-OWN-001, MG-EXC-001, and MG-EXC-002 reject environment-list
overrides because filtering any of them would remove the governance finding
that protects an otherwise skipped target.

## Ownership imports

The configured `appId` and `owner` key lists are applied to workload labels,
workload annotations, namespace labels, and namespace annotations, in that
order. Supplying either key list replaces its defaults.

Default application-ID keys are `openmeshguard.io/app-id`,
`app.kubernetes.io/part-of`, and `app.kubernetes.io/name`. The default owner
key is `openmeshguard.io/owner`.

After metadata, OpenMeshGuard resolves an owner from
`ownership.applications`, then from the ownership import. The import maps a
canonical application ID to its owner, so the same application can be
deployed in any number of namespaces without repeating namespace mappings.

```yaml
apiVersion: openmeshguard.io/v1alpha1
kind: OwnershipImport
metadata:
  name: application-catalog
  version: 2026-07-24
applications:
  - appId: payments-api
    owner: team-payments
  - appId: checkout
    owner: team-checkout
```

Pass the file through `inputs.ownershipImport` or
`--ownership-import ./ownership.yaml`, but not both. A CSV bridge is also
accepted with the exact header:

```csv
appId,owner
payments-api,team-payments
checkout,team-checkout
```

If Kubernetes metadata needed by a higher-precedence source is unavailable,
ownership remains unknown; config and imports do not silently override that
missing evidence.

For controller-backed workloads, resource metadata supplements pod-template
metadata and wins on a duplicate key. When policy variation requires the
normalizer to emit per-Pod workload results, metadata from the owning
ReplicaSet and controller is retained and the highest owning controller wins
on collisions. Pod and template metadata fill missing keys. This keeps
ownership and exception references stable during rollouts instead of changing
governance identity when a controller is temporarily projected as per-Pod
results.

## Exception records

OpenMeshGuard consumes exception decisions made in the organization's existing
review system. It does not implement approval workflow or scope selectors.
The workload opts into one record by exact ID:

```yaml
metadata:
  annotations:
    openmeshguard.io/exception: EXC-1042
```

The referenced record is:

```yaml
apiVersion: openmeshguard.io/v1alpha1
kind: Exception
metadata:
  name: EXC-1042
spec:
  controlIds:
    - MG-MTLS-001
  owner: team-payments
  approver: security@example.com
  justification: Approved migration window
  ticket: https://tickets.example.com/SEC-1042
  expiresAt: 2026-10-01T00:00:00Z
```

Pass files or directories with repeatable `--exceptions` flags, or list them
under `inputs.exceptions`, but not both. Directories load `.yaml` and `.yml`
regular files in lexical order. Duplicate IDs and structurally invalid YAML
are scan errors. MG-EXC-001 reports missing or invalid record fields and
dangling annotations.

Exceptions are applied after control evaluation. Only an underlying `open`
finding can become `excepted`; unknown and not-applicable results are
unchanged. The finding is never removed, and its severity, reasoning,
resources, and resolution chain remain intact. An expired record leaves the
finding `open` at its original severity, attaches the expired exception
evidence, and raises MG-EXC-002. MG-EXC-001 and MG-EXC-002 cannot themselves
be excepted.

The record `spec.owner` must exactly match the workload's resolved owner before
the annotation can except a finding. A mismatch raises MG-EXC-001 and leaves
the original finding open. If owner evidence is unavailable, exception
matching remains unknown and the original finding also stays open. This
prevents a valid exception ID approved for one team from being replayed by
another workload without adding a new scope-selector format.

Removing a record revokes it for the next scan; Git history remains the audit
trail. Cluster, namespace, selector, and implicit resource scopes, explicit
revocation/status fields, multiple-exception resolution, and ticket-system
integrations are intentionally deferred.
