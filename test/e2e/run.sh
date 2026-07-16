#!/bin/sh

set -eu

. "$(dirname -- "$0")/lib.sh"

cd "$E2E_ROOT"

require_command docker
require_command jq
require_command kubectl

KIND=$(kind_binary)
if ! "$KIND" get clusters 2>/dev/null | grep -Fx "$E2E_CLUSTER_NAME" >/dev/null; then
	echo "Kind cluster $E2E_CLUSTER_NAME does not exist; run make kind-up first" >&2
	exit 1
fi

started=$(date +%s)
results="$E2E_STATE_DIR/results"
kubeconfigs="$E2E_STATE_DIR/kubeconfigs"
goldens="$E2E_ROOT/test/fixtures/sidecar-basic/golden"
mkdir -p "$results" "$kubeconfigs"

make_sa_kubeconfig() {
	service_account=$1
	output=$2
	manager_kubeconfig=${3:-}
	if [ -n "$manager_kubeconfig" ]; then
		token=$(kubectl --kubeconfig "$manager_kubeconfig" -n "$E2E_HARNESS_NAMESPACE" create token "$service_account" --duration=30m)
	else
		token=$(admin_kubectl -n "$E2E_HARNESS_NAMESPACE" create token "$service_account" --duration=30m)
	fi
	admin_kubectl config view --raw --minify --flatten -o json | jq \
		--arg token "$token" \
		--arg user "$service_account" '
		.clusters[0].name = "openmeshguard-e2e" |
		.contexts = [{
		  "name": "openmeshguard-e2e",
		  "context": {"cluster": "openmeshguard-e2e", "user": $user}
		}] |
		.users = [{"name": $user, "user": {"token": $token}}] |
		."current-context" = "openmeshguard-e2e"
	' >"$output"
}

fixture_kubectl() {
	kubectl --kubeconfig "$kubeconfigs/fixture-manager.yaml" "$@"
}

assert_scanner_bindings() {
	bindings=$(fixture_kubectl get clusterrolebindings,rolebindings -A -o json)
	echo "$bindings" | jq -e \
		--arg namespace "$E2E_HARNESS_NAMESPACE" \
		--arg name "$E2E_CLUSTER_SCANNER" '
		[
		  .items[] |
		  select(any(.subjects[]?; .kind == "ServiceAccount" and .namespace == $namespace and .name == $name))
		] |
		length == 1 and
		.[0].kind == "ClusterRoleBinding" and
		.[0].roleRef.kind == "ClusterRole" and
		.[0].roleRef.name == "openmeshguard-cluster-scan"
	' >/dev/null
	echo "$bindings" | jq -e \
		--arg namespace "$E2E_HARNESS_NAMESPACE" \
		--arg name "$E2E_NAMESPACE_SCANNER" '
		[
		  .items[] |
		  select(any(.subjects[]?; .kind == "ServiceAccount" and .namespace == $namespace and .name == $name))
		] |
		length == 1 and
		.[0].kind == "RoleBinding" and
		.[0].metadata.namespace == "omg-strict" and
		.[0].roleRef.kind == "Role" and
		.[0].roleRef.name == "openmeshguard-namespace-scan"
	' >/dev/null
}

normalize_report() {
	jq '
		.generatedAt = "2000-01-01T00:00:00Z" |
		.scan.clusterContext = "openmeshguard-e2e"
	' "$1" >"$2"
}

compare_golden() {
	name=$1
	actual="$results/$name.json"
	golden="$goldens/$name.json"
	if [ "${UPDATE_GOLDEN:-0}" = 1 ]; then
		cp "$actual" "$golden"
		echo "updated $golden"
		return
	fi
	if ! diff -u "$golden" "$actual"; then
		echo "golden mismatch for $name; actual report: $actual" >&2
		exit 1
	fi
}

scan_fixture() {
	name=$1
	namespace=$2
	kubeconfig=$3
	raw="$results/$name.raw.json"
	"$E2E_ROOT/bin/openmeshguard" scan \
		--kubeconfig "$kubeconfig" \
		--namespace "$namespace" >"$raw"
	normalize_report "$raw" "$results/$name.json"
	OPENMESHGUARD_SCHEMA_REPORT="$results/$name.json" \
		go test ./internal/output -run '^TestExternalScanOutputMatchesSchema$' -count=1 >/dev/null
	compare_golden "$name"
}

admin_kubectl apply -f "$E2E_ROOT/test/e2e/harness-bootstrap.yaml" >/dev/null
make_sa_kubeconfig "$E2E_FIXTURE_MANAGER" "$kubeconfigs/fixture-manager.yaml"

