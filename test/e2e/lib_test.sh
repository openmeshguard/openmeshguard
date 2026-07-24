#!/bin/sh

set -eu

TEST_SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/openmeshguard-e2e-lib.XXXXXX")
trap 'rm -rf "$TEST_ROOT"' EXIT HUP INT TERM

OPENMESHGUARD_E2E_STATE_DIR="$TEST_ROOT/state"
export OPENMESHGUARD_E2E_STATE_DIR
. "$TEST_SCRIPT_DIR/lib.sh"

# Use synthetic versions and locally generated artifacts to exercise the
# download/checksum paths without duplicating the real pins or using a network.
E2E_ROOT="$TEST_ROOT/repo"
mkdir -p "$E2E_ROOT" "$TEST_ROOT/fixtures/istio" "$TEST_ROOT/path"

printf '%s\n' '#!/bin/sh' 'echo "kind test-kind-version"' >"$TEST_ROOT/fixtures/kind"
printf '%s\n' '#!/bin/sh' 'echo "client version: test-istio-version"' >"$TEST_ROOT/fixtures/istio/istioctl"
printf '%s\n' 'apiVersion: apiextensions.k8s.io/v1' 'kind: CustomResourceDefinition' >"$TEST_ROOT/fixtures/gateway-api.yaml"
chmod +x "$TEST_ROOT/fixtures/kind" "$TEST_ROOT/fixtures/istio/istioctl"
tar -czf "$TEST_ROOT/fixtures/istioctl.tar.gz" -C "$TEST_ROOT/fixtures/istio" istioctl

test_sha256() {
	test_sha_path=$1
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$test_sha_path" | awk '{print $1}'
	else
		sha256sum "$test_sha_path" | awk '{print $1}'
	fi
}

test_kind_checksum=$(test_sha256 "$TEST_ROOT/fixtures/kind")
test_istio_checksum=$(test_sha256 "$TEST_ROOT/fixtures/istioctl.tar.gz")
test_gateway_api_checksum=$(test_sha256 "$TEST_ROOT/fixtures/gateway-api.yaml")
{
	printf 'kind: test-kind-version\n'
	printf 'istio: test-istio-version\n'
	printf 'gatewayAPI: test-gateway-api-version\n'
	printf 'kindSHA256LinuxAMD64: %s\n' "$test_kind_checksum"
	printf 'istioctlSHA256LinuxAMD64: %s\n' "$test_istio_checksum"
	printf 'gatewayAPIExperimentalSHA256: %s\n' "$test_gateway_api_checksum"
} >"$E2E_ROOT/versions.yaml"

# Shadow installed tools with the wrong version so the helpers must use their
# verified fallback binaries on developer machines as well as in CI.
printf '%s\n' '#!/bin/sh' 'echo "kind wrong-version"' >"$TEST_ROOT/path/kind"
printf '%s\n' '#!/bin/sh' 'echo "client version: wrong-version"' >"$TEST_ROOT/path/istioctl"
chmod +x "$TEST_ROOT/path/kind" "$TEST_ROOT/path/istioctl"
PATH="$TEST_ROOT/path:$PATH"
export PATH

host_os() {
	echo linux
}

release_os() {
	echo linux
}

host_arch() {
	echo amd64
}

require_command() {
	:
}

download() {
	test_download_url=$1
	test_download_output=$2
	case "$test_download_url" in
		https://kind.sigs.k8s.io/dl/test-kind-version/kind-linux-amd64)
			cp "$TEST_ROOT/fixtures/kind" "$test_download_output"
			;;
		https://github.com/istio/istio/releases/download/test-istio-version/istioctl-test-istio-version-linux-amd64.tar.gz)
			cp "$TEST_ROOT/fixtures/istioctl.tar.gz" "$test_download_output"
			;;
		https://github.com/kubernetes-sigs/gateway-api/releases/download/test-gateway-api-version/experimental-install.yaml)
			cp "$TEST_ROOT/fixtures/gateway-api.yaml" "$test_download_output"
			;;
		*)
			echo "unexpected download URL: $test_download_url" >&2
			return 1
			;;
	esac
}

test_kind_path=$(kind_binary)
test_istio_path=$(istioctl_binary)
test_gateway_api_path=$(gateway_api_crds_bundle)

if [ "$test_kind_path" != "$E2E_STATE_DIR/bin/kind" ]; then
	echo "kind fallback returned unexpected path: $test_kind_path" >&2
	exit 1
fi
if [ "$test_istio_path" != "$E2E_STATE_DIR/bin/istioctl" ]; then
	echo "istioctl fallback returned unexpected path: $test_istio_path" >&2
	exit 1
fi
if [ "$test_gateway_api_path" != "$E2E_STATE_DIR/downloads/gateway-api-test-gateway-api-version-experimental-install.yaml" ]; then
	echo "Gateway API fallback returned unexpected path: $test_gateway_api_path" >&2
	exit 1
fi

echo "E2E library fallback tests passed"
