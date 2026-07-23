#!/bin/sh

set -eu

TEST_SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/openmeshguard-audit-assertions.XXXXXX")
trap 'rm -rf "$TEST_ROOT"' EXIT HUP INT TERM

. "$TEST_SCRIPT_DIR/lib.sh"
. "$TEST_SCRIPT_DIR/audit-assertions.sh"

cluster_user="system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_CLUSTER_SCANNER"
namespace_user="system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_NAMESPACE_SCANNER"
waypoint_limited_user="system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_WAYPOINT_LIMITED_SCANNER"
probe_user="system:serviceaccount:$E2E_HARNESS_NAMESPACE:audit-probe"
fixture_manager_user="system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_FIXTURE_MANAGER"

write_valid_audit() {
	audit_test_output=$1
	jq -cn \
		--arg cluster_user "$cluster_user" \
		--arg namespace_user "$namespace_user" \
		--arg waypoint_limited_user "$waypoint_limited_user" \
		--arg probe_user "$probe_user" '
		{"user":{"username":$cluster_user,"groups":["system:serviceaccounts"]},"verb":"list","objectRef":{"apiGroup":"","resource":"pods","namespace":"omg-strict"},"responseStatus":{"code":200}},
		{"user":{"username":$namespace_user,"groups":["system:serviceaccounts"]},"verb":"list","objectRef":{"apiGroup":"security.istio.io","resource":"peerauthentications","namespace":"istio-system"},"responseStatus":{"code":403}},
		{"user":{"username":$waypoint_limited_user,"groups":["system:serviceaccounts"]},"verb":"list","objectRef":{"apiGroup":"security.istio.io","resource":"authorizationpolicies","namespace":"omg-ambient-unavailable"},"responseStatus":{"code":200}},
		{"user":{"username":$probe_user,"groups":["system:serviceaccounts"]},"verb":"create","objectRef":{"apiGroup":"","resource":"configmaps","namespace":"omg-strict"},"responseStatus":{"code":403}}
	' >"$audit_test_output"
}

expect_rejected() {
	audit_test_description=$1
	audit_test_file=$2
	if assert_scanner_audit "$audit_test_file" 2>/dev/null; then
		echo "audit assertion accepted $audit_test_description" >&2
		exit 1
	fi
}

write_valid_audit "$TEST_ROOT/valid.jsonl"
assert_scanner_audit "$TEST_ROOT/valid.jsonl"

cp "$TEST_ROOT/valid.jsonl" "$TEST_ROOT/scanner-get.jsonl"
jq -cn --arg user "$cluster_user" \
	'{"user":{"username":$user},"verb":"get","objectRef":{"apiGroup":"","resource":"pods","namespace":"omg-strict","name":"api-1"},"responseStatus":{"code":200}}' \
	>>"$TEST_ROOT/scanner-get.jsonl"
expect_rejected "a named-object scanner get" "$TEST_ROOT/scanner-get.jsonl"

cp "$TEST_ROOT/valid.jsonl" "$TEST_ROOT/scanner-write.jsonl"
jq -cn --arg user "$cluster_user" \
	'{"user":{"username":$user},"verb":"create","objectRef":{"apiGroup":"","resource":"configmaps","namespace":"omg-strict"},"responseStatus":{"code":201}}' \
	>>"$TEST_ROOT/scanner-write.jsonl"
expect_rejected "a scanner write" "$TEST_ROOT/scanner-write.jsonl"

cp "$TEST_ROOT/valid.jsonl" "$TEST_ROOT/fixture-manager-write.jsonl"
jq -cn --arg user "$fixture_manager_user" \
	'{"user":{"username":$user},"verb":"create","objectRef":{"apiGroup":"","resource":"configmaps","namespace":"omg-strict"},"responseStatus":{"code":201}}' \
	>>"$TEST_ROOT/fixture-manager-write.jsonl"
expect_rejected "a fixture-manager write" "$TEST_ROOT/fixture-manager-write.jsonl"

cp "$TEST_ROOT/valid.jsonl" "$TEST_ROOT/admin-write.jsonl"
jq -cn \
	'{"user":{"username":"kubernetes-admin","groups":["kubeadm:cluster-admins"]},"verb":"create","objectRef":{"apiGroup":"","resource":"configmaps","namespace":"omg-strict"},"responseStatus":{"code":201}}' \
	>>"$TEST_ROOT/admin-write.jsonl"
expect_rejected "a Kind administrator write" "$TEST_ROOT/admin-write.jsonl"

echo "E2E audit assertion tests passed"