fixture_kubectl apply -f "$E2E_ROOT/test/fixtures/sidecar-basic/manifests.yaml" >/dev/null
fixture_kubectl apply -f "$E2E_ROOT/deploy/rbac/cluster-role.yaml" >/dev/null
fixture_kubectl -n omg-strict apply -f "$E2E_ROOT/deploy/rbac/namespace-role.yaml" >/dev/null
fixture_kubectl apply -f "$E2E_ROOT/test/e2e/scanner-bindings.yaml" >/dev/null

for namespace in omg-strict omg-permissive omg-port-override omg-dr-contradiction omg-not-in-mesh omg-unclassified; do
	fixture_kubectl -n "$namespace" wait --for=condition=Available deployment --all --timeout=300s >/dev/null
done

for namespace in omg-strict omg-permissive omg-port-override omg-dr-contradiction omg-unclassified; do
	fixture_kubectl -n "$namespace" get pods -o json | jq -e '
		.items | length == 1 and all(.[]; any(.spec.containers[]; .name == "istio-proxy"))
	' >/dev/null
done
fixture_kubectl -n omg-not-in-mesh get pods -o json | jq -e '
	.items | length == 1 and all(.[]; all(.spec.containers[]; .name != "istio-proxy"))
' >/dev/null

assert_scanner_bindings
make_sa_kubeconfig "$E2E_CLUSTER_SCANNER" "$kubeconfigs/scanner-cluster.yaml" "$kubeconfigs/fixture-manager.yaml"
make_sa_kubeconfig "$E2E_NAMESPACE_SCANNER" "$kubeconfigs/scanner-namespace.yaml" "$kubeconfigs/fixture-manager.yaml"

# Discard all setup activity. The audit policy records only scanner SAs, and
# truncating here makes the proof artifact visibly scoped to scanner runs.
docker exec "$E2E_CLUSTER_NAME-control-plane" sh -c ': > /var/log/kubernetes/audit.log'

scan_fixture strict omg-strict "$kubeconfigs/scanner-cluster.yaml"
scan_fixture permissive omg-permissive "$kubeconfigs/scanner-cluster.yaml"
scan_fixture port-level-override omg-port-override "$kubeconfigs/scanner-cluster.yaml"
scan_fixture dr-contradiction omg-dr-contradiction "$kubeconfigs/scanner-cluster.yaml"
scan_fixture not-in-mesh omg-not-in-mesh "$kubeconfigs/scanner-cluster.yaml"
scan_fixture unclassified omg-unclassified "$kubeconfigs/scanner-cluster.yaml"
scan_fixture namespace-role-degraded omg-strict "$kubeconfigs/scanner-namespace.yaml"

jq -e '
	.workloadPostures | length == 1 and
	.[0].workload == {"namespace": "omg-port-override", "name": "port-api", "kind": "Deployment"} and
	.[0].dataPlaneMode == "sidecar" and
	.[0].mtls.effective == "unknown" and
	.[0].mtls.chain == [] and
	.[0].mtls.unknownReason == "workload ports unavailable for port-level PeerAuthentication on omg-port-override/port-override" and
	.[0].authorization.effective == "unknown"
' "$results/port-level-override.json" >/dev/null

jq -e '
	.workloadPostures | length == 1 and
	.[0].mtls.effective == "strict" and
	(.[0].mtls | has("clientTLSContradiction") | not)
' "$results/dr-contradiction.json" >/dev/null

jq -e '
	any(.permissionSummary[]; .apiGroup == "" and .resource == "namespaces" and .granted == false) and
	any(.permissionSummary[]; .apiGroup == "security.istio.io" and .resource == "peerauthentications" and .granted == false) and
	all(.workloadPostures[]; .mtls.effective == "unknown") and
	all(.findings[]; .status == "unknown")
' "$results/namespace-role-degraded.json" >/dev/null

docker exec "$E2E_CLUSTER_NAME-control-plane" cat /var/log/kubernetes/audit.log >"$results/audit.jsonl"
jq -s -e \
	--arg cluster_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_CLUSTER_SCANNER" \
	--arg namespace_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_NAMESPACE_SCANNER" '
	map(select(.user.username == $cluster_user or .user.username == $namespace_user)) as $events |
	($events | length > 0) and
	all($events[]; .verb == "get" or .verb == "list") and
	any($events[]; .user.username == $cluster_user) and
	any($events[]; .user.username == $namespace_user) and
	any($events[];
	  .user.username == $namespace_user and
	  .verb == "list" and
	  .objectRef.apiGroup == "security.istio.io" and
	  .objectRef.resource == "peerauthentications" and
	  .objectRef.namespace == "istio-system" and
	  .responseStatus.code == 403
	)
' "$results/audit.jsonl" >/dev/null

events=$(jq -s 'length' "$results/audit.jsonl")
finished=$(date +%s)
echo "RBAC proofs passed; scanner audit contains $events get/list events and no other verbs"
echo "e2e duration: $((finished - started))s"
