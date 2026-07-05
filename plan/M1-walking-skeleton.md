# M1 — Walking Skeleton (one control, end to end)

Branch: `m1-walking-skeleton`

## Goal
`openmeshguard scan` connects to a cluster read-only, collects a minimal resource set, resolves a PROVISIONAL effective mTLS posture (mesh + namespace-level PeerAuthentication only), and emits canonical JSON with inventory, workloadPostures, permissionSummary, and findings from one hardwired provisional check. Thin but real, top to bottom.

## Context
SPEC.md §6, §7 (partial), §12, §13. Contracts: canonical schema (authoritative), resolver types.

## Deliverables
- [x] `internal/collect`: typed read-only collectors for namespaces, pods, deployments/replicasets/statefulsets/daemonsets, services, PeerAuthentications. List-based, bounded concurrency, namespace-scoped or all-namespaces.
- [x] Fake-client unit tests including an **action audit test**: assert the only verbs ever issued are get/list (this test is permanent and grows with every collector).
- [x] Graceful permission degradation: forbidden/notfound per resource recorded into permissionSummary; scan continues.
- [x] `internal/normalize`: raw objects → normalized inventory + WorkloadInput (subset: no ports/DR/authz yet).
- [x] Provisional resolver implementation covering ONLY mesh-wide + namespace PA precedence, returning chains; port-level, workload-selector, and DR interplay explicitly return `unknown` with UnknownReason "not yet implemented (M2)".
- [x] `internal/output`: canonical JSON writer; every scan output validates against the schema (extend schema-test to run on real output).
- [x] One provisional finding path (hardcoded, replaced in M3): emit a finding when a namespace's resolved posture is permissive. Mark clearly `// PROVISIONAL: replaced by CEL engine in M3`.
- [x] Sidecar data-plane detection (istio-proxy container / injection labels); ambient detection stub returns unknown.

## Definition of Done
- `make test lint schema-test` green; action-audit test in place.
- Manual verification against a Kind cluster with Istio + a namespace PA (document exact commands in the task file when done).
- Scan output validates against the canonical schema with zero warnings.

## Manual verification commands (owner-run; not checked off by agent)

Prerequisites: `kind`, `kubectl`, `istioctl`, `jq`, and Docker running locally.

```bash
kind create cluster --name openmeshguard-m1
kubectl config use-context kind-openmeshguard-m1
istioctl install --set profile=default -y

kubectl create namespace omg-m1
kubectl label namespace omg-m1 istio-injection=enabled --overwrite

kubectl -n omg-m1 apply -f - <<'YAML'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  labels:
    app: api
spec:
  replicas: 1
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
        - name: api
          image: docker.io/hashicorp/http-echo:1.0
          args:
            - "-text=ok"
          ports:
            - containerPort: 5678
---
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  selector:
    app: api
  ports:
    - name: http
      port: 80
      targetPort: 5678
YAML

kubectl -n istio-system apply -f - <<'YAML'
apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: default
  namespace: istio-system
spec:
  mtls:
    mode: STRICT
YAML

kubectl -n omg-m1 rollout status deploy/api --timeout=180s
go run ./cmd/openmeshguard scan --context kind-openmeshguard-m1 --namespace omg-m1 > /tmp/openmeshguard-m1-mesh-strict.json
OPENMESHGUARD_SCHEMA_REPORT=/tmp/openmeshguard-m1-mesh-strict.json make schema-test

jq -e '
  any(.workloadPostures[]; .workload.namespace == "omg-m1" and .workload.name == "api" and .mtls.effective == "strict") and
  ([.findings[] | select(any(.resources[]; .namespace == "omg-m1" and .name == "api"))] | length == 0)
' /tmp/openmeshguard-m1-mesh-strict.json

kubectl -n omg-m1 apply -f - <<'YAML'
apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: default
  namespace: omg-m1
spec:
  mtls:
    mode: PERMISSIVE
YAML

go run ./cmd/openmeshguard scan --context kind-openmeshguard-m1 --namespace omg-m1 > /tmp/openmeshguard-m1.json
OPENMESHGUARD_SCHEMA_REPORT=/tmp/openmeshguard-m1.json make schema-test

jq -e '
  .schemaVersion == "v1alpha1" and
  .scan.scope.allNamespaces == false and
  .scan.scope.namespaces == ["omg-m1"] and
  .inventory.counts.deployments == 1 and
  any(.workloadPostures[]; .workload.namespace == "omg-m1" and .workload.name == "api" and .mtls.effective == "permissive") and
  any(.findings[]; .controlId == "MG-MTLS-001" and any(.resources[]; .namespace == "omg-m1" and .name == "api"))
' /tmp/openmeshguard-m1.json
```

Expected: both `jq -e` commands exit `0`. The first scan proves a scoped namespace scan still sees the mesh-wide STRICT PeerAuthentication in `istio-system`; the second scan proves a namespace PERMISSIVE PeerAuthentication overrides it and emits the provisional M1 finding.

```text
true
```

Cleanup:

```bash
kind delete cluster --name openmeshguard-m1
```

## Out of scope
Full resolver semantics (M2), CEL (M3), HTML/SARIF (M6), Prometheus (M7).

## Deferred
- Optimize `internal/normalize` pod-to-workload matching; the current M1 path is correct but still scans pods per workload.
- Implement Istio duplicate PeerAuthentication tie-breaks in M2 using the policy-ordering semantics documented at https://istio.io/latest/docs/concepts/security/#peer-authentication. M2 will need a reviewed `CreationTimestamp`/oldest-policy contract in the normalized resolver inputs; that touches the frozen exported resolver shape and requires human approval before changing it.
- Avoid redundant resolver sorting once M2 introduces richer PeerAuthentication precedence tables.
- Compile the canonical JSON schema once per schema-test run if validation cost becomes material.
- Move per-namespace existence/list preparation behind the bounded collector runner if larger scoped scans show startup latency.
