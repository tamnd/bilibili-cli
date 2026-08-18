#!/usr/bin/env bash
#
# Print the CHANGELOG.md section for one version, for GoReleaser's
# --release-notes flag.
#
# The generated alternative is a list of commit subjects, which says what was
# done and never says what it means for somebody who has the previous version
# installed. Exit codes changing from two values to eight is one line in a
# changelog and is invisible in a commit list.
#
# Missing section is a failure, not an empty file. A release that quietly ships
# with no notes is worse than one that does not ship until somebody writes them.
#
# Usage: scripts/release_notes.sh v0.3.0 [CHANGELOG.md]

set -euo pipefail

tag="${1:?usage: release_notes.sh <tag> [changelog]}"
file="${2:-CHANGELOG.md}"
version="${tag#v}"

notes="$(awk -v want="## v${version}" '
	$0 == want        { collecting = 1; next }
	collecting && /^## / { exit }
	collecting        { print }
' "${file}")"

# Trim the blank lines the section boundaries leave behind.
notes="$(printf '%s\n' "${notes}" | sed -e '/./,$!d' -e :a -e '/^\n*$/{$d;N;ba' -e '}')"

if [ -z "${notes}" ]; then
	echo "${file} has no '## v${version}' section, so ${tag} has no release notes" >&2
	exit 1
fi

printf '%s\n' "${notes}"
