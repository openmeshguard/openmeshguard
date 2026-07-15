# Control Pack Format — v1alpha1 (frozen contract)

Controls are data, not Go code. A control pack is a YAML file containing pack metadata and a list of controls. Built-in packs live in `controls/` and are embedded into the binary; users supply additional packs with `--control-pack <path>` (repeatable). Changes to this format require human approval.

## Pack structure

```yaml
apiVersion: openmeshguard.io/v1alpha1
kind: ControlPack
metadata:
  name: builtin-mtls
  version: 0.1.0
controls:
  - id: MG-MTLS-001
    title: Mesh-managed workloads must resolve to strict mTLS
    category: mtls                # mtls | authz | exposure | governance | lifecycle
    severity: high                # critical | high | medium | low | info
    evidenceType: config          # config | runtime | context
    scope: workload               # workload | namespace | resource
    environments: [production]    # empty = all environments; classified envs only
    requires:                     # posture/context fields this control needs;
      - mtls.effective            # if any are unavailable, the engine emits an
                                  # `unknown` finding instead of evaluating CEL
    applicability: 'workload.dataPlaneMode != "not-applicable"'
    expression: 'workload.mtls.effective == "strict"'
    message: >-
      Effective mTLS for {{ .Workload }} resolves to
      {{ .Posture.Mtls.Effective }}, not strict.
    remediation:
      guidance: >-
        Apply or correct PeerAuthentication so the workload resolves to STRICT,
        then inspect the attached resolution chain.
      suggestedYAMLTemplate: peerauthentication-strict.tmpl
    frameworks:                   # tags only, never compliance claims
      - nist-csf-2.0/PR.DS-02
      - owasp-k8s-2025/K05
```

## Semantics (binding on the engine)

1. **Scope** determines the iteration unit. `workload` controls evaluate once per entry in `workloadPostures`; `namespace` controls once per mesh namespace; `resource` controls once per matching source API group and resource kind (declared via `match.apiGroups` and `match.kinds`).
2. **Environments** filter by resolved classification. A control scoped to `production` never evaluates unclassified namespaces — those are covered separately by MG-ENV-001. Empty list = evaluate everywhere.
3. **`applicability`** is a CEL expression. False ⇒ finding status `not-applicable` (not a pass, not counted in pass rates).
4. **`requires`** lists exact evidence paths into the evaluation input. Fixed fields use dotted segments; literal map keys that are not identifiers use bracket notation, such as `namespace.labels["app.kubernetes.io/name"]`. If any required path resolves to an unknown/unavailable value, the engine emits status `unknown` with `unknownReason` set, and never evaluates `expression`. This is how "unknown is never pass and never fail" is enforced mechanically — controls cannot forget it.
5. **`expression`** is a CEL expression returning bool. `true` = pass (no finding). `false` = finding with the control's severity. Any CEL evaluation error is an engine error surfaced at pack-load or scan time — never silently converted to pass or fail.
6. **Exceptions are engine concerns, not control concerns.** The engine matches findings to exception records after evaluation; controls never reference exceptions.
7. Severity and environment baselines may be overridden per environment in the scan config file; overrides are recorded in the report's `scanner.controlPacks` provenance.

## CEL environment

Available variables by scope:

| Variable | Scope | Contents |
| --- | --- | --- |
| `workload` | workload | A `WorkloadPosture` object per the canonical schema (mtls, authorization, verified, dataPlaneMode, environment, owner). |
| `namespace` | namespace, workload | Namespace name, labels, resolved environment, mesh-enrollment state. |
| `resource` | resource | The normalized resource under evaluation (typed views for Istio/Gateway API kinds). |
| `inventory` | all | Cluster-level summary: data plane mode, ztunnel coverage, control plane version, multiCluster signals. |
| `params` | all | Control-pack or scan-config supplied parameters (e.g. approved version baselines, allowed egress hosts). |

Standard CEL macros plus the `strings` extension are enabled. No custom functions in v1alpha1 without a contract update.

`requires` paths use the same field and literal-key boundaries as CEL. These
two forms therefore refer to the same exact evidence:

```yaml
requires: ['namespace.labels["app.kubernetes.io/name"]']
expression: 'namespace.labels["app.kubernetes.io/name"] == "payments"'
```

