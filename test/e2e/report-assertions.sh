#!/bin/sh

assert_golden_case_bijection() {
	report_guard_cases=$1
	report_guard_goldens=$2
	report_guard_include_degraded=${3:-false}
	report_guard_declared=$(
		{
			awk -F '\t' 'NF > 0 {print $1}' "$report_guard_cases"
			if [ "$report_guard_include_degraded" = true ]; then
				printf '%s\n' namespace-role-degraded
			fi
		} | LC_ALL=C sort
	)
	report_guard_actual=$(
		for report_guard_golden in "$report_guard_goldens"/*.json; do
			[ -f "$report_guard_golden" ] || continue
			basename "$report_guard_golden" .json
		done | LC_ALL=C sort
	)
	report_guard_update=$(printenv UPDATE_GOLDEN 2>/dev/null || true)
	if [ "$report_guard_update" = 1 ]; then
		for report_guard_name in $report_guard_actual; do
			if ! printf '%s\n' "$report_guard_declared" | grep -Fqx "$report_guard_name"; then
				echo "golden update found undeclared golden $report_guard_name" >&2
				return 1
			fi
		done
		return 0
	fi
	if [ "$report_guard_declared" != "$report_guard_actual" ]; then
		echo "fixture cases and golden files are not an exact bijection" >&2
		echo "declared:" >&2
		printf '%s\n' "$report_guard_declared" >&2
		echo "goldens:" >&2
		printf '%s\n' "$report_guard_actual" >&2
		return 1
	fi
}

assert_report_update_guard() {
	report_guard_name=$1
	report_guard_file=$2
	report_guard_expected_findings=$3
	report_guard_actual_findings=$(jq -r '
		[.findings[] | "\(.controlId)=\(.status)"] | sort | join(",")
	' "$report_guard_file")
	if [ "$report_guard_actual_findings" != "$report_guard_expected_findings" ]; then
		echo "semantic assertion failed: $report_guard_name findings were [$report_guard_actual_findings], want [$report_guard_expected_findings] ($report_guard_file)" >&2
		return 1
	fi

		if ! jq -e '
			def nonempty_chain:
			  type == "array" and length > 0;
			all(.workloadPostures[];
			  (.mtls.effective == "unknown" or (.mtls.chain | nonempty_chain)) and
			  (.authorization.effective == "unknown" or (.authorization.chain | nonempty_chain))
			) and
			all(.findings[];
			  .status == "unknown" or (.resolutionChain | nonempty_chain)
			)
		' "$report_guard_file" >/dev/null; then
		echo "semantic assertion failed: $report_guard_name resolved conclusions and findings require non-empty resolution chains ($report_guard_file)" >&2
		return 1
	fi
}

# Human-approved M7 migration boundary. Compare every existing field except
# the new runtime outcomes and their explicitly associated metadata. This guard
# runs BEFORE an update can replace a pre-M7 golden.
assert_m7_declared_compatibility() {
 m7_previous=$1
 m7_actual=$2
 m7_temporary=$(mktemp -d "${TMPDIR:-/tmp}/openmeshguard-m7-compat.XXXXXX")
 m7_projection='
   ([.findings[] | select(.evidenceType == "runtime" and
     (.controlId == "MG-MTLS-101" or .controlId == "MG-MTLS-102") and .status == "unknown")] | length) as $runtime_unknown |
   .findings |= map(select((.evidenceType == "runtime" and
     (.controlId == "MG-MTLS-101" or .controlId == "MG-MTLS-102")) | not)) |
   .permissionSummary |= map(select((.apiGroup == "prometheus" and .resource == "query") | not)) |
   (.permissionSummary[] | select(has("affectedControls")) | .affectedControls) |=
     map(select(. != "MG-MTLS-101" and . != "MG-MTLS-102")) |
   (.scanner.controlPacks[] | select(.name == "builtin-mtls" and .source == "builtin") | .version) = "0.3.0" |
   .scores.categories |= map(if .category == "mtls" then
     .unknown = ((.unknown // 0) - $runtime_unknown) |
     if .unknown == 0 then del(.unknown) else . end
     else . end)
 '
 if ! jq -S "$m7_projection" "$m7_previous" >"$m7_temporary/previous.json" ||
    ! jq -S "$m7_projection" "$m7_actual" >"$m7_temporary/actual.json"; then
  rm -rf "$m7_temporary"
  return 1
 fi
 if ! diff -u "$m7_temporary/previous.json" "$m7_temporary/actual.json"; then
  echo "M7 golden update rejected: existing declared report changed ($m7_actual)" >&2
  rm -rf "$m7_temporary"
  return 1
 fi
 rm -rf "$m7_temporary"
}
