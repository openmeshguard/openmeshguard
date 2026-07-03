# M0 — Repo Scaffold and CI

Branch: `m0-scaffold`

## Goal
A buildable, lint-clean, CI-green skeleton with the contracts wired in as enforced artifacts — no scanner logic yet.

## Context
SPEC.md §12, §13, §18. Contracts: all of `docs/contracts/`.

## Deliverables
- [x] Go module init (`github.com/openmeshguard/openmeshguard`), directory layout per AGENTS.md.
- [x] Cobra CLI with `version` command and stubs for `scan`, `report`, `export`, `score` (stubs print "not implemented" and exit 2).
- [x] Move `docs/contracts/resolver_types.go` into `internal/resolver/types.go` (contract header comment intact); package compiles.
- [x] `Makefile`: build, test, lint, schema-test, kind-up, e2e, kind-down (kind targets may be placeholders until M4).
- [x] golangci-lint config + an import-purity check failing the build if `internal/resolver` imports client-go, net/http, or os.
- [x] Schema tooling: vendor `docs/contracts/canonical-json-schema.json` into a `schema-test` that validates a checked-in minimal example report fixture.
- [x] GitHub Actions: lint + test + schema-test on PR; Go version pinned.
- [x] `controls/` directory with an empty valid pack loaded by a placeholder test (format validation only — engine comes in M3).
- [x] Copy README.md, SPEC.md, LICENSE (Apache 2.0 canonical text) into place.

## Definition of Done
- `make build test lint schema-test` green locally and in CI.
- `openmeshguard version` prints version + resolverVersion placeholder.
- Import-purity check demonstrably fails when a client-go import is added to internal/resolver (prove in a throwaway commit, then revert).

## Out of scope
Any cluster access, any resolver logic, any CEL.

## Deferred
(record follow-ups here)