Literal bracket keys are one map segment even when they contain dots or
slashes. Dynamic map indexes cannot be declared exactly and are rejected;
iteration over a specifically required collection remains valid.

## Resource matching and views

Resource controls are explicit about the native API family they evaluate. They
must declare both `match.apiGroups` and `match.kinds`:

```yaml
match:
  apiGroups: [gateway.networking.k8s.io]
  kinds: [Gateway]
```

Values within each list are ORed; the two dimensions are ANDed. The example
above matches `Gateway` resources in `gateway.networking.k8s.io`, regardless of
the served version. The core Kubernetes API group is written as an empty string
(`apiGroups: [""]`). A resource that does not match is not a control target: it
produces no finding and does not affect scoring. `match` is evaluated before
`applicability` and never produces `not-applicable`.

The `resource` CEL variable preserves the matched source API rather than
translating between API families. `resource.apiVersion`, `resource.kind`,
`resource.namespace`, and `resource.name` identify the native object;
source-native fields retain their API structure under `resource.spec`. Engine-
derived evidence may be exposed alongside `spec` when its semantics are common,
such as `resource.isPubliclyExposed`. Missing source or derived evidence follows
the normal `requires` rule and produces `unknown`; it is never synthesized by
cross-API translation.

Equivalent objectives implemented by APIs with different schemas use separate
control IDs. For example, an Istio `Gateway` control reads
`resource.spec.servers`, while a Kubernetes Gateway API control reads
`resource.spec.listeners`. This keeps evaluation and remediation source-native
and makes any parity between the controls explicit and testable.

## Validation

`openmeshguard controls validate <path>` (and pack loading at scan time) must reject:

- Unknown fields, missing required fields, malformed IDs (`^MG-[A-Z]+-[0-9]{3}$` for built-ins; user packs may use their own prefix but must match `^[A-Z]+-[A-Z]+-[0-9]{3}$`).
- CEL expressions that fail compilation or reference variables outside the declared scope.
- Invalid `requires` syntax, including non-literal bracket keys; every expression dependency must have an exact matching evidence path.
- `runtime` evidenceType controls that do not `require` a `verified.*` field.
- Resource controls without non-empty `match.apiGroups` and `match.kinds` lists (the empty-string value explicitly selects the core API group); `match` is invalid for workload and namespace controls.
- Duplicate control IDs across all loaded packs.

Error messages must point at the pack file, control ID, and CEL compile position.

## Worked examples

```yaml
  - id: MG-GW-001
    title: Public Istio gateways must not use wildcard hosts
    category: exposure
    severity: high
    evidenceType: config
    scope: resource
    match:
      apiGroups: [networking.istio.io]
      kinds: [Gateway]
    requires: [resource.spec.servers]
    applicability: 'resource.isPubliclyExposed'
    expression: '!resource.spec.servers.exists(s, s.hosts.exists(h, h == "*" || h.startsWith("*.")))'
    message: 'Istio Gateway {{ .Resource }} exposes wildcard host(s) publicly.'

  - id: MG-GW-002
    title: Public Kubernetes gateways must not use wildcard hosts
    category: exposure
    severity: high
    evidenceType: config
    scope: resource
    match:
      apiGroups: [gateway.networking.k8s.io]
      kinds: [Gateway]
    requires: [resource.spec.listeners]
    applicability: 'resource.isPubliclyExposed'
    expression: >-
      !resource.spec.listeners.exists(l,
        has(l.hostname) &&
        (l.hostname == "*" || l.hostname.startsWith("*.")))
    message: 'Kubernetes Gateway {{ .Resource }} exposes a wildcard hostname publicly.'

  - id: MG-MTLS-101
    title: No plaintext traffic observed to mesh workloads
    category: mtls
    severity: critical
    evidenceType: runtime
    scope: workload
    requires: [verified.plaintextObserved]
    applicability: 'workload.dataPlaneMode != "not-applicable"'
    expression: 'workload.verified.plaintextObserved == false'
    message: >-
      Plaintext traffic observed to {{ .Workload }} within
      {{ .Verified.Window }} from {{ .Verified.PlaintextSources }}.
```
