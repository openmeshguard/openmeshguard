# Development and acceptance testing

## Local prerequisites

The unit gates need Go. The M4 acceptance harness additionally needs Docker,
`kubectl`, `curl`, `tar`, and `jq`. A matching Kind or istioctl binary is reused
when installed; otherwise the harness downloads the exact versions declared in
`versions.yaml` into the ignored `.e2e/bin` directory.

`versions.yaml` is the only version source for:

- the Kind CLI;
- the digest-pinned Kind node image (and therefore Kubernetes version); and
- istioctl plus the installed Istio control plane/data plane.

Do not copy those values into scripts or workflows. Updating the file is the
future version-matrix automation hook.

## Kind lifecycle

From a clean worktree with Docker running:

```bash
make kind-up e2e kind-down
```

`kind-up` refuses to reuse an existing `openmeshguard-e2e` cluster. `e2e`
builds the scanner, applies the sidecar fixtures, scans with both published RBAC
profiles, schema-validates and diffs seven scrubbed canonical reports, and
checks the API-server audit proof. `kind-down` deletes the disposable cluster.

Set `UPDATE_GOLDEN=1` only when intentionally reviewing a behavior change:

```bash
UPDATE_GOLDEN=1 make e2e
```

The comparison replaces `generatedAt` and the kubeconfig context with fixed
values. Workloads, permission entries, postures, findings, chains, and scores
remain untouched. The Go output path and engine already sort their slices and
map keys deterministically; no posture evidence is normalized away.

## RBAC identities and proof

The harness keeps three identities visibly separate:

- `fixture-manager` is an ephemeral setup ServiceAccount bound to the
  disposable cluster's existing `cluster-admin` role. It applies fixtures,
  published profiles, and bindings, and is never used to scan.
- `scanner-cluster` has exactly one explicit binding, to
  `openmeshguard-cluster-scan`.
- `scanner-namespace` has exactly one explicit binding, to
  `openmeshguard-namespace-scan` in `omg-strict`.

The harness asserts those scanner bindings before issuing scanner tokens. The
namespace scanner can read its workload namespace but cannot list the
cluster-scoped Namespace object or root-namespace PeerAuthentication. Its
golden therefore contains denied permission-summary entries and unknown mTLS
findings instead of a failed scan.

The setup identity requests short-lived ServiceAccount tokens; the scanner only
receives the finished kubeconfigs and never requests a token or impersonates a
user.

## No-write audit choice

M4 uses kube-apiserver audit logging inside Kind, rather than an auditing proxy.
This proves what the API server actually received after authentication and
authorization, including the namespace Role's denied root-namespace request.
The Kind kubeadm patch mounts `test/e2e/audit-policy.yaml` and a writable audit
directory into the API server.

The policy records metadata only for the two scanner ServiceAccounts. After all
fixture setup and token issuance, the harness truncates the audit log, runs the
scans, and asserts:

- both scanner identities produced events;
- every received verb was `get` or `list`;
- the namespace scanner received a `403` when listing PeerAuthentications in
  `istio-system`; and
- no other scanner verb occurred.

The most recent audit artifact is `.e2e/results/audit.jsonl`. CI uploads the
results directory on failure.

## Recorded M4 proof

Recorded locally on 2026-07-15 (America/Chicago):

| Component | Exact version |
|---|---|
| Kind | v0.31.0 |
| Kubernetes node | `kindest/node:v1.35.0@sha256:452d707d4862f52530247495d180205e029056831160e22870e37e3f6c1ac31f` |
| Istio | 1.30.2 |

The final clean lifecycle was:

| Target | Duration | Result |
|---|---:|---|
| `make kind-up` | 41s | green |
| `make e2e` | 21s | seven schema-valid goldens matched; 63 audited events, all get/list |
| `make kind-down` | 0s | green |

Before the final clean lifecycle, two consecutive ordinary `make e2e` runs on
the same cluster completed in 15s each with identical goldens and the same 63
get/list-only audit events. A separate fresh-cluster comparison also matched,
which prevents stale workload-controller objects from entering the goldens.

The same exact clean lifecycle was rerun after the closeout review changed the
harness, producing the 41s/21s/0s result above.

The pinned Istio minor now provides the version input needed by the M2 deferred
root-namespace-selector decision. M4 does not change that resolver behavior;
the version-specific semantics remain a follow-up.
