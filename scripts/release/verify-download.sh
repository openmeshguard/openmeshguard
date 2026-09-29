#!/bin/sh
# Download a public or authenticated draft release and verify its publisher.
# Requires gh, cosign >=3, python3, tar. The destination must not exist.
set -eu
umask 077
if [ "$#" -ne 2 ]; then
  echo "usage: $0 <vMAJOR.MINOR.PATCH> <new-directory>" >&2
  exit 2
fi
release_tag=$1
release_dir=$2
python3 - "$release_tag" <<'PY'
import re, sys
if not re.fullmatch(r'v[0-9]+\.[0-9]+\.[0-9]+', sys.argv[1]):
    raise SystemExit('expected a stable release tag, for example v0.1.0')
PY
mkdir "$release_dir"
release_dir=$(CDPATH= cd -- "$release_dir" && pwd)
gh release download "$release_tag" --repo openmeshguard/openmeshguard --dir "$release_dir"
identity="https://github.com/openmeshguard/openmeshguard/.github/workflows/release.yml@refs/tags/$release_tag"
version=${release_tag#v}
for artifact in checksums.txt \
  "openmeshguard_${version}_darwin_amd64.tar.gz" \
  "openmeshguard_${version}_darwin_arm64.tar.gz" \
  "openmeshguard_${version}_linux_amd64.tar.gz" \
  "openmeshguard_${version}_linux_arm64.tar.gz"; do
  cosign verify-blob --certificate-identity "$identity" \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com \
    --bundle "$release_dir/$artifact.sigstore.json" "$release_dir/$artifact"
done
# Require exactly the four expected archive checksums; verify bytes independently
# of platform-specific sha256sum/shasum output and flags.
python3 - "$release_dir" "$version" <<'PY'
import hashlib, pathlib, sys
root, version = pathlib.Path(sys.argv[1]), sys.argv[2]
expected = {f'openmeshguard_{version}_{os}_{arch}.tar.gz'
            for os in ('darwin', 'linux') for arch in ('amd64', 'arm64')}
seen = set()
for line in (root / 'checksums.txt').read_text().splitlines():
    digest, name = line.split(maxsplit=1)
    name = name.removeprefix('*')
    if name not in expected or name in seen:
        raise SystemExit(f'unexpected or duplicate checksum target: {name}')
    if hashlib.sha256((root / name).read_bytes()).hexdigest() != digest:
        raise SystemExit(f'checksum mismatch: {name}')
    seen.add(name)
if seen != expected:
    raise SystemExit(f'missing checksums: {expected - seen}')
PY
case $(uname -s) in
  Darwin) release_os=darwin ;;
  Linux) release_os=linux ;;
  *) echo "unsupported verification host" >&2; exit 2 ;;
esac
case $(uname -m) in
  arm64|aarch64) release_arch=arm64 ;;
  x86_64|amd64) release_arch=amd64 ;;
  *) echo "unsupported verification architecture" >&2; exit 2 ;;
esac
mkdir "$release_dir/bin"
tar -xzf "$release_dir/openmeshguard_${version}_${release_os}_${release_arch}.tar.gz" \
  -C "$release_dir/bin" openmeshguard
"$release_dir/bin/openmeshguard" version > "$release_dir/version.txt"
python3 - "$release_dir/version.txt" "$release_tag" <<'PY'
import pathlib, sys
if pathlib.Path(sys.argv[1]).read_text().splitlines()[0] != f'version={sys.argv[2]}':
    raise SystemExit('downloaded binary version does not match tag')
PY
cat "$release_dir/version.txt"
echo "Verified release binary: $release_dir/bin/openmeshguard"
