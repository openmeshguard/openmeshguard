# Verify release binaries

Release archives and `checksums.txt` each have a Sigstore bundle. Keyless signing
uses the repository's tag-triggered GitHub Actions release workflow. Verification
must constrain both the workflow identity and GitHub's OIDC issuer; accepting an
arbitrary valid signature does not establish the publisher.

## Verify a selected archive

Install [Cosign v3](https://docs.sigstore.dev/cosign/system_config/installation/)
separately. The example below downloads the macOS arm64 archive; change `archive`
to the Linux/macOS and amd64/arm64 filename for your machine. Requires `curl`,
`cosign`, and `shasum` (on Linux, `sha256sum --check` can replace `shasum -a 256 -c`).
Run in a new empty directory.

```sh
tag=v0.1.0
archive=openmeshguard_0.1.0_darwin_arm64.tar.gz
base="https://github.com/openmeshguard/openmeshguard/releases/download/$tag"
for file in "$archive" "$archive.sigstore.json" checksums.txt checksums.txt.sigstore.json; do
  curl --fail --location --output "$file" "$base/$file" || exit 1
done
identity="https://github.com/openmeshguard/openmeshguard/.github/workflows/release.yml@refs/tags/$tag"
cosign verify-blob --certificate-identity "$identity" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --bundle checksums.txt.sigstore.json checksums.txt || exit 1
cosign verify-blob --certificate-identity "$identity" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --bundle "$archive.sigstore.json" "$archive" || exit 1
awk -v archive="$archive" '$2 == archive { print; found++ } END { if (found != 1) exit 1 }' \
  checksums.txt > selected-checksum.txt || exit 1
shasum -a 256 -c selected-checksum.txt || exit 1
tar -xzf "$archive" openmeshguard
./openmeshguard version
```

Expect `version=v0.1.0`. Only run the binary after all signature and checksum
checks succeed. Signature verification uses Sigstore trust infrastructure; the
scanner itself connects only to your configured cluster API (and, when M7 ships,
configured Prometheus endpoint).

## Maintainer release procedure

1. Merge the M6.5 branch after independent reviews and green build/unit/lint/schema
   and Kind acceptance proofs. Enable private vulnerability reporting before
   publishing SECURITY.md. Create an annotated `v0.1.0` tag on that reviewed commit
   and push the tag; do not move a published version tag.
2. An authenticated preflight rejects an already public release for the exact
   tag before any upload; only an absent release or confirmed draft proceeds.
   Unexpected API, authentication, or network errors fail closed. The workflow builds four archives using pinned GoReleaser and Cosign,
   signs every archive plus the checksum file using GitHub OIDC, and uploads a
   **draft** release. It downloads those artifacts afresh and checks all five
   bundles against the exact tag workflow identity, every checksum, and the
   native binary's version.
3. A clean installation uses fresh GOPATH, module/build caches, and GOBIN,
   `GOWORK=off`, and `GOPROXY=direct` so a new tag does not depend on proxy indexing.
   The workflow creates a pinned disposable Kind/Istio cluster, completes an
   actual scan with the downloaded binary, validates its canonical JSON, and
   runs the default `dev` build through fixture, RBAC, and no-write acceptance.
4. The workflow removes its cluster and publishes only after verification
   succeeds. Failed verification leaves the draft unpublished. Inspect and fix
   the failure before publication; never bypass a failed signature or scan test.
   Do not overwrite already public release assets.

The all-archive verifier is also runnable from a checkout with authenticated
`gh`, Cosign v3, Python 3, and tar. The destination must not already exist:

```sh
./scripts/release/verify-download.sh v0.1.0 /tmp/openmeshguard-release-verification
/tmp/openmeshguard-release-verification/bin/openmeshguard scan \
  --context my-cluster --all-namespaces > /tmp/openmeshguard-release-report.json
OPENMESHGUARD_SCHEMA_REPORT=/tmp/openmeshguard-release-report.json \
  go test ./internal/output -run '^TestExternalScanOutputMatchesSchema$' -count=1
```

For draft releases, authenticate `gh` with repository access. The workflow's
`GH_TOKEN` is repository-scoped and needs `contents: write` to upload/publish and
`id-token: write` to sign; no additional long-lived signing key is stored.

Local packaging validation does not claim publication or signing:

```sh
goreleaser check
goreleaser release --snapshot --clean --skip=sign,publish
```

`make build` explicitly stamps `dev`; release packaging stamps the tag. Keep
release binaries out of the dev-version fixture comparison path.
