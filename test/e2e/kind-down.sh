#!/bin/sh

set -eu

. "$(dirname -- "$0")/lib.sh"

KIND=$(kind_binary)
started=$(date +%s)
"$KIND" delete cluster --name "$E2E_CLUSTER_NAME"
finished=$(date +%s)
echo "kind-down duration: $((finished - started))s"
