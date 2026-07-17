#!/bin/sh

assert_scanner_audit() {
	audit_assertions_file=$1
	jq -s -e \
		--arg cluster_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_CLUSTER_SCANNER" \
		--arg namespace_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_NAMESPACE_SCANNER" \
		--arg probe_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:audit-probe" \
		--arg fixture_manager_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_FIXTURE_MANAGER" \
		--arg kind_admin_user "kubernetes-admin" \
		-f "$E2E_ROOT/test/e2e/audit-assertions.jq" \
		"$audit_assertions_file" >/dev/null
}
