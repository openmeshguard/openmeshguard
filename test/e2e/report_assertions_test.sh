#!/bin/sh

set -eu

TEST_SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/openmeshguard-report-assertions.XXXXXX")
trap 'rm -rf "$TEST_ROOT"' EXIT HUP INT TERM

. "$TEST_SCRIPT_DIR/report-assertions.sh"

fixtures="$TEST_SCRIPT_DIR/../fixtures/sidecar-basic"
strict_expected=$(awk -F '\t' '$1 == "strict" {print $5}' "$fixtures/cases.tsv")
permissive_expected=$(awk -F '\t' '$1 == "permissive" {print $5}' "$fixtures/cases.tsv")

assert_report_update_guard strict "$fixtures/golden/strict.json" "$strict_expected"
assert_report_update_guard permissive "$fixtures/golden/permissive.json" "$permissive_expected"

jq 'del(.findings[] | select(.controlId == "MG-MTLS-001"))' \
	"$fixtures/golden/permissive.json" >"$TEST_ROOT/missing-finding.json"
if assert_report_update_guard permissive "$TEST_ROOT/missing-finding.json" "$permissive_expected" 2>/dev/null; then
	echo "report guard accepted a missing expected finding" >&2
	exit 1
fi

jq '(.workloadPostures[].mtls.chain) = []' \
	"$fixtures/golden/strict.json" >"$TEST_ROOT/missing-posture-chain.json"
if assert_report_update_guard strict "$TEST_ROOT/missing-posture-chain.json" "$strict_expected" 2>/dev/null; then
	echo "report guard accepted a resolved posture without a chain" >&2
	exit 1
fi

jq '(.findings[].resolutionChain) = []' \
	"$fixtures/golden/permissive.json" >"$TEST_ROOT/missing-finding-chain.json"
if assert_report_update_guard permissive "$TEST_ROOT/missing-finding-chain.json" "$permissive_expected" 2>/dev/null; then
	echo "report guard accepted resolved findings without chains" >&2
	exit 1
fi

echo "E2E report assertion tests passed"
