# Development and acceptance testing

## Local prerequisites

The unit gates need Go, a POSIX shell, `tar`, `jq`, and either `shasum` or
`sha256sum`. The M4 acceptance harness additionally needs Docker, `kubectl`,
and `curl`. A matching system Kind or istioctl binary is reused. Otherwise the
harness downloads the declared release into the ignored `.e2e/bin` directory
with bounded retries and timeouts, then verifies its platform-specific SHA-256
before execution.

`versions.yaml` is the only version source for:

- the Kind CLI and its platform checksums;
- the digest-pinned Kind node image (and therefore Kubernetes version); and
- istioctl, its platform checksums, and the installed Istio control/data plane.

Do not copy those values into scripts or workflows. Updating this file is the
future version-matrix automation hook.

## Kind lifecycle

From a clean worktree with Docker running:

```bash
make kind-up e2e kind-down
```

`kind-up` refuses to reuse an existing `openmeshguard-e2e` cluster. `e2e`
always rebuilds `bin/openmeshguard`, resets the fixture namespaces, runs eight
small golden scans plus one all-namespaces ClusterRole scan, schema-validates
all nine reports, and executes the RBAC/audit proofs. `kind-down` deletes the
disposable cluster.

A failed local E2E run intentionally leaves the cluster available for
inspection; collect what you need and run `make kind-down`. In CI, one failure
step owns diagnostics collection and the always-step then tears the cluster
down, so evidence is captured before deletion and is not overwritten.

Set `UPDATE_GOLDEN=1` only when intentionally reviewing a behavior change:

```bash
UPDATE_GOLDEN=1 make e2e
```

Update mode runs every schema and semantic assertion before copying. Those
guards require the exact control/status set declared for each fixture,
non-empty chains for every resolved posture and its findings, the expected
live precedence chain, the critical disabled-mTLS finding, and non-vacuous
namespace-RBAC degradation. Mutation tests prove that a missing control,
posture chain, or finding chain is rejected before update mode can copy. The
comparison replaces only `generatedAt` and the kubeconfig context, after first
proving both fields were emitted by the raw scanner output. No posture evidence
is normalized away.

`OPENMESHGUARD_E2E_STATE_DIR` may be absolute or relative to the repository.
Relative overrides are canonicalized before report paths are passed to Go
schema tests. Host paths are emitted as quoted YAML scalars in the generated
Kind configuration, preserving spaces, `#`, and apostrophes.

## Fixture coverage boundary

The live scanner resolves mesh/namespace STRICT, namespace PERMISSIVE, and a
namespace-STRICT versus workload-DISABLE precedence conflict. The
port-level-override and DestinationRule-contradiction manifests are also
verified to exist live, but their goldens intentionally retain current
unavailable evidence: workload ports and DestinationRule collection have no
producers yet. Injection-disabled membership remains unknown until M6 owns
ambient enrollment detection. M4 does not silently wire any of those deferred
inputs.

## RBAC identities and proof

The harness keeps four identities visibly separate:

- `fixture-manager` is an ephemeral setup ServiceAccount bound to the
  disposable cluster's existing `cluster-admin` role. It applies and resets
  fixtures and is never used to scan.
- `scanner-cluster` is bound to `openmeshguard-cluster-scan`.
- `scanner-namespace` is bound to `openmeshguard-namespace-scan` only in
  `omg-strict`.
- `audit-probe` has no RBAC binding. Its one denied ConfigMap create is the
  positive control for the API-server audit path, never a scanner call.

Before scanning, the harness compares each referenced live Role/ClusterRole
proof shape to the published manifest, including rules and `aggregationRule`,
and rejects any direct, User, or service-account-group resource binding beyond
the expected profile. `kind-up` removes Kubernetes' default
`system:basic-user` binding because that role grants `create` on self-review
resources to every authenticated ServiceAccount. E2E fails if the binding
reappears. The only allowed authenticated-group exceptions are
`system:discovery`, `system:public-info-viewer`, and
`system:service-account-issuer-discovery`; their live ClusterRoles must contain
only non-resource `get` rules with no API groups or resources. Any other
binding, or a widened default discovery role, fails the proof.

The namespace scanner can read its workload namespace but cannot list the
cluster-scoped Namespace object or root-namespace PeerAuthentication. Its
golden contains denied permission-summary entries and three unknown mTLS
findings instead of a failed scan.

All ServiceAccount tokens last ten minutes. Kubeconfigs are created as mode
0600 inside a mode-0700 random directory and deleted by an exit trap. The
scanner identities never request tokens or impersonate users.

## No-write audit choice

M4 uses kube-apiserver audit logging inside Kind rather than an auditing proxy.
This observes authenticated requests at the API server. The API server uses
`blocking-strict` audit mode so a successful response cannot outrun the proof
log. The metadata-only policy records the scanner identities and the separate
unbound audit probe.

After setup and RBAC propagation checks, the harness truncates the log. It
first requires the probe's denied write event, then runs the scanners and
asserts:

- both scanner identities produced events;
- every scanner event was `get` or `list` for a SPEC section 13 resource;
- no scanner event targeted Secrets or any subresource;
- the namespace scanner received a 403 for root-namespace
  PeerAuthentications; and
- the separate probe's ConfigMap create was recorded and denied.

The latest audit artifact is `.e2e/results/audit.jsonl`. CI also retains pod,
event, control-plane, Kind, and audit diagnostics on failure.

## Recorded M4 proof

Recorded locally on 2026-07-16 (America/Chicago):

| Component | Exact version |
|---|---|
| Kind | v0.31.0 |
| Kubernetes node | `kindest/node:v1.35.0@sha256:452d707d4862f52530247495d180205e029056831160e22870e37e3f6c1ac31f` |
| Istio | 1.30.2 |

The final clean lifecycle and determinism proof was:

| Target | Duration | Result |
|---|---:|---|
| `make kind-up` | 46s | green in the exact combined lifecycle using `/tmp/openmeshguard review # state` |
| `make e2e` | 27s | eight goldens matched; nine reports schema-valid; RBAC/audit proofs green |
| `make kind-down` | 0s | green in the exact combined lifecycle |

On a second fresh cluster, setup took 43s and two consecutive ordinary
`make e2e` runs completed in 24s and 35s with identical goldens and the same 80
approved scanner API events. Teardown took 0s.

The audit contained 71 cluster-scanner list events, nine namespace-scanner
list events, and exactly one separate audit-probe create event with a 403.
The `system:basic-user` binding remained absent, and all three allowed default
roles were verified as non-resource `get` only. No token-bearing kubeconfig
remained after any run.

The pinned Istio minor now provides the version input needed by the M2 deferred
root-namespace-selector decision. M4 does not change that resolver behavior;
the version-specific semantics remain deferred.
