# Sidecar basic acceptance fixtures

Each namespace has one workload and one focused condition. STRICT,
PERMISSIVE, namespace-vs-workload precedence, workload ports, and
DestinationRule client TLS exercise the live mTLS resolver. The cluster-Role
scanner runs once per namespace so every golden stays small and diagnostic.

| Golden | Live condition | Expected current evidence |
|---|---|---|
| `strict.json` | mesh and namespace STRICT | strict |
| `permissive.json` | mesh STRICT, namespace PERMISSIVE | permissive |
| `workload-conflict.json` | namespace STRICT, workload DISABLE | disabled; workload policy wins |
| `port-level-override.json` | workload port 8080 DISABLE | disabled; the only Service-bound port is disabled |
| `dr-contradiction.json` | server STRICT, DestinationRule DISABLE | server strict; client contradiction true |
| `not-in-mesh.json` | injection explicitly disabled | unknown; sidecar/ambient membership is not inferred |
| `unclassified.json` | unlabeled namespace, workload-level sidecar injection | sidecar with mesh STRICT |
| `namespace-role-degraded.json` | strict fixture under namespace-only RBAC | mTLS and authorization unknown after root policy denial |

The port and DestinationRule fixtures are resolved in M5 from typed collection
and explicit availability evidence. The injection-disabled case remains
unknown until M6 owns ambient enrollment detection.

`cases.tsv` also declares each fixture's exact expected `controlId=status`
set. E2E verifies that set and all required resolution chains before golden
update mode can copy a report. A permanent bijection check rejects any stale
golden without a live case and any declared case without a golden.
