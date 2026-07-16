#!/bin/sh

set -eu

E2E_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
E2E_STATE_DIR=${OPENMESHGUARD_E2E_STATE_DIR:-"$E2E_ROOT/.e2e"}
E2E_CLUSTER_NAME=${OPENMESHGUARD_E2E_CLUSTER_NAME:-openmeshguard-e2e}
E2E_CONTEXT="kind-$E2E_CLUSTER_NAME"
case "$E2E_STATE_DIR" in
	/*) ;;
	*) E2E_STATE_DIR="$E2E_ROOT/$E2E_STATE_DIR" ;;
esac
E2E_HARNESS_NAMESPACE=openmeshguard-e2e
E2E_FIXTURE_MANAGER=fixture-manager
E2E_CLUSTER_SCANNER=scanner-cluster
E2E_NAMESPACE_SCANNER=scanner-namespace

version_value() {
	key=$1
	value=$(awk -v key="$key" '
		$0 ~ "^[[:space:]]*" key ":[[:space:]]*" {
			sub("^[[:space:]]*" key ":[[:space:]]*", "")
			gsub(/^['\"']|['\"']$/, "")
			print
			exit
		}
	' "$E2E_ROOT/versions.yaml")
	if [ -z "$value" ]; then
		echo "versions.yaml missing required key: $key" >&2
		return 1
	fi
	printf '%s\n' "$value"
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "required command not found: $1" >&2
		exit 1
	fi
}

host_os() {
	case $(uname -s) in
		Darwin) echo darwin ;;
		Linux) echo linux ;;
		*) echo "unsupported operating system: $(uname -s)" >&2; exit 1 ;;
	esac
}

release_os() {
	case $(host_os) in
		darwin) echo osx ;;
		linux) echo linux ;;
	esac
}

host_arch() {
	case $(uname -m) in
		x86_64|amd64) echo amd64 ;;
		arm64|aarch64) echo arm64 ;;
		*) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
	esac
}

checksum_key() {
	prefix=$1
	os=$2
	arch=$3
	case "$prefix:$os:$arch" in
		kind:darwin:amd64) echo kindSHA256DarwinAMD64 ;;
		kind:darwin:arm64) echo kindSHA256DarwinARM64 ;;
		kind:linux:amd64) echo kindSHA256LinuxAMD64 ;;
		kind:linux:arm64) echo kindSHA256LinuxARM64 ;;
		istioctl:osx:amd64) echo istioctlSHA256OSXAMD64 ;;
		istioctl:osx:arm64) echo istioctlSHA256OSXARM64 ;;
		istioctl:linux:amd64) echo istioctlSHA256LinuxAMD64 ;;
		istioctl:linux:arm64) echo istioctlSHA256LinuxARM64 ;;
		*) echo "unsupported checksum platform: $prefix $os/$arch" >&2; return 1 ;;
	esac
}

verify_sha256() {
	path=$1
	want=$2
	if command -v shasum >/dev/null 2>&1; then
		got=$(shasum -a 256 "$path" | awk '{print $1}')
	elif command -v sha256sum >/dev/null 2>&1; then
		got=$(sha256sum "$path" | awk '{print $1}')
	else
		echo "required command not found: shasum or sha256sum" >&2
		return 1
	fi
	if [ "$got" != "$want" ]; then
		echo "SHA-256 mismatch for $path: got $got, want $want" >&2
		return 1
	fi
}

download() {
	url=$1
	output=$2
	curl -fsSL --retry 3 --retry-all-errors --connect-timeout 15 --max-time 180 "$url" -o "$output"
}

kind_binary() {
	want=$(version_value kind)
	if command -v kind >/dev/null 2>&1 && [ "$(kind version 2>/dev/null | awk '{print $2}')" = "$want" ]; then
		command -v kind
		return
	fi

	require_command curl
	mkdir -p "$E2E_STATE_DIR/bin"
	bin="$E2E_STATE_DIR/bin/kind"
	os=$(host_os)
	arch=$(host_arch)
	checksum=$(version_value "$(checksum_key kind "$os" "$arch")")
	if [ ! -x "$bin" ] || ! verify_sha256 "$bin" "$checksum" ||
		[ "$($bin version 2>/dev/null | awk '{print $2}')" != "$want" ]
	then
		temporary="$bin.download"
		echo "Downloading Kind $want for $os/$arch" >&2
		download "https://kind.sigs.k8s.io/dl/$want/kind-$os-$arch" "$temporary"
		verify_sha256 "$temporary" "$checksum"
		mv "$temporary" "$bin"
		chmod +x "$bin"
	fi
	if [ "$($bin version 2>/dev/null | awk '{print $2}')" != "$want" ]; then
		echo "downloaded Kind does not match versions.yaml: $bin" >&2
		exit 1
	fi
	echo "$bin"
}

istioctl_binary() {
	want=$(version_value istio)
	if command -v istioctl >/dev/null 2>&1 && [ "$(istioctl version --remote=false 2>/dev/null | awk '/client version:/ {print $3}')" = "$want" ]; then
		command -v istioctl
		return
	fi

	require_command curl
	require_command tar
	mkdir -p "$E2E_STATE_DIR/bin"
	bin="$E2E_STATE_DIR/bin/istioctl"
	os=$(release_os)
	arch=$(host_arch)
	archive="$E2E_STATE_DIR/istioctl-$want-$os-$arch.tar.gz"
	checksum=$(version_value "$(checksum_key istioctl "$os" "$arch")")
	if [ ! -f "$archive" ] || ! verify_sha256 "$archive" "$checksum"; then
		temporary="$archive.download"
		echo "Downloading istioctl $want for $os/$arch" >&2
		download "https://github.com/istio/istio/releases/download/$want/istioctl-$want-$os-$arch.tar.gz" "$temporary"
		verify_sha256 "$temporary" "$checksum"
		mv "$temporary" "$archive"
	fi
	# Re-extract every cached download before execution so a modified binary
	# cannot self-attest by printing the pinned version.
	tar -xzf "$archive" -C "$E2E_STATE_DIR/bin" istioctl
	if [ "$($bin version --remote=false 2>/dev/null | awk '/client version:/ {print $3}')" != "$want" ]; then
		echo "downloaded istioctl does not match versions.yaml: $bin" >&2
		exit 1
	fi
	echo "$bin"
}

admin_kubectl() {
	kubectl --context "$E2E_CONTEXT" "$@"
}
