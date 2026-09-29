# Runtime verification demonstration

Run `make kind-up`, `make build`, then `sh test/e2e/runtime-verification.sh`.
The harness requires Docker, kubectl, curl, jq, and Go. It uses only the named
OpenMeshGuard E2E Kind cluster. Do not run concurrently with `make e2e`.

The administrator harness installs a sidecar receiver with PERMISSIVE mTLS,
a statically configured Prometheus instance, and two bounded traffic Jobs.
Each Job sends HTTP requests and TCP connections; one has a sidecar and the
other does not. Acceptance requires actual destination-reporter counters for
both protocols and both `none` and `mutual_tls` security labels, a nonzero
plaintext finding, an mTLS event share strictly between zero and one, and a
schema-valid canonical report. The event share mixes request and connection
counts; it is not a percentage of all application requests.

Fixture setup, ServiceAccount token issuance, and localhost port forwarding
are administrator harness actions. The scanner runs using a separate
ServiceAccount with the published read-only cluster role. It receives only
the configured localhost Prometheus URL. No scanner RBAC additions are needed.
Temporary credentials and forwarding are removed on exit; fixture resources
remain for inspection until `make kind-down` removes the disposable cluster.
Results are written under `.e2e/runtime-results/` (including the report and
raw aggregate metric evidence).

This demonstration validates sidecar destination telemetry only. Ambient
runtime verification and source attribution remain unsupported. It is an
explicit, optional live integration test; it does not add Prometheus or
traffic generators to the default acceptance suite.
