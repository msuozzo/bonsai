#!/bin/sh
# Tag the release declared by main's go.mods. This ONLY creates git tags on
# GitHub -- it does not commit, edit go.mods, or move bookmarks.
#
# The version is not an input: release prep (RELEASING.md) writes it into
# every language go.mod, so this script derives it from main and syncs
# GitHub's tags to match. Rerunning is always safe: a tag that already
# exists at main's commit is skipped, so a fully tagged release is a no-op
# and a partial failure is resumed by rerunning.
#
#   ./release.sh -n          # dry run: show what would be tagged
#   ./release.sh             # tag whatever main's go.mods declare
#   ./release.sh vX.Y.Z      # same, but fail unless main declares vX.Y.Z
#
# Validations, all before any tag is created:
#   - main is immutable (pushed, not a rewritable WIP)
#   - every cross-module require at main agrees on one version
#   - an existing tag at any other commit fails the release
#
# The module list is derived from the bonsai-*/build.env files at main, so
# new languages are tagged automatically.
# (Go resolves submodule ".../bonsai-x" from the tag "bonsai-x/vX.Y.Z".)
set -eu
cd "$(dirname "$0")"

REPO="msuozzo/bonsai"
BASE="github.com/msuozzo/bonsai"

die() {
	echo "release: $*" >&2
	exit 1
}

DRY=0
if [ "${1:-}" = "-n" ]; then
	DRY=1
	shift
fi
WANT="${1:-}"

# Print the version of the require of $BASE in the go.mod on stdin.
# Matches both go.mod shapes (`require <path> <ver>` and a block entry)
# because the pattern only anchors on the path and version themselves.
base_require() {
	sed -n "s|.*$BASE \(v[^[:space:]]*\).*|\1|p" | head -1
}

# Fail fast on auth/repo problems, before any per-tag work.
gh api "repos/$REPO" --silent >/dev/null || die "gh cannot reach $REPO (auth?)"

SHA=$(jj log --no-graph -r main -T commit_id 2>/dev/null) || SHA=
[ -n "$SHA" ] || die "no 'main' bookmark"

# main must be immutable: pushed, and not a WIP we might still rewrite.
[ -z "$(jj log --no-graph -r 'main & mutable()' -T commit_id 2>/dev/null)" ] \
	|| die "main ($SHA) is mutable -- commit + push it before tagging"

# Module list from main, defined as all those with build.env files.
MODS=$(jj file list -r main | sed -n 's|^\(bonsai-[^/]*\)/build\.env$|\1|p')
[ -n "$MODS" ] || die "no bonsai-*/build.env at main"
for m in $MODS; do
	FIRST=$m
	break
done

# The release version is whatever main's go.mods declare.
VER=$(jj file show -r main "$FIRST/go.mod" 2>/dev/null | base_require)
printf '%s\n' "$VER" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' \
	|| die "could not find a vX.Y.Z version from $FIRST/go.mod at main (got '$VER')"
[ -z "$WANT" ] || [ "$WANT" = "$VER" ] || die "main's go.mods declare $VER, not $WANT"

# A version sitting in the working copy but not on main is the most likely
# way to run this too early. Point at the missing step instead of silently
# re-syncing the old release.
WCVER=$(base_require <"$FIRST/go.mod" 2>/dev/null) || WCVER=
if [ -n "$WCVER" ] && [ "$WCVER" != "$VER" ]; then
	echo "warning: working-copy go.mods differ from main ($WCVER != $VER)" >&2
	echo "         commit + push main to release $WCVER." >&2
fi

# Validate that each bonsai dep in go.mod files uses only new version VER.
# Checks both base requires as well as lang-to-lang requires like
# bonsai-markdown -> bonsai-markdown-inline. The go.mod files are written by
# `go mod edit`, so requires are format-normalized.
for m in $MODS; do
	REQS=$(jj file show -r main "$m/go.mod" 2>/dev/null | grep -E "$BASE(/[^ ]+)? v") || REQS=
	printf '%s\n' "$REQS" | grep -qF "$BASE $VER" \
		|| die "$m/go.mod at main lacks '$BASE $VER' -- bump go.mods first"
	if printf '%s\n' "$REQS" | grep -qv " $VER\$"; then
		die "$m/go.mod at main has stale bonsai requires -- bump go.mods first"
	fi
done

# Calculate tags to create.
TAGS="$VER"
for m in $MODS; do
	TAGS="$TAGS $m/$VER"
done

# Dry run exits early.
if [ "$DRY" = 1 ]; then
	echo "would tag $VER at $SHA:"
	for t in $TAGS; do
		echo "  $t"
	done
	exit 0
fi

# Actual run creates all tags it can, logging errors on failure.
FAIL=0
CREATED=0
for t in $TAGS; do
	EXISTING=$(gh api "repos/$REPO/git/ref/tags/$t" --jq .object.sha 2>/dev/null) || EXISTING=
	if [ "$EXISTING" = "$SHA" ]; then
		echo "exists  $t"
	elif [ -n "$EXISTING" ]; then
		echo "release: $t already exists at $EXISTING, not $SHA" >&2
		FAIL=1
	elif gh api "repos/$REPO/git/refs" -f ref="refs/tags/$t" -f sha="$SHA" --silent; then
		echo "tagged  $t"
		CREATED=$((CREATED + 1))
	else
		echo "release: could not tag $t" >&2
		FAIL=1
	fi
done

[ "$FAIL" = 0 ] || die "$VER is INCOMPLETE -- fix the errors above and rerun"
if [ "$CREATED" = 0 ]; then
	echo "Nothing to do: $VER is already fully tagged at $SHA."
else
	echo "Released $VER at $SHA."
fi
