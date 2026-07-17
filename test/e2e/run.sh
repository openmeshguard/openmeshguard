#!/bin/sh

set -eu
umask 077

. "$(dirname -- "$0")/lib.sh"
. "$(dirname -- "$0")/report-assertions.sh"

cd "$E2E_ROOT"

require_command docker
require_command jq
require_command kubectl

KIND=$(kind_binary)
if ! "$KIND" get clusters 2>/dev/null | grep -Fx "$E2E_CLUSTER_NAME" >/dev/null; then
	echo "Kind cluster $E2E_CLUSTER_NAME does not exist; run make kind-up first" >&2
	exit 1
fi
if [ ! -x "$E2E_ROOT/bin/openmeshguard" ]; then
	echo "scanner binary not found: $E2E_ROOT/bin/openmeshguard; run make build" >&2
	exit 1
fi

started=$(date +%s)
results="$E2E_STATE_DIR/results"
goldens="$E2E_ROOT/test/fixtures/sidecar-basic/golden"
cases="$E2E_ROOT/test/fixtures/sidecar-basic/cases.tsv"
mkdir -p "$results"
find "$results" -mindepth 1 -delete
kubeconfigs=$(mktemp -d "$E2E_STATE_DIR/kubeconfigs.XXXXXX")
chmod 700 "$kubeconfigs"
tab=$(printf '\t')

cleanup_credentials() {
	find "$kubeconfigs" -type f -delete 2>/dev/null || true
	rmdir "$kubeconfigs" 2>/dev/null || true
}
trap cleanup_credentials EXIT
trap 'exit 130' HUP INT TERM

assert_json() {
	description=$1
	file=$2
	filter=$3
	if ! jq -e "$filter" "$file" >/dev/null; then
		echo "semantic assertion failed: $description ($file)" >&2
		return 1
	fi
}

make_sa_kubeconfig() {
	service_account=$1
	output=$2
	manager_kubeconfig=$3
	if [ -n "$manager_kubeconfig" ]; then
		token=$(kubectl --kubeconfig "$manager_kubeconfig" -n "$E2E_HARNESS_NAMESPACE" create token "$service_account" --duration=10m)
	else
		token=$(admin_kubectl -n "$E2E_HARNESS_NAMESPACE" create token "$service_account" --duration=10m)
	fi
	temporary="$output.tmp"
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
	' >"$temporary"
	chmod 600 "$temporary"
	mv "$temporary" "$output"
}

fixture_kubectl() {
	kubectl --kubeconfig "$kubeconfigs/fixture-manager.yaml" "$@"
}

