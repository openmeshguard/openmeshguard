#!/bin/sh

set -eu

. "$(dirname -- "$0")/lib.sh"

KIND=$(kind_binary)
cleanup_admin_kubeconfig() {
	rm -f "$E2E_ADMIN_KUBECONFIG"
}
trap cleanup_admin_kubeconfig EXIT HUP INT TERM
started=$(date +%s)
"$KIND" delete cluster --name "$E2E_CLUSTER_NAME"
finished=$(date +%s)
echo "kind-down duration: $((finished - started))s"
