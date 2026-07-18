#!/bin/sh

set -eu
umask 077

. "$(dirname -- "$0")/lib.sh"

require_command docker
require_command jq
require_command kubectl

KIND=$(kind_binary)
ISTIOCTL=$(istioctl_binary)
KIND_NODE_IMAGE=$(version_value kindNodeImage)
ISTIO_VERSION=$(version_value istio)

if "$KIND" get clusters 2>/dev/null | grep -Fx "$E2E_CLUSTER_NAME" >/dev/null; then
	echo "Kind cluster $E2E_CLUSTER_NAME already exists; run make kind-down first" >&2
	exit 1
fi

mkdir -p "$E2E_STATE_DIR/audit" "$E2E_STATE_DIR/results"
chmod 700 "$E2E_STATE_DIR"
rm -f "$E2E_ADMIN_KUBECONFIG"
kind_up_complete=0
cleanup_failed_kind_up() {
	if [ "$kind_up_complete" -ne 1 ]; then
		rm -f "$E2E_ADMIN_KUBECONFIG"
	fi
}
trap cleanup_failed_kind_up EXIT HUP INT TERM
audit_policy="$E2E_ROOT/test/e2e/audit-policy.yaml"
kind_config="$E2E_STATE_DIR/kind-config.yaml"
write_kind_config "$kind_config" "$audit_policy" "$E2E_STATE_DIR/audit"

started=$(date +%s)
"$KIND" create cluster \
	--name "$E2E_CLUSTER_NAME" \
	--image "$KIND_NODE_IMAGE" \
	--config "$kind_config" \
	--kubeconfig "$E2E_ADMIN_KUBECONFIG" \
	--wait 180s
chmod 600 "$E2E_ADMIN_KUBECONFIG"

# ServiceAccounts are automatically members of system:authenticated. The
# default system:basic-user binding grants create on self-review resources, so
# remove it from this disposable cluster before proving exclusive scanner RBAC.
admin_kubectl delete clusterrolebinding system:basic-user --ignore-not-found >/dev/null

"$ISTIOCTL" install \
	--kubeconfig "$E2E_ADMIN_KUBECONFIG" \
	--context "$E2E_CONTEXT" \
	--set profile=default \
	--skip-confirmation

admin_kubectl -n istio-system rollout status deployment/istiod --timeout=300s

finished=$(date +%s)
echo "Kind $($(kind_binary) version | awk '{print $2}'), node $KIND_NODE_IMAGE, Istio $ISTIO_VERSION"
echo "kind-up duration: $((finished - started))s"
kind_up_complete=1
