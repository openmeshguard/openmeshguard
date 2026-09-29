#!/bin/sh
# Administrator fixture setup is separate from the read-only scanner credential.
set -eu
umask 077
. "$(dirname -- "$0")/lib.sh"
cd "$E2E_ROOT"
require_command docker
require_command kubectl
require_command jq
require_command curl
scanner_binary=${OPENMESHGUARD_E2E_BINARY:-"$E2E_ROOT/bin/openmeshguard"}
[ -x "$scanner_binary" ] || { echo 'run make build first' >&2; exit 1; }
KIND=$(kind_binary)
credentials=
forward_pid=
cleanup() {
    if [ -n "$forward_pid" ]; then
        kill "$forward_pid" 2>/dev/null || true
        wait "$forward_pid" 2>/dev/null || true
    fi
    if [ -n "$credentials" ]; then rm -rf "$credentials"; fi
    rm -f "$E2E_ADMIN_KUBECONFIG"
}
trap cleanup EXIT
trap 'exit 130' HUP INT TERM
"$KIND" export kubeconfig --name "$E2E_CLUSTER_NAME" --kubeconfig "$E2E_ADMIN_KUBECONFIG" >/dev/null
chmod 600 "$E2E_ADMIN_KUBECONFIG"
credentials=$(mktemp -d "$E2E_STATE_DIR/runtime-credentials.XXXXXX")
results="$E2E_STATE_DIR/runtime-results"
mkdir -p "$results"
fixtures="$E2E_ROOT/test/fixtures/runtime-verification"
admin_kubectl apply -f "$fixtures/resources.yaml" >/dev/null
admin_kubectl -n omg-runtime rollout status deployment/runtime-receiver --timeout=180s
admin_kubectl -n omg-runtime rollout status deployment/runtime-prometheus --timeout=180s
# Reuse the audit policy's scanner identity, with only the published role.
admin_kubectl apply -f "$E2E_ROOT/deploy/rbac/cluster-role.yaml" >/dev/null
admin_kubectl create namespace "$E2E_HARNESS_NAMESPACE" --dry-run=client -o yaml | admin_kubectl apply -f - >/dev/null
admin_kubectl -n "$E2E_HARNESS_NAMESPACE" create serviceaccount "$E2E_CLUSTER_SCANNER" --dry-run=client -o yaml | admin_kubectl apply -f - >/dev/null
admin_kubectl create clusterrolebinding openmeshguard-e2e-cluster-scan --clusterrole=openmeshguard-cluster-scan \
    --serviceaccount="$E2E_HARNESS_NAMESPACE:$E2E_CLUSTER_SCANNER" --dry-run=client -o yaml | admin_kubectl apply -f - >/dev/null
admin_kubectl -n "$E2E_HARNESS_NAMESPACE" create token "$E2E_CLUSTER_SCANNER" --duration=10m > "$credentials/token"
admin_kubectl config view --raw --minify --flatten -o json | jq --rawfile token "$credentials/token" '
    .users = [{name:"runtime-scanner",user:{token:($token|rtrimstr("\n"))}}] |
    .contexts[0].context.user = "runtime-scanner"
' > "$credentials/scanner.json"
rm -f "$credentials/token"
# Forwarding is a harness administrator action; scanner only receives a URL.
admin_kubectl -n omg-runtime port-forward --address=127.0.0.1 service/runtime-prometheus :9090 > "$results/port-forward.log" 2>&1 &
forward_pid=$!
port=
ready=false
for attempt in $(seq 1 30); do
    port=$(sed -n 's/^Forwarding from 127\.0\.0\.1:\([0-9]*\) -> .*/\1/p' "$results/port-forward.log" | head -1)
    if [ -n "$port" ] && curl -fsS --max-time 2 "http://127.0.0.1:$port/-/ready" >/dev/null; then ready=true; break; fi
    sleep 1
done
[ "$ready" = true ] || { cat "$results/port-forward.log" >&2; exit 1; }
admin_kubectl -n omg-runtime delete job runtime-plaintext runtime-meshed --ignore-not-found >/dev/null
admin_kubectl apply -f "$fixtures/traffic-plaintext.yaml" -f "$fixtures/traffic-meshed.yaml" >/dev/null
admin_kubectl -n omg-runtime wait --for=condition=complete job/runtime-plaintext job/runtime-meshed --timeout=240s
# Wait until both HTTP and TCP counters have both security labels in Prometheus.
metric_query='sum by (__name__, connection_security_policy) ({__name__=~"istio_requests_total|istio_tcp_connections_opened_total",reporter="destination",destination_workload="runtime-receiver",destination_workload_namespace="omg-runtime"})'
for attempt in $(seq 1 30); do
    curl -fsS --max-time 5 --get --data-urlencode "query=$metric_query" "http://127.0.0.1:$port/api/v1/query" > "$results/metrics.json"
    if jq -e '[.data.result[] | select((.value[1]|tonumber)>0) | [.metric.__name__,.metric.connection_security_policy]] | unique | length == 4' "$results/metrics.json" >/dev/null; then break; fi
    sleep 2
done
jq -e '[.data.result[] | select((.value[1]|tonumber)>0) | [.metric.__name__,.metric.connection_security_policy]] | unique | sort == [["istio_requests_total","mutual_tls"],["istio_requests_total","none"],["istio_tcp_connections_opened_total","mutual_tls"],["istio_tcp_connections_opened_total","none"]]' "$results/metrics.json" >/dev/null
set +e
"$scanner_binary" scan --kubeconfig "$credentials/scanner.json" --namespace omg-runtime \
    --prometheus-url "http://127.0.0.1:$port" --prometheus-lookback 5m --prometheus-step 5s > "$results/report.json"
scan_status=$?
set -e
[ "$scan_status" -eq 0 ] || { echo "scan failed: $scan_status" >&2; exit 1; }
jq -e '
    . as $report |
    [.workloadPostures[] | select(.workload.name=="runtime-receiver") | .verified] as $verified |
    ($verified|length)==1 and $verified[0].plaintextObserved==true and
    $verified[0].mtlsTrafficShare>0 and $verified[0].mtlsTrafficShare<1 and
    all(["MG-MTLS-101","MG-MTLS-102"][]; . as $control |
        any($report.findings[]; .controlId==$control and .status=="open" and
            any(.resources[]; .name=="runtime-receiver" and .namespace=="omg-runtime")))
' "$results/report.json" >/dev/null
docker exec "$E2E_CLUSTER_NAME-control-plane" cat /var/log/kubernetes/audit.log > "$results/audit.jsonl"
jq -s -e --arg user "system:serviceaccount:$E2E_HARNESS_NAMESPACE:$E2E_CLUSTER_SCANNER" '
    [.[] | select(.user.username==$user)] as $calls |
    ($calls|length)>0 and all($calls[];
        (.verb=="get" or .verb=="list") and .objectRef.resource!="secrets" and
        .impersonatedUser==null and .objectRef.subresource!="token")
' "$results/audit.jsonl" >/dev/null
OPENMESHGUARD_SCHEMA_REPORT="$results/report.json" go test ./internal/output -run '^TestExternalScanOutputMatchesSchema$' -count=1 >/dev/null
printf 'Live HTTP + TCP plaintext/mTLS demonstration passed: %s\n' "$results/report.json"
