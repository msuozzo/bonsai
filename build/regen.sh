#!/usr/bin/env bash
# Hermetic regen for one grammar module. Reads the grammar pins from
# bonsai-<lang>/build.env, builds the per-grammar builder image (see
# Dockerfile.builder) if it isn't already present, then runs it against
# the repo to rewrite bonsai-<lang>/{module_gen.go,module_gen.dat,
# libc_gen.go,meta_gen.go}.
#
# Usage:
#   build/regen.sh <lang>               # ensure image exists, regenerate
#   build/regen.sh all                  # ... every lang with a build.env
#   build/regen.sh --image-only <lang>  # (re)build the image, skip regen
#
# `go generate ./bonsai-<lang>` (from the repo root) is equivalent to
# `build/regen.sh <lang>`.
#
# The image is only built when missing. After editing Dockerfile.builder
# or build.env, force a rebuild with --image-only. Docker layer caching
# makes a no-change rebuild cheap.
set -euo pipefail

image_only=0
if [[ ${1:-} == --image-only ]]; then
  image_only=1
  shift
fi
lang=${1:?usage: build/regen.sh [--image-only] <lang>|all}

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

if [[ $lang == all ]]; then
  found=0
  for f in bonsai-*/build.env; do
    [[ -e $f ]] || continue
    found=1
    l=${f%/build.env}
    if ((image_only)); then
      "$0" --image-only "${l#bonsai-}"
    else
      "$0" "${l#bonsai-}"
    fi
  done
  ((found)) || { echo "regen.sh: no bonsai-*/build.env found" >&2; exit 1; }
  exit 0
fi

env_file=bonsai-$lang/build.env
if [[ ! -f $env_file ]]; then
  known=$(ls -d bonsai-*/build.env 2>/dev/null \
    | sed 's|^bonsai-||; s|/build.env$||' | paste -sd' ' -)
  echo "regen.sh: $env_file not found (known langs: ${known:-none})" >&2
  exit 1
fi

# One --build-arg per KEY=VALUE line. Blank lines and # comments are skipped.
build_args=()
while IFS= read -r line; do
  [[ -z $line || $line == \#* ]] && continue
  if [[ ! $line =~ ^[A-Z_]+=[^[:space:]]+$ ]]; then
    echo "regen.sh: malformed line in $env_file: $line" >&2
    exit 1
  fi
  build_args+=(--build-arg "$line")
done <"$env_file"

image=bonsai-builder-$lang:dev
if ((image_only)) || ! docker image inspect "$image" >/dev/null 2>&1; then
  docker build --platform linux/amd64 -f Dockerfile.builder \
    "${build_args[@]}" -t "$image" .
fi
if ((image_only)); then
  exit 0
fi

exec docker run --rm --platform linux/amd64 \
  -v "$root:/work" -u "$(id -u):$(id -g)" "$image"
