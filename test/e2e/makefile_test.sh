#!/bin/sh

set -eu

TEST_SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TEST_ROOT=$(CDPATH= cd -- "$TEST_SCRIPT_DIR/../.." && pwd)
test_binary="${TMPDIR:-/tmp}"
test_binary="${test_binary%/}/openmeshguard-makefile-test/openmeshguard"
dry_run=$(make -n -C "$TEST_ROOT" BINARY="$test_binary" e2e)

if ! printf '%s\n' "$dry_run" | grep -F "go build -o \"$test_binary\" ./cmd/openmeshguard" >/dev/null; then
	echo "make e2e did not build the overridden binary" >&2
	exit 1
fi
if ! printf '%s\n' "$dry_run" | grep -F "OPENMESHGUARD_E2E_BINARY=\"$test_binary\" ./test/e2e/run.sh" >/dev/null; then
	echo "make e2e did not pass the overridden binary to the harness" >&2
	exit 1
fi

echo "E2E Makefile binary propagation test passed"
