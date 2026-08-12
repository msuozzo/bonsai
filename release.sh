#!/bin/sh
# Tag the release declared by main's go.mods. This ONLY creates git tags and
# will not commit, edit go.mods, or move bookmarks.
#
# The version must be updated in all go.mod files, committed, and pushed to
# main before running (see RELEASING.md). Rerunning the script is safe since
# pushed tags are skipped, partial failures are resumed from where they
# encountered errors, and tags will never be moved (`jj tag set` refuses).
#
#   ./release.sh -n          # dry run: validate + show what would be tagged
#   ./release.sh             # tag whatever main's go.mods declare
#   ./release.sh vX.Y.Z      # same, but fail unless main declares vX.Y.Z
#
# Validations, all before any tag is created:
#   - main is immutable (pushed, not a rewritable WIP)
#   - main matches main@REMOTE after a fetch, so we tag what the remote has
#   - every cross-module require at main agrees on one version
#   - an existing tag at any other commit fails the release
#
# The module list is derived from the bonsai-*/build.env files at main, so
# new languages are tagged automatically.
# (Go resolves submodule ".../bonsai-x" from the tag "bonsai-x/vX.Y.Z".)
set -eu
cd "$(dirname "$0")"

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

# Fail fast on a jj that cannot push tags (stabilized in 0.44.0).
jj tag set --help >/dev/null 2>&1 || die "need jj (>=v0.44.0)"

# Find and release to remote matching $BASE (first match wins).
REMOTE=$(jj git remote list | tr ':' '/' \
	| grep -E "$BASE(\.git)?/?$" \
	| head -1 | cut -d' ' -f1)
[ -n "$REMOTE" ] || die "no remote has a $BASE url"

# Ensure we're synced to the upstream before fetching main, tags.
jj git fetch --quiet --remote "$REMOTE" || die "fetch from $REMOTE failed"

SHA=$(jj log --no-graph -r main -T commit_id 2>/dev/null) || SHA=
[ -n "$SHA" ] || die "no 'main' bookmark"

# main must be immutable: pushed, and not a WIP we might still rewrite.
[ -z "$(jj log --no-graph -r 'main & mutable()' -T commit_id 2>/dev/null)" ] \
	|| die "main ($SHA) is mutable -- commit + push it before tagging"

# Identify remote's main as target to tag.
RSHA=$(jj log --no-graph -r "main@$REMOTE" -T commit_id 2>/dev/null) || RSHA=
[ "$RSHA" = "$SHA" ] || die "main ($SHA) != main@$REMOTE (${RSHA:-absent}) -- push main first"

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

# Reject a bump version in the working copy but not on main.
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

# Partition into missing tags and tags already at main.
# An existing tag at any other commit fails the release.
[ "$DRY" = 0 ] || echo "would tag $VER at $SHA:"
MISSING=
for t in $TAGS; do
	AT=$(jj tag list "exact:$t" -T 'if(remote, "", normal_target.commit_id())' 2>/dev/null) || AT=
	if [ -z "$AT" ]; then
		MISSING="$MISSING $t"
		[ "$DRY" = 0 ] || echo "  $t"
	elif [ "$AT" != "$SHA" ]; then
		die "$t already exists at $AT, not $SHA"
	else
		[ "$DRY" = 0 ] || echo "  $t (exists)"
	fi
done
[ "$DRY" = 0 ] || exit 0

# Create the missing tags in one batch (jj refuses existing tags).
[ -z "$MISSING" ] || jj tag set $MISSING -r main
jj git push --remote "$REMOTE" $(printf ' -t exact:%s' $TAGS) \
	|| die "$VER is INCOMPLETE -- fix the errors above and rerun"

if [ -z "$MISSING" ]; then
	echo "Nothing to do: $VER was already fully tagged at $SHA (push synced)."
else
	echo "Released $VER at $SHA."
fi
