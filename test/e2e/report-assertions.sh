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
