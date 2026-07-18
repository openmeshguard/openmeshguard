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
E2E_ADMIN_KUBECONFIG="$E2E_STATE_DIR/admin.kubeconfig"

version_value() {
	version_key=$1
	version_result=$(awk -v key="$version_key" '
		$0 ~ "^[[:space:]]*" key ":[[:space:]]*" {
			sub("^[[:space:]]*" key ":[[:space:]]*", "")
			gsub(/^['\"']|['\"']$/, "")
			print
			exit
		}
	' "$E2E_ROOT/versions.yaml")
	if [ -z "$version_result" ]; then
		echo "versions.yaml missing required key: $version_key" >&2
		return 1
	fi
	printf '%s\n' "$version_result"
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
	checksum_prefix=$1
	checksum_os=$2
	checksum_arch=$3
	case "$checksum_prefix:$checksum_os:$checksum_arch" in
		kind:darwin:amd64) echo kindSHA256DarwinAMD64 ;;
		kind:darwin:arm64) echo kindSHA256DarwinARM64 ;;
		kind:linux:amd64) echo kindSHA256LinuxAMD64 ;;
		kind:linux:arm64) echo kindSHA256LinuxARM64 ;;
		istioctl:osx:amd64) echo istioctlSHA256OSXAMD64 ;;
		istioctl:osx:arm64) echo istioctlSHA256OSXARM64 ;;
		istioctl:linux:amd64) echo istioctlSHA256LinuxAMD64 ;;
		istioctl:linux:arm64) echo istioctlSHA256LinuxARM64 ;;
		*) echo "unsupported checksum platform: $checksum_prefix $checksum_os/$checksum_arch" >&2; return 1 ;;
	esac
}

verify_sha256() {
	sha_path=$1
	sha_expected=$2
	if command -v shasum >/dev/null 2>&1; then
		sha_actual=$(shasum -a 256 "$sha_path" | awk '{print $1}')
	elif command -v sha256sum >/dev/null 2>&1; then
		sha_actual=$(sha256sum "$sha_path" | awk '{print $1}')
	else
		echo "required command not found: shasum or sha256sum" >&2
		return 1
	fi
	if [ "$sha_actual" != "$sha_expected" ]; then
		echo "SHA-256 mismatch for $sha_path: got $sha_actual, want $sha_expected" >&2
		return 1
	fi
}

download() {
	download_url=$1
	download_output=$2
	curl -fsSL --retry 3 --retry-all-errors --connect-timeout 15 --max-time 180 "$download_url" -o "$download_output"
}

yaml_quote() {
	yaml_value=$1
	jq -Rn --arg value "$yaml_value" '$value'
}

write_kind_config() {
	kind_config_output=$1
	kind_config_audit_policy=$(yaml_quote "$2")
	kind_config_audit_dir=$(yaml_quote "$3")
	cat >"$kind_config_output" <<EOF
apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: ClusterConfiguration
        apiServer:
          extraArgs:
            audit-log-path: /var/log/kubernetes/audit.log
            audit-log-mode: blocking-strict
            audit-policy-file: /etc/kubernetes/policies/audit-policy.yaml
          extraVolumes:
            - name: audit-policies
              hostPath: /etc/kubernetes/policies
              mountPath: /etc/kubernetes/policies
              readOnly: true
              pathType: DirectoryOrCreate
            - name: audit-logs
              hostPath: /var/log/kubernetes
              mountPath: /var/log/kubernetes
              readOnly: false
              pathType: DirectoryOrCreate
    extraMounts:
      - hostPath: $kind_config_audit_policy
        containerPath: /etc/kubernetes/policies/audit-policy.yaml
        readOnly: true
      - hostPath: $kind_config_audit_dir
        containerPath: /var/log/kubernetes
EOF
}

kind_binary() {
	kind_version=$(version_value kind)
	if command -v kind >/dev/null 2>&1 && [ "$(kind version 2>/dev/null | awk '{print $2}')" = "$kind_version" ]; then
		command -v kind
		return
	fi

	require_command curl
	mkdir -p "$E2E_STATE_DIR/bin"
	kind_path="$E2E_STATE_DIR/bin/kind"
	kind_os=$(host_os)
	kind_arch=$(host_arch)
	kind_checksum=$(version_value "$(checksum_key kind "$kind_os" "$kind_arch")")
	if [ ! -x "$kind_path" ] || ! verify_sha256 "$kind_path" "$kind_checksum" ||
		[ "$("$kind_path" version 2>/dev/null | awk '{print $2}')" != "$kind_version" ]
	then
		kind_temporary="$kind_path.download"
		echo "Downloading Kind $kind_version for $kind_os/$kind_arch" >&2
		download "https://kind.sigs.k8s.io/dl/$kind_version/kind-$kind_os-$kind_arch" "$kind_temporary"
		verify_sha256 "$kind_temporary" "$kind_checksum"
		mv "$kind_temporary" "$kind_path"
		chmod +x "$kind_path"
	fi
	if [ "$("$kind_path" version 2>/dev/null | awk '{print $2}')" != "$kind_version" ]; then
		echo "downloaded Kind does not match versions.yaml: $kind_path" >&2
		exit 1
	fi
	echo "$kind_path"
}

istioctl_binary() {
	istio_version=$(version_value istio)
	if command -v istioctl >/dev/null 2>&1 && [ "$(istioctl version --remote=false 2>/dev/null | awk '/client version:/ {print $3}')" = "$istio_version" ]; then
		command -v istioctl
		return
	fi

	require_command curl
	require_command tar
	mkdir -p "$E2E_STATE_DIR/bin"
	istio_path="$E2E_STATE_DIR/bin/istioctl"
	istio_os=$(release_os)
	istio_arch=$(host_arch)
	istio_archive="$E2E_STATE_DIR/istioctl-$istio_version-$istio_os-$istio_arch.tar.gz"
	istio_checksum=$(version_value "$(checksum_key istioctl "$istio_os" "$istio_arch")")
	if [ ! -f "$istio_archive" ] || ! verify_sha256 "$istio_archive" "$istio_checksum"; then
		istio_temporary="$istio_archive.download"
		echo "Downloading istioctl $istio_version for $istio_os/$istio_arch" >&2
		download "https://github.com/istio/istio/releases/download/$istio_version/istioctl-$istio_version-$istio_os-$istio_arch.tar.gz" "$istio_temporary"
		verify_sha256 "$istio_temporary" "$istio_checksum"
		mv "$istio_temporary" "$istio_archive"
	fi
	# Re-extract every cached download before execution so a modified binary
	# cannot self-attest by printing the pinned version.
	tar -xzf "$istio_archive" -C "$E2E_STATE_DIR/bin" istioctl
	if [ "$("$istio_path" version --remote=false 2>/dev/null | awk '/client version:/ {print $3}')" != "$istio_version" ]; then
		echo "downloaded istioctl does not match versions.yaml: $istio_path" >&2
		exit 1
	fi
	echo "$istio_path"
}

admin_kubectl() {
	kubectl --kubeconfig "$E2E_ADMIN_KUBECONFIG" --context "$E2E_CONTEXT" "$@"
}
