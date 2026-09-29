#!/bin/sh

set -eu

TEST_SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/openmeshguard-report-assertions.XXXXXX")
trap 'rm -rf "$TEST_ROOT"' EXIT HUP INT TERM

. "$TEST_SCRIPT_DIR/report-assertions.sh"

fixtures="$TEST_SCRIPT_DIR/../fixtures/sidecar-basic"
governance_fixtures="$TEST_SCRIPT_DIR/../fixtures/governance-context"
strict_expected=$(awk -F '\t' '$1 == "strict" {print $5}' "$fixtures/cases.tsv")
permissive_expected=$(awk -F '\t' '$1 == "permissive" {print $5}' "$fixtures/cases.tsv")
active_exception_expected=$(awk -F '\t' '$1 == "governance-active-exception" {print $5}' "$governance_fixtures/cases.tsv")
expired_exception_expected=$(awk -F '\t' '$1 == "governance-expired-exception" {print $5}' "$governance_fixtures/cases.tsv")

assert_golden_case_bijection "$fixtures/cases.tsv" "$fixtures/golden" true
assert_golden_case_bijection "$governance_fixtures/cases.tsv" "$governance_fixtures/golden"
assert_report_update_guard strict "$fixtures/golden/strict.json" "$strict_expected"
assert_report_update_guard permissive "$fixtures/golden/permissive.json" "$permissive_expected"
assert_report_update_guard governance-active-exception \
	"$governance_fixtures/golden/governance-active-exception.json" \
	"$active_exception_expected"
assert_report_update_guard governance-expired-exception \
	"$governance_fixtures/golden/governance-expired-exception.json" \
	"$expired_exception_expected"
degraded="$fixtures/golden/namespace-role-degraded.json"
degraded_expected=$(jq -r '
	[.findings[] | "\(.controlId)=\(.status)"] | sort | join(",")
' "$degraded")

mkdir -p "$TEST_ROOT/bijection/golden"
printf 'strict\tnamespace\tdeployment\tsidecar\tfinding=unknown\n' >"$TEST_ROOT/bijection/cases.tsv"
printf '{}\n' >"$TEST_ROOT/bijection/golden/strict.json"
printf '{}\n' >"$TEST_ROOT/bijection/golden/namespace-role-degraded.json"
assert_golden_case_bijection "$TEST_ROOT/bijection/cases.tsv" "$TEST_ROOT/bijection/golden" true

printf '{}\n' >"$TEST_ROOT/bijection/golden/stale.json"
if assert_golden_case_bijection "$TEST_ROOT/bijection/cases.tsv" "$TEST_ROOT/bijection/golden" true 2>/dev/null; then
	echo "golden bijection accepted a stale golden" >&2
	exit 1
fi
rm "$TEST_ROOT/bijection/golden/stale.json"
rm "$TEST_ROOT/bijection/golden/strict.json"
if assert_golden_case_bijection "$TEST_ROOT/bijection/cases.tsv" "$TEST_ROOT/bijection/golden" true 2>/dev/null; then
	echo "golden bijection accepted a missing golden" >&2
	exit 1
fi
(
	export UPDATE_GOLDEN=1
	assert_golden_case_bijection "$TEST_ROOT/bijection/cases.tsv" "$TEST_ROOT/bijection/golden" true
)

jq 'del(.findings[] | select(.controlId == "MG-MTLS-002"))' \
	"$fixtures/golden/permissive.json" >"$TEST_ROOT/missing-finding.json"
if assert_report_update_guard permissive "$TEST_ROOT/missing-finding.json" "$permissive_expected" 2>/dev/null; then
	echo "report guard accepted a missing expected finding" >&2
	exit 1
fi

jq 'del(.findings[] | select(.controlId == "MG-MTLS-001"))' \
	"$governance_fixtures/golden/governance-active-exception.json" \
	>"$TEST_ROOT/missing-excepted-finding.json"
if assert_report_update_guard governance-active-exception \
	"$TEST_ROOT/missing-excepted-finding.json" "$active_exception_expected" 2>/dev/null
then
	echo "report guard accepted removal of an excepted finding" >&2
	exit 1
fi

jq 'del(.findings[] | select(.controlId == "MG-MTLS-001"))' \
	"$governance_fixtures/golden/governance-expired-exception.json" \
	>"$TEST_ROOT/missing-restored-finding.json"
if assert_report_update_guard governance-expired-exception \
	"$TEST_ROOT/missing-restored-finding.json" "$expired_exception_expected" 2>/dev/null
then
	echo "report guard accepted removal of an expired-restored finding" >&2
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

jq '
	(.findings[] | select(.status != "unknown") | .resolutionChain) = []
' "$degraded" >"$TEST_ROOT/missing-degraded-finding-chain.json"
if assert_report_update_guard namespace-role-degraded "$TEST_ROOT/missing-degraded-finding-chain.json" "$degraded_expected" 2>/dev/null; then
	echo "report guard accepted degraded resolved findings without chains" >&2
	exit 1
fi

echo "E2E report assertion tests passed"

# The additive migration must accept runtime metadata, but must reject changes
# to any established conclusion, finding, permission, or unrelated score count.
base="$fixtures/golden/strict.json"
jq '
 .findings += [{controlId:"MG-MTLS-101",evidenceType:"runtime",status:"unknown"}] |
 .permissionSummary += [{apiGroup:"prometheus",resource:"query",verbs:["get"],granted:false,optional:true}] |
 (.scanner.controlPacks[] | select(.name=="builtin-mtls") | .version) = "0.4.0" |
 (.scores.categories[] | select(.category=="mtls") | .unknown) += 1
' "$base" >"$TEST_ROOT/approved-runtime-addition.json"
assert_m7_declared_compatibility "$base" "$TEST_ROOT/approved-runtime-addition.json"
for mutation in \
 '(.workloadPostures[0].mtls.effective)="disabled"' \
 '(.findings[0].reasoning)="changed existing reasoning"' \
 '(.permissionSummary[0].granted)=false' \
 '(.scores.categories[] | select(.category=="authz") | .unknown)+=1' \
 '(.scores.categories[] | select(.category=="mtls") | .unknown)+=1'
do
 jq "$mutation" "$TEST_ROOT/approved-runtime-addition.json" >"$TEST_ROOT/declared-drift.json"
 if assert_m7_declared_compatibility "$base" "$TEST_ROOT/declared-drift.json" >/dev/null 2>&1; then
  echo "M7 guard accepted declared drift: $mutation" >&2
  exit 1
 fi
done
echo "M7 declared compatibility guard tests passed"
