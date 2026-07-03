#!/usr/bin/env bash
set -euo pipefail

blocked=0

while IFS= read -r -d '' file; do
	line_number=0
	while IFS= read -r line || [[ -n "${line}" ]]; do
		line_number=$((line_number + 1))
		if [[ "${line}" =~ ^[[:space:]]*(import[[:space:]]+)?([._[:alnum:]]+[[:space:]]+)?\"(os|net/http|k8s\.io/client-go(/[^\"\ ]*)?)\" ]]; then
			echo "${file}:${line_number}: internal/resolver imports forbidden package: ${BASH_REMATCH[3]}" >&2
			blocked=1
		fi
	done < "${file}"
done < <(find internal/resolver -name '*.go' -print0)

exit "${blocked}"
