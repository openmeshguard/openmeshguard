#!/bin/sh

set -eu

E2E_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
E2E_STATE_DIR=${OPENMESHGUARD_E2E_STATE_DIR:-"$E2E_ROOT/.e2e"}
E2E_CLUSTER_NAME=${OPENMESHGUARD_E2E_CLUSTER_NAME:-openmeshguard-e2e}
E2E_CONTEXT="kind-$E2E_CLUSTER_NAME"
E2E_HARNESS_NAMESPACE=openmeshguard-e2e
E2E_FIXTURE_MANAGER=fixture-manager
E2E_CLUSTER_SCANNER=scanner-cluster
E2E_NAMESPACE_SCANNER=scanner-namespace

version_value() {
	key=$1
	awk -v key="$key" '
		$0 ~ "^[[:space:]]*" key ":[[:space:]]*" {
			sub("^[[:space:]]*" key ":[[:space:]]*", "")
			gsub(/^['\"']|['\"']$/, "")
			print
			exit
		}
	' "$E2E_ROOT/versions.yaml"
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

kind_binary() {
	want=$(version_value kind)
	if command -v kind >/dev/null 2>&1 && [ "$(kind version 2>/dev/null | awk '{print $2}')" = "$want" ]; then
		command -v kind
		return
	fi

	require_command curl
	mkdir -p "$E2E_STATE_DIR/bin"
	bin="$E2E_STATE_DIR/bin/kind"
	if [ ! -x "$bin" ] || [ "$($bin version 2>/dev/null | awk '{print $2}')" != "$want" ]; then
		os=$(host_os)
		arch=$(host_arch)
		echo "Downloading Kind $want for $os/$arch" >&2
		curl -fsSL "https://kind.sigs.k8s.io/dl/$want/kind-$os-$arch" -o "$bin"
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
	if [ ! -x "$bin" ] || [ "$($bin version --remote=false 2>/dev/null | awk '/client version:/ {print $3}')" != "$want" ]; then
		os=$(release_os)
		arch=$(host_arch)
		archive="$E2E_STATE_DIR/istioctl-$want-$os-$arch.tar.gz"
		echo "Downloading istioctl $want for $os/$arch" >&2
		curl -fsSL "https://github.com/istio/istio/releases/download/$want/istioctl-$want-$os-$arch.tar.gz" -o "$archive"
		tar -xzf "$archive" -C "$E2E_STATE_DIR/bin" istioctl
	fi
	if [ "$($bin version --remote=false 2>/dev/null | awk '/client version:/ {print $3}')" != "$want" ]; then
		echo "downloaded istioctl does not match versions.yaml: $bin" >&2
		exit 1
	fi
	echo "$bin"
}

admin_kubectl() {
	kubectl --context "$E2E_CONTEXT" "$@"
}
