# Ambient basic

These fixtures prove the live ambient producer chain rather than resolver-only
tables. Both workloads are enrolled with `istio.io/dataplane-mode=ambient` and
selected by the same L7 `AuthorizationPolicy` shape. The ready case selects a
real Gateway API waypoint that reaches `Programmed=True`; the missing case
selects a nonexistent waypoint and must resolve to
`waypoint-policy-unenforced`. The unavailable case uses an acceptance-only
namespace-scoped scanner identity that can read workload and mesh-root policy
but cannot observe the selected cross-namespace waypoint, so the resolver must
preserve that evidence gap as `unknown`.
