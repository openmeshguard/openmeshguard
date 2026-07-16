#!/bin/sh

set -eu

. "$(dirname -- "$0")/lib.sh"

require_command docker
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
audit_policy="$E2E_ROOT/test/e2e/audit-policy.yaml"
kind_config="$E2E_STATE_DIR/kind-config.yaml"

cat >"$kind_config" <<EOF
apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: ClusterConfiguration
        apiServer:
          extraArgs:
            audit-log-path: /var/log/kubernetes/audit.log
            audit-log-mode: blocking-strict
            audit-policy-file: /etc/kubernetes/policies/audit-policy.yaml
          extraVolumes:
            - name: audit-policies
              hostPath: /etc/kubernetes/policies
              mountPath: /etc/kubernetes/policies
              readOnly: true
              pathType: DirectoryOrCreate
            - name: audit-logs
              hostPath: /var/log/kubernetes
              mountPath: /var/log/kubernetes
              readOnly: false
              pathType: DirectoryOrCreate
    extraMounts:
      - hostPath: $audit_policy
        containerPath: /etc/kubernetes/policies/audit-policy.yaml
        readOnly: true
      - hostPath: $E2E_STATE_DIR/audit
        containerPath: /var/log/kubernetes
EOF

started=$(date +%s)
"$KIND" create cluster \
	--name "$E2E_CLUSTER_NAME" \
	--image "$KIND_NODE_IMAGE" \
	--config "$kind_config" \
	--wait 180s

"$ISTIOCTL" install \
	--context "$E2E_CONTEXT" \
	--set profile=default \
	--skip-confirmation

admin_kubectl -n istio-system rollout status deployment/istiod --timeout=300s

finished=$(date +%s)
echo "Kind $($(kind_binary) version | awk '{print $2}'), node $KIND_NODE_IMAGE, Istio $ISTIO_VERSION"
echo "kind-up duration: $((finished - started))s"
