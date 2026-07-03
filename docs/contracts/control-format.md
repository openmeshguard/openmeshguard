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
    title: Production workloads must have effective strict mTLS
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
        and check for DestinationRule TLS contradictions in the resolution chain.
      suggestedYAMLTemplate: peerauthentication-strict.tmpl
    frameworks:                   # tags only, never compliance claims
      - nist-csf-2.0/PR.DS
      - owasp-k8s/K01
```

## Semantics (binding on the engine)

1. **Scope** determines the iteration unit. `workload` controls evaluate once per entry in `workloadPostures`; `namespace` controls once per mesh namespace; `resource` controls once per matching resource kind (declared via `match.kinds`).
2. **Environments** filter by resolved classification. A control scoped to `production` never evaluates unclassified namespaces — those are covered separately by MG-ENV-001. Empty list = evaluate everywhere.
3. **`applicability`** is a CEL expression. False ⇒ finding status `not-applicable` (not a pass, not counted in pass rates).
4. **`requires`** lists dotted paths into the evaluation input. If any resolves to an unknown/unavailable value, the engine emits status `unknown` with `unknownReason` set, and never evaluates `expression`. This is how "unknown is never pass and never fail" is enforced mechanically — controls cannot forget it.
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

## Validation

`openmeshguard controls validate <path>` (and pack loading at scan time) must reject:

- Unknown fields, missing required fields, malformed IDs (`^MG-[A-Z]+-[0-9]{3}$` for built-ins; user packs may use their own prefix but must match `^[A-Z]+-[A-Z]+-[0-9]{3}$`).
- CEL expressions that fail compilation or reference variables outside the declared scope.
- `runtime` evidenceType controls that do not `require` a `verified.*` field.
- Duplicate control IDs across all loaded packs.

Error messages must point at the pack file, control ID, and CEL compile position.

## Worked examples

```yaml
  - id: MG-GW-001
    title: Public gateways must not use wildcard hosts
    category: exposure
    severity: high
    evidenceType: config
    scope: resource
    match: { kinds: [Gateway] }        # istio networking Gateway
    requires: [resource.servers]
    applicability: 'resource.isPubliclyExposed'
    expression: '!resource.servers.exists(s, s.hosts.exists(h, h == "*" || h.startsWith("*.")))'
    message: 'Gateway {{ .Resource }} exposes wildcard host(s) publicly.'

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
