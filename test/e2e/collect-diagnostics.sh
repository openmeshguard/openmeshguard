#!/bin/sh

set -u
umask 077

. "$(dirname -- "$0")/lib.sh"

mkdir -p "$E2E_STATE_DIR/results/diagnostics"
diagnostics="$E2E_STATE_DIR/results/diagnostics"
diagnostics_kubeconfig="$diagnostics/admin.kubeconfig"
trap 'rm -f "$diagnostics_kubeconfig"' EXIT HUP INT TERM

if command -v kubectl >/dev/null 2>&1; then
	KIND=$(kind_binary 2>/dev/null || true)
	if [ -n "$KIND" ]; then
		"$KIND" export kubeconfig --name "$E2E_CLUSTER_NAME" --kubeconfig "$diagnostics_kubeconfig" >/dev/null 2>&1 || true
		chmod 600 "$diagnostics_kubeconfig" 2>/dev/null || true
		kubectl --kubeconfig "$diagnostics_kubeconfig" --context "$E2E_CONTEXT" get pods -A -o wide >"$diagnostics/pods.txt" 2>&1 || true
		kubectl --kubeconfig "$diagnostics_kubeconfig" --context "$E2E_CONTEXT" get events -A >"$diagnostics/events.txt" 2>&1 || true
	fi
fi

if command -v docker >/dev/null 2>&1; then
	docker logs "$E2E_CLUSTER_NAME-control-plane" >"$diagnostics/control-plane.log" 2>&1 || true
	docker exec "$E2E_CLUSTER_NAME-control-plane" cat /var/log/kubernetes/audit.log >"$diagnostics/audit.jsonl" 2>&1 || true
fi

KIND=$(kind_binary 2>/dev/null || true)
if [ -n "$KIND" ]; then
	"$KIND" export logs "$diagnostics/kind" --name "$E2E_CLUSTER_NAME" >"$diagnostics/kind-export.log" 2>&1 || true
fi
