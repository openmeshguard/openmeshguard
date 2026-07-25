# Consumable outputs and CI contract

Canonical JSON is OpenMeshGuard's single source of truth. HTML, SARIF, the
human-readable score command, and CI exit decisions read a canonical report;
they do not run the resolver or control engine again.

## Zero-config first run

No scan config, ownership import, exception file, or Prometheus endpoint is
required:

```bash
openmeshguard scan --context my-cluster --all-namespaces > openmeshguard.json
openmeshguard report \
  --input openmeshguard.json \
  --format html \
  --output openmeshguard.html
```

The first screen contains:

- a Declared / Verified / Unknown summary table;
- category grades and the numeric score;
- permission and evidence availability;
- environment-classification coverage;
- unknown findings and missing telemetry as prominent first-class states; and
- every canonical finding, including expandable resolution chains.

With no context files, unclassified namespaces and unresolved owners remain
visible. With no Prometheus input, the Verified column says runtime
verification is unavailable; it does not disappear and it is not treated as a
pass. When M7 supplies runtime posture, the report preserves the canonical
`corroborated`, `contradicted`, `no-traffic-observed`, and `unknown` states and
their window, traffic share, plaintext, and source evidence without blending
them. The generated HTML is one server-less file with inline CSS, no scripts,
no external assets, and a content-security policy that disables network
loading.

All projection commands read `--input -` from stdin and write `--output -` to
stdout by default, so the same workflow can be piped:

```bash
openmeshguard scan --all-namespaces |
  openmeshguard report --format html --output openmeshguard.html
```

## SARIF 2.1.0 projection

```bash
openmeshguard export \
  --input openmeshguard.json \
  --format sarif \
  --output openmeshguard.sarif
```

Every object in canonical `findings` becomes exactly one SARIF `result`.
Unique canonical `controlId` values referenced by those findings become SARIF
rules; the canonical schema does not contain a catalog of passing controls, so
the exporter does not load control packs or invent additional rules.

Severity mapping for confirmed open and excepted findings is:

| Canonical severity | SARIF level |
| --- | --- |
| `critical`, `high` | `error` |
| `medium` | `warning` |
| `low`, `info` | `note` |

Status is represented independently:

| Canonical status | SARIF representation |
| --- | --- |
| `open` | `kind: fail`, mapped severity |
| `excepted` | `kind: fail`, mapped severity, accepted external suppression |
| `unknown` | `kind: review`, `level: warning`; original severity retained in properties |
| `not-applicable` | `kind: notApplicable`, `level: none` |

Every result retains the canonical finding ID, status, severity, confidence,
evidence sources, unknown reason, exception, and resolution chain in SARIF
properties. Kubernetes resource references become `kubernetes:` artifact
locations and logical locations. Tests validate output against the official
OASIS SARIF 2.1.0 schema incorporating Errata 01 and assert exact finding
count/ID parity.

## Score model

```bash
openmeshguard score --input openmeshguard.json
openmeshguard score --input openmeshguard.json --namespace payments
```

`score` prints only values already stored under canonical `scores`. Category
grades use the control engine's pass rates. The published 100-point weights
are control-pack data:

| Frozen control category | Weight | SPEC dimensions represented |
| --- | ---: | --- |
| `mtls` | 25 | effective mTLS |
| `authz` | 25 | authorization coverage |
| `exposure` | 25 | exposure 15 + egress 10 |
| `governance` | 20 | ownership 10 + exception hygiene 10 |
| `lifecycle` | 5 | lifecycle baseline |

Unknown categories have no pass rate and are excluded from the weighted
denominator; unknown state remains separately reported. A published weighted
dimension with no current controls, such as lifecycle in v0, remains an
explicit unknown category instead of disappearing. Namespace scores use the
weights for their resolved environment. The cluster score uses global
canonical category aggregates so cluster-scoped controls such as exception
hygiene cannot disappear behind namespace rollups. An open critical finding
caps the affected namespace score and the cluster score at 59. Excepted,
unknown, and not-applicable findings do not apply the cap; an expired exception
is already represented canonically as an open finding and therefore remains
active risk.

`score --namespace` prints the canonical namespace score followed by clearly
labeled cluster category grades because the frozen schema does not contain
per-namespace category grades. The command never derives those missing fields.

Weights and the critical cap can be overridden through the existing
scan-config parameter mechanism. Supply the complete weight map:

```yaml
controls:
  parameters:
    environments:
      production:
        scoring:
          weights:
            mtls: 30
            authz: 30
            exposure: 20
            governance: 15
            lifecycle: 5
          criticalCap: 49
```

## CI exit codes

Both `scan` and `score` accept `--fail-on <severity>` and
`--fail-on-unknown`:

```bash
openmeshguard scan \
  --all-namespaces \
  --fail-on high > openmeshguard.json

openmeshguard score \
  --input openmeshguard.json \
  --fail-on high
```

| Exit code | Meaning |
| ---: | --- |
| 0 | Scan/projection succeeded and no configured failure condition matched |
| 1 | An open finding met or exceeded `--fail-on`, or `--fail-on-unknown` matched |
| 2 | Scan, input, validation, projection, or output error |

Severity order is `critical > high > medium > low > info`, and the boundary is
inclusive. Only canonical `status: open` findings participate in severity
thresholds. Unknown findings never affect exit status unless
`--fail-on-unknown` is explicitly set. Excepted and not-applicable findings do
not fail CI. Before any projection or threshold decision, the complete input
is validated against the embedded build-time copy of the frozen canonical
schema. A parity test requires that copy to remain byte-equivalent after JSON
compaction to `docs/contracts/canonical-json-schema.json`.
