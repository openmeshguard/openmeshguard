# Sidecar basic acceptance fixtures

Each namespace has one workload and one focused resolver condition. The
cluster-Role scanner runs once per namespace so every golden stays small and
diagnostic.

| Golden | Live condition | Expected current evidence |
|---|---|---|
| `strict.json` | mesh and namespace STRICT | strict |
| `permissive.json` | mesh STRICT, namespace PERMISSIVE | permissive |
| `port-level-override.json` | workload port 8080 DISABLE | unknown; workload ports have no producer |
| `dr-contradiction.json` | server STRICT, DestinationRule DISABLE | server strict; client contradiction unavailable |
| `not-in-mesh.json` | injection explicitly disabled | unknown; sidecar/ambient membership is not inferred |
| `unclassified.json` | unlabeled namespace, workload-level sidecar injection | sidecar with mesh STRICT |
| `namespace-role-degraded.json` | strict fixture under namespace-only RBAC | mTLS unknown after root policy denial |

The two unavailable-evidence goldens have same-basename `.note.md` files. They
are acceptance constraints, not future expected values: M4 does not wire the
M3-deferred ports or DestinationRule producers.