assert_live_profile() {
	name=$1
	manifest=$2
	namespace=$3
	resource=$4
	expected="$kubeconfigs/$name-expected.json"
	live="$results/$name-live.json"
	if [ -n "$namespace" ]; then
		fixture_kubectl -n "$namespace" create --dry-run=client --validate=false -f "$manifest" -o json >"$expected"
		fixture_kubectl -n "$namespace" get "$resource" -o json >"$live"
	else
		fixture_kubectl create --dry-run=client --validate=false -f "$manifest" -o json >"$expected"
		fixture_kubectl get "$resource" -o json >"$live"
	fi
	if ! jq -e --slurpfile expected "$expected" '
		def proof_shape: {
		  apiVersion,
		  kind,
		  name: .metadata.name,
		  aggregationRule: (.aggregationRule // null),
		  rules
		};
		. as $live |
		($expected[0] | proof_shape) == ($live | proof_shape)
	' "$live" >/dev/null; then
		echo "live RBAC profile differs from published manifest: $live" >&2
		return 1
	fi
}

assert_scanner_binding() {
	bindings=$1
	name=$2
	expected_kind=$3
	expected_namespace=$4
	expected_role_kind=$5
	expected_role=$6
	if ! printf '%s\n' "$bindings" | jq -e \
		--arg harness_namespace "$E2E_HARNESS_NAMESPACE" \
		--arg name "$name" \
		--arg expected_kind "$expected_kind" \
		--arg expected_namespace "$expected_namespace" \
		--arg expected_role_kind "$expected_role_kind" \
		--arg expected_role "$expected_role" '
		def affects_scanner:
		  any(.subjects[]?;
		    (.kind == "ServiceAccount" and .namespace == $harness_namespace and .name == $name) or
		    (.kind == "User" and .name == ("system:serviceaccount:" + $harness_namespace + ":" + $name)) or
		    (.kind == "Group" and (
		      .name == "system:serviceaccounts" or
		      .name == ("system:serviceaccounts:" + $harness_namespace) or
		      .name == "system:authenticated"
		    ))
		  );
		def verified_nonresource_default:
		  .kind == "ClusterRoleBinding" and
		  .roleRef.kind == "ClusterRole" and
		  (
		    .roleRef.name == "system:discovery" or
		    .roleRef.name == "system:public-info-viewer" or
		    .roleRef.name == "system:service-account-issuer-discovery"
		  );
		[
		  .items[] |
		  select(affects_scanner) |
		  select(verified_nonresource_default | not)
		] |
		length == 1 and
		.[0].kind == $expected_kind and
		(.[0].metadata.namespace // "") == $expected_namespace and
		.[0].roleRef.kind == $expected_role_kind and
		.[0].roleRef.name == $expected_role
	' >/dev/null; then
		echo "scanner $name has a resource-authorizing binding beyond its published profile; inspect $results/scanner-bindings.json" >&2
		return 1
	fi
}

assert_nonresource_default_role() {
	default_role_name=$1
	default_role_file="$results/$default_role_name.json"
	fixture_kubectl get clusterrole "$default_role_name" -o json >"$default_role_file"
	if ! jq -e '
		(.aggregationRule == null) and
		(.rules | length) > 0 and
		all(.rules[];
		  (.verbs == ["get"]) and
		  ((.nonResourceURLs // []) | length) > 0 and
		  ((.apiGroups // []) | length) == 0 and
		  ((.resources // []) | length) == 0 and
		  ((.resourceNames // []) | length) == 0
		)
	' "$default_role_file" >/dev/null; then
		echo "Kubernetes default role $default_role_name is not limited to non-resource get access; inspect $default_role_file" >&2
		return 1
	fi
}

assert_default_rbac_isolation() {
	if fixture_kubectl get clusterrolebinding system:basic-user >/dev/null 2>&1; then
		echo "system:basic-user must be unbound in the disposable proof cluster" >&2
		return 1
	fi
	assert_nonresource_default_role system:discovery
	assert_nonresource_default_role system:public-info-viewer
	assert_nonresource_default_role system:service-account-issuer-discovery
}

assert_scanner_bindings() {
	bindings=$(fixture_kubectl get clusterrolebindings,rolebindings -A -o json)
	printf '%s\n' "$bindings" >"$results/scanner-bindings.json"
	assert_scanner_binding "$bindings" "$E2E_CLUSTER_SCANNER" ClusterRoleBinding "" ClusterRole openmeshguard-cluster-scan
	assert_scanner_binding "$bindings" "$E2E_NAMESPACE_SCANNER" RoleBinding omg-strict Role openmeshguard-namespace-scan
}

assert_schema_test_available() {
	tests=$(go test ./internal/output -list '^TestExternalScanOutputMatchesSchema$')
	if ! printf '%s\n' "$tests" | grep -Fx 'TestExternalScanOutputMatchesSchema' >/dev/null; then
		echo "schema test TestExternalScanOutputMatchesSchema was not discovered" >&2
		exit 1
	fi
}

validate_schema() {
	report=$1
	if ! OPENMESHGUARD_SCHEMA_REPORT="$report" \
		go test ./internal/output -run '^TestExternalScanOutputMatchesSchema$' -count=1 >/dev/null
	then
		echo "schema validation failed: $report" >&2
		return 1
	fi
}

normalize_report() {
	raw=$1
	normalized=$2
	assert_json "raw report emits generatedAt and scan.clusterContext" "$raw" '
		.generatedAt as $generated |
		.scan.clusterContext as $context |
		($generated | type) == "string" and
		($generated | length) > 0 and
		($context | type) == "string" and
		($context | length) > 0
	'
	jq '
		.generatedAt = "2000-01-01T00:00:00Z" |
		.scan.clusterContext = "openmeshguard-e2e"
	' "$raw" >"$normalized"
}

compare_golden() {
	name=$1
	actual="$results/$name.json"
	golden="$goldens/$name.json"
	update_golden=$(printenv UPDATE_GOLDEN 2>/dev/null || true)
	if [ "$update_golden" = 1 ]; then
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
	validate_schema "$results/$name.json"
}

scan_cluster() {
	raw="$results/cluster-scan.raw.json"
	"$E2E_ROOT/bin/openmeshguard" scan \
		--kubeconfig "$kubeconfigs/scanner-cluster.yaml" \
		--all-namespaces >"$raw"
	normalize_report "$raw" "$results/cluster-scan.json"
	validate_schema "$results/cluster-scan.json"
}

capture_audit() {
	docker exec "$E2E_CLUSTER_NAME-control-plane" cat /var/log/kubernetes/audit.log >"$results/audit.jsonl"
}

assert_schema_test_available

echo "e2e: bootstrap distinct fixture-manager, scanner, and audit-probe identities"
admin_kubectl apply -f "$E2E_ROOT/test/e2e/harness-bootstrap.yaml" >/dev/null
make_sa_kubeconfig "$E2E_FIXTURE_MANAGER" "$kubeconfigs/fixture-manager.yaml" ""

echo "e2e: reset and apply sidecar fixtures and published RBAC profiles"
while IFS="$tab" read -r name namespace deployment proxy expected_findings; do
	fixture_kubectl delete namespace "$namespace" --ignore-not-found --wait=false >/dev/null
done <"$cases"
attempt=0
while :; do
	remaining=0
	while IFS="$tab" read -r name namespace deployment proxy expected_findings; do
		if fixture_kubectl get namespace "$namespace" >/dev/null 2>&1; then
			remaining=1
		fi
	done <"$cases"
	if [ "$remaining" -eq 0 ]; then
		break
	fi
	attempt=$((attempt + 1))
	if [ "$attempt" -ge 120 ]; then
		echo "fixture namespaces did not terminate within 120s" >&2
		exit 1
	fi
	sleep 1
done
fixture_kubectl apply -f "$E2E_ROOT/test/fixtures/sidecar-basic/manifests.yaml" >/dev/null
fixture_kubectl apply -f "$E2E_ROOT/deploy/rbac/cluster-role.yaml" >/dev/null
fixture_kubectl -n omg-strict apply -f "$E2E_ROOT/deploy/rbac/namespace-role.yaml" >/dev/null
fixture_kubectl apply -f "$E2E_ROOT/test/e2e/scanner-bindings.yaml" >/dev/null

echo "e2e: wait for fixture workloads and verify sidecar enrollment"
while IFS="$tab" read -r name namespace deployment proxy expected_findings; do
	fixture_kubectl -n "$namespace" rollout status "deployment/$deployment" --timeout=300s >/dev/null
	pods="$kubeconfigs/$name-pods.json"
	fixture_kubectl -n "$namespace" get pods -o json >"$pods"
	if [ "$proxy" = sidecar ]; then
		assert_json "$namespace has one pod with istio-proxy" "$pods" '
			.items | length == 1 and
			all(.[]; ([.spec.containers[], .spec.initContainers[]?] | any(.name == "istio-proxy")))
		'
	else
		assert_json "$namespace has one pod without istio-proxy" "$pods" '
			.items | length == 1 and
			all(.[]; ([.spec.containers[], .spec.initContainers[]?] | all(.name != "istio-proxy")))
		'
	fi
done <"$cases"

fixture_kubectl -n omg-port-override get peerauthentication port-override -o json >"$kubeconfigs/port-policy.json"
assert_json "live port override input is DISABLE on 8080" "$kubeconfigs/port-policy.json" '
	.spec.portLevelMtls["8080"].mode == "DISABLE"
'
fixture_kubectl -n omg-dr-contradiction get destinationrule dr-api -o json >"$kubeconfigs/destination-rule.json"
assert_json "live DestinationRule input disables client TLS" "$kubeconfigs/destination-rule.json" '
	.spec.trafficPolicy.tls.mode == "DISABLE"
'
fixture_kubectl -n omg-workload-conflict get peerauthentication conflict-api -o json >"$kubeconfigs/workload-policy.json"
assert_json "live workload policy overrides namespace STRICT with DISABLE" "$kubeconfigs/workload-policy.json" '
	.spec.selector.matchLabels == {"app": "conflict-api"} and .spec.mtls.mode == "DISABLE"
'

echo "e2e: prove each scanner has only its published resource-authorizing RBAC binding"
assert_default_rbac_isolation
assert_live_profile cluster-role "$E2E_ROOT/deploy/rbac/cluster-role.yaml" "" clusterrole/openmeshguard-cluster-scan
assert_live_profile namespace-role "$E2E_ROOT/deploy/rbac/namespace-role.yaml" omg-strict role/openmeshguard-namespace-scan
assert_scanner_bindings
make_sa_kubeconfig "$E2E_CLUSTER_SCANNER" "$kubeconfigs/scanner-cluster.yaml" "$kubeconfigs/fixture-manager.yaml"
make_sa_kubeconfig "$E2E_NAMESPACE_SCANNER" "$kubeconfigs/scanner-namespace.yaml" "$kubeconfigs/fixture-manager.yaml"
make_sa_kubeconfig audit-probe "$kubeconfigs/audit-probe.yaml" "$kubeconfigs/fixture-manager.yaml"

attempt=0
while ! kubectl --kubeconfig "$kubeconfigs/scanner-cluster.yaml" --request-timeout=5s get namespace omg-strict >/dev/null 2>&1 ||
	! kubectl --kubeconfig "$kubeconfigs/scanner-namespace.yaml" --request-timeout=5s -n omg-strict get pods >/dev/null 2>&1
do
	attempt=$((attempt + 1))
	if [ "$attempt" -ge 20 ]; then
		echo "scanner RBAC bindings did not become observable within 20s" >&2
		exit 1
	fi
	sleep 1
done

# Discard setup and RBAC-settle activity. The audit policy retains only the
# scanners plus the deliberately unprivileged positive-control identity.
docker exec "$E2E_CLUSTER_NAME-control-plane" sh -c ': > /var/log/kubernetes/audit.log'

echo "e2e: prove the API-server audit path records a denied write positive control"
if kubectl --kubeconfig "$kubeconfigs/audit-probe.yaml" -n omg-strict \
	create configmap openmeshguard-audit-positive-control --from-literal=proof=denied >"$kubeconfigs/audit-probe.out" 2>&1
then
	echo "audit positive control unexpectedly created a ConfigMap" >&2
	exit 1
fi
attempt=0
while :; do
	capture_audit
	if jq -s -e \
		--arg user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:audit-probe" '
		any(.[];
		  .user.username == $user and
		  .verb == "create" and
		  (.objectRef.apiGroup // "") == "" and
		  .objectRef.resource == "configmaps" and
		  .responseStatus.code == 403
		)
	' "$results/audit.jsonl" >/dev/null; then
		break
	fi
	attempt=$((attempt + 1))
	if [ "$attempt" -ge 30 ]; then
		echo "audit positive control was not recorded within 30s" >&2
		exit 1
	fi
	sleep 1
done

echo "e2e: scan namespace fixtures and compare canonical JSON goldens"
while IFS="$tab" read -r name namespace deployment proxy expected_findings; do
	scan_fixture "$name" "$namespace" "$kubeconfigs/scanner-cluster.yaml"
done <"$cases"
scan_fixture namespace-role-degraded omg-strict "$kubeconfigs/scanner-namespace.yaml"

echo "e2e: exercise the published ClusterRole with an all-namespaces scan"
scan_cluster
assert_json "cluster scan workload targets are globally ordered" "$results/cluster-scan.json" '
	[.workloadPostures[].workload | "\(.namespace)/\(.kind)/\(.name)"] as $targets |
	$targets == ($targets | sort)
'
while IFS="$tab" read -r name namespace deployment proxy expected_findings; do
	if ! jq -e --arg namespace "$namespace" --arg deployment "$deployment" '
		any(.workloadPostures[];
		  .workload.namespace == $namespace and
		  .workload.name == $deployment and
		  .workload.kind == "Deployment"
		)
	' "$results/cluster-scan.json" >/dev/null; then
		echo "cluster scan missing fixture workload $namespace/Deployment/$deployment" >&2
		exit 1
	fi
done <"$cases"

while IFS="$tab" read -r name namespace deployment proxy expected_findings; do
	assert_json "$name emits one workload posture and at least one finding" "$results/$name.json" '
		(.workloadPostures | length) == 1 and
		(.findings | length) > 0
	'
	assert_report_update_guard "$name" "$results/$name.json" "$expected_findings"
done <"$cases"
assert_json "namespace Role scan emits one workload posture and all three built-in findings" "$results/namespace-role-degraded.json" '
	(.workloadPostures | length) == 1 and
	(.findings | length) == 3
'
assert_report_update_guard namespace-role-degraded "$results/namespace-role-degraded.json" \
	"MG-MTLS-001=unknown,MG-MTLS-002=unknown,MG-MTLS-003=unknown"

assert_json "strict namespace resolves strict" "$results/strict.json" '
	.workloadPostures | length == 1 and .[0].mtls.effective == "strict"
'
assert_json "permissive namespace overrides mesh strict" "$results/permissive.json" '
	.workloadPostures | length == 1 and .[0].mtls.effective == "permissive"
'
assert_json "workload policy overrides namespace strict" "$results/workload-conflict.json" '
	. as $report |
	($report.workloadPostures | length) == 1 and
	$report.workloadPostures[0].mtls.effective == "disabled" and
	any($report.workloadPostures[0].mtls.chain[]; .kind == "PeerAuthentication" and .namespace == "omg-workload-conflict" and .name == "conflict-api") and
	any($report.findings[];
	  .controlId == "MG-MTLS-003" and
	  .severity == "critical" and
	  .status == "open"
	)
'
assert_json "port override remains honest unknown without workload-port evidence" "$results/port-level-override.json" '
	.workloadPostures | length == 1 and
	.[0].workload == {"namespace": "omg-port-override", "name": "port-api", "kind": "Deployment"} and
	.[0].dataPlaneMode == "sidecar" and
	.[0].mtls.effective == "unknown" and
	.[0].mtls.chain == [] and
	.[0].mtls.unknownReason == "workload ports unavailable for port-level PeerAuthentication on omg-port-override/port-override" and
	.[0].authorization.effective == "unknown"
'
assert_json "DestinationRule contradiction remains unavailable evidence" "$results/dr-contradiction.json" '
	.workloadPostures | length == 1 and
	.[0].mtls.effective == "strict" and
	(.[0].mtls | has("clientTLSContradiction") | not)
'
assert_json "injection-disabled fixture remains honest membership unknown" "$results/not-in-mesh.json" '
	.workloadPostures | length == 1 and
	.[0].dataPlaneMode == "unknown" and
	.[0].mtls.effective == "unknown" and
	.[0].mtls.unknownReason == "data plane membership unavailable"
'
assert_json "namespace Role degrades denied root policy evidence" "$results/namespace-role-degraded.json" '
	(.workloadPostures | length) > 0 and
	(.findings | length) > 0 and
	any(.permissionSummary[]; .apiGroup == "" and .resource == "namespaces" and .granted == false) and
	any(.permissionSummary[]; .apiGroup == "security.istio.io" and .resource == "peerauthentications" and .granted == false) and
	all(.workloadPostures[]; .mtls.effective == "unknown") and
	all(.findings[]; .status == "unknown")
'

capture_audit
echo "e2e: prove API-server audit saw only approved get/list resource calls from scanner identities"
if ! jq -s -e \
	--arg cluster_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_CLUSTER_SCANNER" \
	--arg namespace_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_NAMESPACE_SCANNER" \
	--arg probe_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:audit-probe" '
	def allowed_resource:
	  (.objectRef.apiGroup // "") as $group |
	  .objectRef.resource as $resource |
	  if $group == "" then
	    ["namespaces", "pods", "services"] | index($resource) != null
	  elif $group == "apps" then
	    ["daemonsets", "deployments", "replicasets", "statefulsets"] | index($resource) != null
	  elif $group == "discovery.k8s.io" then
	    ["endpointslices"] | index($resource) != null
	  elif $group == "networking.istio.io" then
	    ["destinationrules", "envoyfilters", "gateways", "proxyconfigs", "serviceentries", "sidecars", "virtualservices", "workloadentries", "workloadgroups"] | index($resource) != null
	  elif $group == "security.istio.io" then
	    ["authorizationpolicies", "peerauthentications", "requestauthentications"] | index($resource) != null
	  elif $group == "telemetry.istio.io" then
	    $resource == "telemetries"
	  elif $group == "gateway.networking.k8s.io" then
	    ["backendtlspolicies", "gatewayclasses", "gateways", "grpcroutes", "httproutes", "referencegrants", "tcproutes", "tlsroutes", "udproutes"] | index($resource) != null
	  else false
	  end;
	map(select(.user.username == $cluster_user or .user.username == $namespace_user)) as $events |
	($events | length > 0) and
	all($events[];
	  (.verb == "get" or .verb == "list") and
	  ((.objectRef.subresource // "") == "") and
	  allowed_resource
	) and
	any($events[]; .user.username == $cluster_user) and
	any($events[]; .user.username == $namespace_user) and
	any($events[];
	  .user.username == $namespace_user and
	  .verb == "list" and
	  .objectRef.apiGroup == "security.istio.io" and
	  .objectRef.resource == "peerauthentications" and
	  .objectRef.namespace == "istio-system" and
	  .responseStatus.code == 403
	) and
	any(.[];
	  .user.username == $probe_user and
	  .verb == "create" and
	  (.objectRef.apiGroup // "") == "" and
	  .objectRef.resource == "configmaps" and
	  .responseStatus.code == 403
	)
' "$results/audit.jsonl" >/dev/null; then
	echo "audit proof failed; inspect $results/audit.jsonl" >&2
	exit 1
fi

while IFS="$tab" read -r name namespace deployment proxy expected_findings; do
	compare_golden "$name"
done <"$cases"
compare_golden namespace-role-degraded

events=$(jq -s \
	--arg cluster_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_CLUSTER_SCANNER" \
	--arg namespace_user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_NAMESPACE_SCANNER" '
	[.[] | select(.user.username == $cluster_user or .user.username == $namespace_user)] | length
' "$results/audit.jsonl")
finished=$(date +%s)
echo "RBAC proofs passed; scanner audit contains $events approved get/list events and no other calls"
echo "e2e duration: $((finished - started))s"
