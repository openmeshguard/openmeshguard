#!/usr/bin/env python3
"""Fail closed before uploads can mutate an already public release."""
import json
import os
import re
import subprocess
import sys

REPOSITORY = "openmeshguard/openmeshguard"


def api(path):
    result = subprocess.run(
        ["gh", "api", "--hostname", "github.com", "--include", path],
        capture_output=True,
        text=True,
        check=False,
    )
    # gh --include prints the HTTP status and headers before the JSON body.
    parts = result.stdout.replace("\r\n", "\n").split("\n\n", 1)
    match = re.match(r"HTTP/\S+ ([0-9]{3})(?:\s|$)", parts[0])
    if not match or len(parts) != 2:
        raise RuntimeError("GitHub API returned no usable HTTP response")
    status = int(match.group(1))
    if status not in (200, 404) or (status == 200 and result.returncode != 0):
        raise RuntimeError(f"GitHub API request failed (HTTP {status})")
    try:
        body = json.loads(parts[1])
    except json.JSONDecodeError as error:
        raise RuntimeError("GitHub API returned invalid JSON") from error
    return status, body


def main():
    if len(sys.argv) != 2 or not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+", sys.argv[1]):
        raise RuntimeError("usage: preflight.py <vMAJOR.MINOR.PATCH>")
    # Never probe anonymously: a missing or inaccessible release can both be 404.
    if not (os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")):
        raise RuntimeError("authenticated GH_TOKEN or GITHUB_TOKEN is required")
    tag = sys.argv[1]
    status, repository = api(f"repos/{REPOSITORY}")
    if status != 200 or repository.get("full_name") != REPOSITORY:
        raise RuntimeError("cannot confirm access to the release repository")
    status, release = api(f"repos/{REPOSITORY}/releases/tags/{tag}")
    if status == 404:
        print(f"Release preflight passed: {tag} does not exist")
        return
    if release.get("tag_name") != tag or release.get("draft") is not True:
        raise RuntimeError("refusing uploads: exact-tag release is public or not a confirmed draft")
    print(f"Release preflight passed: {tag} remains a draft")


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, AttributeError) as error:
        print(f"Release preflight failed: {error}", file=sys.stderr)
        sys.exit(1)
