#!/bin/sh
# Prints the RELEASE_NOTES.md section for a version (e.g. 0.2.0), without its
# "## vX.Y.Z" heading and with "###" subheadings promoted to "##" for the
# GitHub release page. Fails if the section is missing or empty.
set -eu
version=$1
notes="$(dirname "$0")/../RELEASE_NOTES.md"
section=$(awk -v h="## v$version" '
	$0 == h { found = 1; next }
	found && /^## / { exit }
	found { sub(/^### /, "## "); print }
' "$notes")
if [ -z "$(printf '%s' "$section" | tr -d '[:space:]')" ]; then
	echo "release-notes: no \"## v$version\" section in RELEASE_NOTES.md" >&2
	exit 1
fi
printf '%s\n' "$section" | sed -e '/./,$!d'
