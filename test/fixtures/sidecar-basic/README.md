# Sidecar basic acceptance fixtures

Each namespace has one workload and one focused condition. STRICT,
PERMISSIVE, and namespace-vs-workload precedence exercise the live M2 resolver.
The port, DestinationRule, and injection-disabled cases instead lock the
scanner's current unavailable-evidence boundaries; their deferred inputs are
not silently synthesized by M4. The cluster-Role scanner runs once per
namespace so every golden stays small and diagnostic.

| Golden | Live condition | Expected current evidence |
|---|---|---|
| `strict.json` | mesh and namespace STRICT | strict |
| `permissive.json` | mesh STRICT, namespace PERMISSIVE | permissive |
| `workload-conflict.json` | namespace STRICT, workload DISABLE | disabled; workload policy wins |
| `port-level-override.json` | workload port 8080 DISABLE | unknown; workload ports have no producer |
| `dr-contradiction.json` | server STRICT, DestinationRule DISABLE | server strict; client contradiction unavailable |
| `not-in-mesh.json` | injection explicitly disabled | unknown; sidecar/ambient membership is not inferred |
| `unclassified.json` | unlabeled namespace, workload-level sidecar injection | sidecar with mesh STRICT |
| `namespace-role-degraded.json` | strict fixture under namespace-only RBAC | mTLS unknown after root policy denial |

The port and DestinationRule unavailable-evidence goldens have same-basename
`.note.md` files. They are acceptance constraints, not future expected values:
M4 does not wire the M3-deferred producers. The injection-disabled case
likewise remains unknown until M6 owns ambient enrollment detection.

`cases.tsv` also declares each fixture's exact expected `controlId=status`
set. E2E verifies that set and all required resolution chains before golden
update mode can copy a report. A permanent bijection check rejects any stale
golden without a live case and any declared case without a golden.
