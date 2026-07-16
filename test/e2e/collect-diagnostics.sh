#!/bin/sh

set -u

. "$(dirname -- "$0")/lib.sh"

mkdir -p "$E2E_STATE_DIR/results/diagnostics"
diagnostics="$E2E_STATE_DIR/results/diagnostics"

if command -v kubectl >/dev/null 2>&1; then
	kubectl --context "$E2E_CONTEXT" get pods -A -o wide >"$diagnostics/pods.txt" 2>&1 || true
	kubectl --context "$E2E_CONTEXT" get events -A >"$diagnostics/events.txt" 2>&1 || true
fi

if command -v docker >/dev/null 2>&1; then
	docker logs "$E2E_CLUSTER_NAME-control-plane" >"$diagnostics/control-plane.log" 2>&1 || true
	docker exec "$E2E_CLUSTER_NAME-control-plane" cat /var/log/kubernetes/audit.log >"$diagnostics/audit.jsonl" 2>&1 || true
fi

KIND=$(kind_binary 2>/dev/null || true)
if [ -n "$KIND" ]; then
	"$KIND" export logs "$diagnostics/kind" --name "$E2E_CLUSTER_NAME" >"$diagnostics/kind-export.log" 2>&1 || true
fi
