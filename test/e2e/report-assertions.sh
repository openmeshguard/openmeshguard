#!/bin/sh

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
		if any(.workloadPostures[]; .mtls.effective != "unknown") then
		  all(.findings[]; (.resolutionChain | nonempty_chain))
		else
		  true
		end
	' "$report_guard_file" >/dev/null; then
		echo "semantic assertion failed: $report_guard_name resolved conclusions and findings require non-empty resolution chains ($report_guard_file)" >&2
		return 1
	fi
}
