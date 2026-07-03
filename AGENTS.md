# AGENTS.md

Instructions for AI coding agents working in this repository. Read this file fully before making changes.

## Project

OpenMeshGuard is a read-only Go CLI that scans Istio service meshes and reports **effective** security posture (resolved from Istio's real policy evaluation semantics), **verified** posture (corroborated by Prometheus telemetry), and governance posture (ownership, environments, exceptions) — with evidence chains for every finding.

Authoritative documents, in order of precedence when they conflict:

1. `docs/contracts/` — canonical JSON schema, control format, resolver types. **These are frozen interfaces. Never change them without explicit human approval. If a task seems to require a contract change, stop and flag it.**
2. `plan/M*.md` — the milestone task files. Work on exactly one milestone at a time.
3. `SPEC.md` — full product spec. Use it for context and rationale; task files define your scope.

If you find a genuine conflict between these documents, do not guess. Surface it in your summary and ask.

## Hard constraints (non-negotiable)

These are product guarantees, not style preferences. Violating any of them is a failed task even if tests pass.

1. **Read-only, always.** The scanner performs only `get` and `list` against the Kubernetes API. Never `watch`, never any write verb (`create`, `update`, `patch`, `delete`), never `impersonate`, never token creation, never `exec`/`attach`/`portforward`.
2. **Never read Secrets.** No `secrets` access of any kind, including for multi-cluster remote-secret detection. Multi-cluster participation is inferred from gateways, labels, and config — never from Secret contents.
3. **Typed clients first.** Use `k8s.io/client-go`, `istio.io/client-go`, and `sigs.k8s.io/gateway-api` typed clients for supported resources. Dynamic/unstructured access is a fallback for unknown or vendor-specific resources only.
4. **Bounded API usage.** List-based collection with bounded concurrency. No per-object GET loops, no retries that escalate scope, no long-lived connections in the one-shot CLI path.
5. **The resolver is a pure function.** `internal/resolver` takes normalized inputs and returns posture + resolution chains. It must never import client-go or perform I/O. This is enforced by a lint rule / import test — keep it passing.
6. **Controls are data.** Posture rules live in YAML control packs with CEL expressions (`docs/contracts/control-format.md`). Never hardcode a control's logic in Go. Go code provides resolver outputs and the CEL environment; controls decide pass/fail.
7. **Unknown is never pass and never fail.** Missing permissions, missing telemetry, or missing classification produce explicit `unknown` states carried through findings, reports, and exit codes. Never silently skip, never default to compliant, never default to violating.
8. **Declared vs verified are never blended.** Config-derived and telemetry-derived conclusions are separate fields end to end.
9. **Every resolved conclusion carries its resolution chain** — the ordered resources and rules that produced it. A resolver result without a chain is a bug.
10. **Graceful degradation.** Permission or telemetry failures degrade the affected findings and are recorded in the permission summary. They never abort the whole scan.

## Repository layout

```
cmd/openmeshguard/        CLI entrypoints (cobra): scan, report, export, score, version
internal/collect/         Read-only collectors (typed clients, fake-client tests)
internal/normalize/       Raw objects -> normalized inventory model
internal/resolver/        Effective posture resolution (PURE — no I/O, no client-go)
internal/engine/          CEL rule engine, control pack loading/validation
internal/telemetry/       Prometheus queries for runtime-verified controls
internal/context/         Environment classification, ownership, exceptions
internal/output/          Canonical JSON, SARIF, static HTML report, scores
controls/                 Built-in control packs (YAML, embedded via go:embed)
docs/contracts/           Frozen interface contracts
deploy/rbac/              Published RBAC profiles (Role, ClusterRole, add-ons)
test/fixtures/            Kind fixture manifests + expected findings (golden files)
test/e2e/                 Kind acceptance harness
plan/                     Milestone task files
```

## Toolchain and conventions

- Go: latest stable (1.24+). Module path: `github.com/openmeshguard/openmeshguard` (placeholder until org is final — keep it consistent).
- CLI: `spf13/cobra`. CEL: `github.com/google/cel-go`. No web frameworks; the HTML report is a static template rendered from canonical JSON.
- Lint: `golangci-lint` with the repo config; `make lint` must pass.
- Errors: wrap with `fmt.Errorf("...: %w", err)`; no panics outside `main`.
- Logging: `log/slog`, structured, quiet by default, `-v` for debug. Never log resource contents at info level.
- Tests: table-driven. Collectors use fake clients with action audits. Resolver uses pure input/output tables. E2E uses Kind (`make e2e`).
- Every exported type in `internal/output` must round-trip against `docs/contracts/canonical-json-schema.json` — there is a schema validation test; keep it green.

## Commands

```
make build        # build the CLI
make test         # unit tests (no cluster required)
make lint         # golangci-lint + import-purity check for internal/resolver
make schema-test  # validate outputs against docs/contracts schema
make kind-up      # create local Kind cluster + install pinned Istio
make e2e          # run acceptance fixtures against Kind
make kind-down    # tear down
```

Unit tests and lint must pass before you consider any task done. E2E is required only where a milestone's Definition of Done says so.

## Workflow rules

- One milestone per branch. Branch name: `m<N>-<slug>` (e.g., `m2-mtls-resolver`).
- Read the milestone file in `plan/` first. Its **Definition of Done is the only exit criterion.** Do not expand scope; note follow-up ideas in the task file's "Deferred" section instead of implementing them.
- Check off the milestone checklist items as you complete them and keep the task file updated — it is the source of truth for progress.
- Write tests alongside or before implementation. For resolver work, the test tables ARE the specification: if a table's expected output seems wrong versus Istio semantics, stop and flag it for human review with a link to the relevant Istio documentation — do not "fix" expected outputs to match your implementation.
- Small commits with imperative messages. A milestone ends with all DoD boxes checked, `make test lint schema-test` green, and a summary of decisions made and anything flagged.

## Things you must never do

- Add a Kubernetes verb, resource, or API group to the RBAC profiles or client calls beyond what `SPEC.md` §13 allows, without human approval.
- Modify anything in `docs/contracts/` without human approval.
- Invent Istio semantics. When the resolver's behavior for an edge case is unclear, cite the upstream Istio docs in a comment and flag uncertainty rather than guessing silently.
- Weaken, skip, or delete a failing test to make a milestone pass.
- Add network calls to anything other than the target cluster API and the configured Prometheus endpoint.
