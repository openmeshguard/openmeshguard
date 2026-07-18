# RBAC profiles

OpenMeshGuard has no mutating Kubernetes permissions. Bind only the profile for
the desired scan scope; add optional evidence roles independently.

| Profile rule | Why it is needed | Degradation when absent | Optional |
|---|---|---|---|
| namespaces | Mesh-enrollment labels and governance context | Namespace labels and classification are unknown | No (cluster scan only) |
| pods, services | Sidecar detection, workload evidence, service targets | Data-plane mode, service inventory, and related findings become unknown | No |
| endpointslices | Service-to-workload endpoint mapping | Endpoint-backed service evidence becomes unknown | No |
| deployments, replicasets, statefulsets, daemonsets | Normalized workload identity and pod templates | Affected workload kinds are absent or unknown | No |
| networking.istio.io | Client TLS, routing, and visibility policy | Controls depending on the missing traffic-policy type become unknown | No |
| security.istio.io | Effective mTLS, authentication, and authorization policy | Security posture depending on the denied policy scope becomes unknown | No |
| telemetry.istio.io | Declared telemetry configuration | Telemetry-configuration controls become unknown | No |
| gateway.networking.k8s.io | Gateway listeners, routes, references, and TLS policy | Gateway controls become unknown | No |
| nodes add-on | Placement and topology corroboration | Node-backed evidence remains unavailable | Yes |
| events add-on | Recent rollout and policy failure context | Event evidence is omitted | Yes |
| control-plane ConfigMaps add-on | Non-secret mesh defaults and metadata | That control-plane evidence is unavailable | Yes |

The namespace Role cannot grant access to cluster-scoped Namespaces or
GatewayClasses. A namespace scan therefore records those unavailable evidence
sources rather than broadening access. Prometheus uses its configured HTTP
endpoint and needs no Kubernetes RBAC. Vendor APIs are not pre-authorized; a
vendor-specific add-on must name its exact non-secret API resources before use.

No profile grants Secrets, `watch`, write verbs, token creation,
exec/attach/port-forward, or impersonation.
