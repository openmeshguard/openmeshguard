#!/bin/sh

set -eu

TEST_SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
run_script="$TEST_SCRIPT_DIR/run.sh"
audit_policy="$TEST_SCRIPT_DIR/audit-policy.yaml"
kind_up_script="$TEST_SCRIPT_DIR/kind-up.sh"
lib_script="$TEST_SCRIPT_DIR/lib.sh"

if grep -F -- '--arg token' "$run_script" >/dev/null; then
	echo "E2E harness exposes a bearer token in process arguments" >&2
	exit 1
fi
if ! grep -F -- '--rawfile token "$kubeconfig_token"' "$run_script" >/dev/null; then
	echo "E2E harness does not read the bearer token from its protected file" >&2
	exit 1
fi
if ! grep -F 'env -i HOME="$scanner_home" "$scanner_binary"' "$run_script" >/dev/null; then
	echo "E2E scanner child does not run with an isolated environment" >&2
	exit 1
fi
if ! grep -F -- '--kubeconfig "$E2E_ADMIN_KUBECONFIG"' "$kind_up_script" >/dev/null; then
	echo "Kind writes its administrator credential to the inherited/default kubeconfig" >&2
	exit 1
fi
if ! grep -F 'kubectl --kubeconfig "$E2E_ADMIN_KUBECONFIG"' "$lib_script" >/dev/null; then
	echo "harness administrator calls do not use the dedicated kubeconfig" >&2
	exit 1
fi
if ! grep -F 'rm -f "$E2E_ADMIN_KUBECONFIG"' "$run_script" >/dev/null; then
	echo "E2E harness does not remove the Kind administrator kubeconfig before scanning" >&2
	exit 1
fi
if ! grep -F 'export kubeconfig --name "$E2E_CLUSTER_NAME" --kubeconfig "$E2E_ADMIN_KUBECONFIG"' "$run_script" >/dev/null; then
	echo "repeated E2E runs do not recreate the protected administrator kubeconfig" >&2
	exit 1
fi
if ! grep -F 'system:serviceaccount:openmeshguard-e2e:fixture-manager' "$audit_policy" >/dev/null; then
	echo "audit policy does not record fixture-manager requests" >&2
	exit 1
fi
if ! grep -F '      - kubernetes-admin' "$audit_policy" >/dev/null; then
	echo "audit policy does not record Kind administrator requests" >&2
	exit 1
fi

echo "E2E harness credential-isolation tests passed"
