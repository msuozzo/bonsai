# syntax=docker/dockerfile:1.7
#
# bonsai wasm builder (amd64 only). Parameterized per-grammar via
# --build-arg. Each grammar's config lives in bonsai-<NAME>/build.env and
# is plumbed in by build/regen.sh.
#
#   build/regen.sh --image-only python   # builds bonsai-builder-python:dev
#   build/regen.sh python                # runs the image against the repo
#                                        # (same as `go generate ./bonsai-python`)
#
# To bump tool versions: edit the SDK/Binaryen/wasm2go ARGs below and
# refresh the SHA256s by sha256sum'ing the new tarballs.

# ---------- Stage 1: fetch toolchains + grammar + build Go-side tools ----------
# --platform pins both stages to linux/amd64 so wasi-sdk and binaryen (we
# only ship x86_64-linux) match the image's arch.
FROM --platform=linux/amd64 golang:1.26-bookworm AS prep

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl git \
 && rm -rf /var/lib/apt/lists/*

# Toolchain pins (same across all grammars).
ARG WASI_SDK_VERSION=33.0
ARG WASI_SDK_SHA256=0ba8b5bfaeb2adf3f29bab5841d76cf5318ab8e1642ea195f88baba1abd47bce
ARG BINARYEN_VERSION=130
ARG BINARYEN_SHA256=0a18362361ad05465118cd8eeb72edaeec89de6894bc283576ef4e07aa3babcc
ARG TREE_SITTER_TAG=v0.25.10
# Pinned to a commit on the msuozzo/wasm2go fork, which carries a fix not yet
# upstreamed (the approach in ncruces/wasm2go#40). Revert to a tagged
# ncruces/wasm2go release once the fix lands upstream.
ARG WASM2GO_VERSION=f9fd3b0b022e6375d7a50643127535b97aee259f

# Grammar pins (vary per grammar, see bonsai-<name>/build.env).
# GRAMMAR_SRC_SUBDIR points at the directory holding src/ when the repo
# keeps the grammar out of its root (dialect monorepos, split grammars).
# Leave it empty for the common src/-at-root layout.
ARG GRAMMAR_NAME
ARG GRAMMAR_REPO
ARG GRAMMAR_TAG
ARG GRAMMAR_DIR
ARG GRAMMAR_SRC_SUBDIR=""
ARG LANGUAGE_SYMBOL
ARG GRAMMAR_HAS_SCANNER=1

# Build wasm2go and libc-gen as static binaries so the final image
# doesn't need a Go toolchain.
#
# We fetch the fork at the exact WASM2GO_VERSION commit instead of `go install`
# because the fork keeps the upstream module path, which breaks module-version
# installs. Once the fix lands upstream (ncruces/wasm2go#40), revert to a
# tagged upstream release:
#
#   RUN CGO_ENABLED=0 go install -trimpath -ldflags='-s -w' \
#           github.com/ncruces/wasm2go@${WASM2GO_VERSION} \
#    && CGO_ENABLED=0 go install -trimpath -ldflags='-s -w' \
#           github.com/ncruces/wasm2go/libc-gen@${WASM2GO_VERSION}
RUN set -eux; \
    mkdir /tmp/wasm2go; \
    cd /tmp/wasm2go; \
    git init -q; \
    git remote add origin https://github.com/msuozzo/wasm2go.git; \
    git fetch -q --depth=1 origin "${WASM2GO_VERSION}"; \
    git checkout -q FETCH_HEAD; \
    CGO_ENABLED=0 go install -trimpath -ldflags='-s -w' . ./libc-gen; \
    rm -rf /tmp/wasm2go /root/.cache/go-build

# wasi-sdk: download, verify SHA, extract, prune.
RUN set -eux; \
    URL="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-${WASI_SDK_VERSION%.*}/wasi-sdk-${WASI_SDK_VERSION}-x86_64-linux.tar.gz"; \
    curl -fsSL -o /tmp/wasi.tgz "$URL"; \
    echo "${WASI_SDK_SHA256}  /tmp/wasi.tgz" | sha256sum -c -; \
    mkdir -p /opt/wasi-sdk; \
    tar xzf /tmp/wasi.tgz -C /opt/wasi-sdk --strip-components=1; \
    rm /tmp/wasi.tgz; \
    # bonsai passes -nostdlib + its own libc/, so wasi-sysroot (~440 MB) is dead weight.
    rm -rf /opt/wasi-sdk/share/wasi-sysroot \
           /opt/wasi-sdk/share/man /opt/wasi-sdk/share/doc /opt/wasi-sdk/share/misc \
           /opt/wasi-sdk/lib/cmake /opt/wasi-sdk/lib/pkgconfig; \
    # NOTE: keep libedit.so* because clang and ld.lld dynamic-link against it.
    rm -f /opt/wasi-sdk/lib/liblldb.so*; \
    rm -f /opt/wasi-sdk/bin/lldb* \
          /opt/wasi-sdk/bin/clang-tidy /opt/wasi-sdk/bin/clang-format \
          /opt/wasi-sdk/bin/clang-scan-deps /opt/wasi-sdk/bin/clang-apply-replacements \
          /opt/wasi-sdk/bin/clang-cl /opt/wasi-sdk/bin/lld-link /opt/wasi-sdk/bin/ld64.lld \
          /opt/wasi-sdk/bin/git-clang-format

# binaryen: download, verify SHA, keep only wasm-opt + wasm-dis.
RUN set -eux; \
    URL="https://github.com/WebAssembly/binaryen/releases/download/version_${BINARYEN_VERSION}/binaryen-version_${BINARYEN_VERSION}-x86_64-linux.tar.gz"; \
    curl -fsSL -o /tmp/bin.tgz "$URL"; \
    echo "${BINARYEN_SHA256}  /tmp/bin.tgz" | sha256sum -c -; \
    mkdir -p /opt/binaryen; \
    tar xzf /tmp/bin.tgz -C /opt/binaryen --strip-components=1; \
    rm /tmp/bin.tgz; \
    find /opt/binaryen/bin -type f \
         ! -name wasm-opt ! -name wasm-dis -delete

# Sources: tree-sitter core + the grammar named in GRAMMAR_*. The grammar
# is cloned into /sources/$GRAMMAR_DIR. Provenance is recorded for the
# generated meta_gen.go header.
RUN set -eux; \
    git clone --depth 1 --branch "${TREE_SITTER_TAG}" \
        https://github.com/tree-sitter/tree-sitter.git /sources/tree-sitter; \
    git clone --depth 1 --branch "${GRAMMAR_TAG}" \
        "${GRAMMAR_REPO}" "/sources/${GRAMMAR_DIR}"; \
    (cd /sources/tree-sitter            && git rev-parse HEAD > /sources/tree-sitter.sha); \
    (cd "/sources/${GRAMMAR_DIR}"       && git rev-parse HEAD > /sources/grammar.sha); \
    echo "${TREE_SITTER_TAG}"  > /sources/tree-sitter.version; \
    echo "${GRAMMAR_TAG}"      > /sources/grammar.version; \
    echo "${GRAMMAR_DIR}"      > /sources/grammar.dir; \
    echo "${GRAMMAR_REPO}"     > /sources/grammar.repo; \
    rm -rf /sources/tree-sitter/.git "/sources/${GRAMMAR_DIR}/.git"


# ---------- Stage 2: minimal runtime ----------
FROM --platform=linux/amd64 debian:bookworm-slim

# Repeat the ARGs here because Docker ARGs don't cross stage
# boundaries. The toolchain ARGs re-take their stage-1 defaults so the
# LABEL below tracks them automatically. The grammar ARGs are required
# (no default), so callers must pass them through to both stages.
ARG WASI_SDK_VERSION=33.0
ARG BINARYEN_VERSION=130
ARG TREE_SITTER_TAG=v0.25.10
ARG WASM2GO_VERSION=f9fd3b0b022e6375d7a50643127535b97aee259f
ARG GRAMMAR_NAME
ARG GRAMMAR_DIR
ARG GRAMMAR_SRC_SUBDIR=""
ARG LANGUAGE_SYMBOL
ARG GRAMMAR_HAS_SCANNER=1

COPY --from=prep /opt/wasi-sdk    /opt/wasi-sdk
COPY --from=prep /opt/binaryen    /opt/binaryen
COPY --from=prep /sources         /sources
COPY --from=prep /go/bin/wasm2go  /usr/local/bin/wasm2go
COPY --from=prep /go/bin/libc-gen /usr/local/bin/libc-gen

LABEL bonsai.wasi_sdk_version="${WASI_SDK_VERSION}" \
      bonsai.binaryen_version="${BINARYEN_VERSION}" \
      bonsai.tree_sitter_tag="${TREE_SITTER_TAG}" \
      bonsai.wasm2go_version="${WASM2GO_VERSION}" \
      bonsai.grammar_name="${GRAMMAR_NAME}"

# Everything the entrypoint needs derives from the grammar ARGs.
ENV WASI_SDK_PATH=/opt/wasi-sdk \
    BINARYEN_PATH=/opt/binaryen \
    TS_CORE_PATH=/sources/tree-sitter \
    TS_GRAMMAR_PATH=/sources/${GRAMMAR_DIR} \
    GRAMMAR_NAME=${GRAMMAR_NAME} \
    GRAMMAR_SRC_SUBDIR=${GRAMMAR_SRC_SUBDIR} \
    LANGUAGE_SYMBOL=${LANGUAGE_SYMBOL} \
    GRAMMAR_HAS_SCANNER=${GRAMMAR_HAS_SCANNER} \
    OUTPUTS=bonsai-${GRAMMAR_NAME} \
    PATH=/opt/wasi-sdk/bin:/opt/binaryen/bin:/usr/local/bin:${PATH}

# Embed the regen entrypoint. Single-quoted EOF disables shell expansion so
# $VARS survive verbatim and get evaluated at container-run time.
COPY --chmod=0755 <<'EOF' /usr/local/bin/regen
#!/usr/bin/env bash
# Entrypoint for the bonsai-builder image. Assumes the repo is mounted at
# /work. Toolchains + grammar source are baked into the image at the paths
# in $WASI_SDK_PATH, $BINARYEN_PATH, $TS_CORE_PATH, $TS_GRAMMAR_PATH.
#
# Per-grammar env vars (baked at build time):
#   $GRAMMAR_NAME            "python", ...
#   $GRAMMAR_SRC_SUBDIR      "" or the subdir holding src/ (e.g. "dialects/terraform")
#   $LANGUAGE_SYMBOL         "tree_sitter_python", ...
#   $GRAMMAR_HAS_SCANNER     "0" or "1" (does the grammar ship scanner.c)
#   $OUTPUTS                 e.g. bonsai-python
#   $WASM_PACKAGE            e.g. bonsaipython (Go package + wasm name)
set -euo pipefail
cd /work

# Go package + wasm filename. Hyphens are stripped from GRAMMAR_NAME
# so multi-word grammars like "markdown-inline" produce a valid Go
# identifier ("bonsaimarkdowninline").
WASM_PACKAGE=bonsai${GRAMMAR_NAME//-/}

INPUTS=build
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

libc-gen -c-out "$WORK/libc"

# Common wasm exports plus the grammar-specific language symbol.
EXPORTS=()
while IFS=' ' read -r symbol _; do
  [[ -z "$symbol" || "$symbol" == \#* ]] && continue
  EXPORTS+=("-Wl,--export=$symbol")
done < "$INPUTS/exports.txt"
EXPORTS+=("-Wl,--export=${LANGUAGE_SYMBOL}")

# Where src/ actually lives: the repo root, or a subdirectory for
# dialect monorepos and split grammars.
GRAMMAR_SRC="$TS_GRAMMAR_PATH${GRAMMAR_SRC_SUBDIR:+/$GRAMMAR_SRC_SUBDIR}"

# Grammar sources: parser.c always, plus scanner.c if the grammar ships one.
GRAMMAR_SRCS=("$GRAMMAR_SRC/src/parser.c")
if [[ "$GRAMMAR_HAS_SCANNER" == "1" ]]; then
  GRAMMAR_SRCS+=("$GRAMMAR_SRC/src/scanner.c")
fi

# Compile tree-sitter core + grammar + shims to wasm. We link directly to
# $WASM_PACKAGE.wasm so wasm-ld writes that name into the wasm's
# module-name subsection. libc-gen reads it to choose its Go package
# (ignoring -pkg when -wasm is given).
# NOTE: grammars with very large lexer state machines (~1000+ states,
# e.g. tree-sitter-markdown) currently can't ship through this pipeline:
# wasm's structured control flow encodes the state dispatch as deeply
# nested blocks, wasm2go translates those to equally nested Go blocks,
# and go/types (vet, gopls) rejects the result with "exceeded max scope
# depth". The fix belongs in wasm2go (flat goto-based emission past a
# depth threshold); -fno-jump-tables does not help (the nesting comes
# from the branch-target bodies, not the dispatch).
clang --target=wasm32 -ffreestanding -nostdlib -std=c11 -g0 -Oz \
  -DNDEBUG -D__wasi__ \
  -Wall -Wextra -Wno-unused-parameter -Wno-unused-function \
  -include "$INPUTS/shim.h" \
  -o "$WORK/$WASM_PACKAGE.wasm" \
  "$INPUTS/lib_min.c" "$INPUTS/stubs.c" "$WORK/libc/libc.c" \
  "${GRAMMAR_SRCS[@]}" \
  -I"$TS_CORE_PATH/lib/include" -I"$TS_CORE_PATH/lib/src" -I"$GRAMMAR_SRC/src" \
  -I"$INPUTS/inc" -I"$WORK/libc" \
  -mexec-model=reactor \
  -mmutable-globals -mmultivalue \
  -mnontrapping-fptoint -msign-ext \
  -mreference-types -mbulk-memory \
  -mextended-const -mtail-call \
  -Wl,--stack-first \
  -Wl,--export-table \
  -Wl,--import-memory \
  -Wl,--import-undefined \
  "${EXPORTS[@]}"

# Optimize. Read the linked module, write back to the same path.
mv "$WORK/$WASM_PACKAGE.wasm" "$WORK/$WASM_PACKAGE.linked"
wasm-opt -g "$WORK/$WASM_PACKAGE.linked" -o "$WORK/$WASM_PACKAGE.wasm" \
  --low-memory-unused --converge -O4 \
  --enable-mutable-globals --enable-multivalue \
  --enable-nontrapping-float-to-int --enable-sign-ext \
  --enable-reference-types --enable-bulk-memory \
  --enable-extended-const --enable-tail-call \
  --strip --strip-producers

# Go side of the provided libc. libc-gen derives the package from the wasm
# module-name section (appending `_wasm` via mangle.Local) and ignores its
# -pkg flag when -wasm is given. Rewrite the package line so it matches.
libc-gen -wasm "$WORK/$WASM_PACKAGE.wasm" -o "$OUTPUTS/libc_gen.go"
sed -i "s/^package .*/package $WASM_PACKAGE/" "$OUTPUTS/libc_gen.go"

# Translate the wasm to Go. `-embed` writes the data segment to a sibling
# .dat that the generated file picks up via go:embed.
wasm2go -embed -unsafe -pkg "$WASM_PACKAGE" \
  -provided "$OUTPUTS/libc_gen.go" \
  -o "$OUTPUTS/module_gen.go" \
  "$WORK/$WASM_PACKAGE.wasm"

# Extract the wasm's declared initial-memory pages (last integer on the
# memory-import line of the WAT disassembly).
init_pages=$(
  wasm-dis "$WORK/$WASM_PACKAGE.wasm" \
    | grep '"memory"' \
    | grep -oE '[0-9]+' \
    | tail -n1
)
if [[ -z "$init_pages" ]]; then
  echo "could not extract initial memory pages from wasm" >&2
  exit 1
fi

# Format the input lines via printf so the two columns align regardless
# of grammar-name length.
{
  echo "// Code generated by bonsai-builder. DO NOT EDIT."
  echo "//"
  echo "// Inputs (baked into image):"
  printf "//   %-22s %s  (commit %s)\n" \
    "tree-sitter"                  "$(cat /sources/tree-sitter.version)" "$(cat /sources/tree-sitter.sha)"
  printf "//   %-22s %s  (commit %s)\n" \
    "$(cat /sources/grammar.dir)"  "$(cat /sources/grammar.version)"     "$(cat /sources/grammar.sha)"
  echo "//"
  echo "// Grammar source:     $(cat /sources/grammar.repo)"
  echo "// Toolchain versions: see image labels (docker inspect)."
  echo
  echo "package $WASM_PACKAGE"
  echo
  echo "// InitialPages is the wasm module's declared initial linear-memory size,"
  echo "// in 64 KiB WebAssembly pages. Memory passed to New must already be sized"
  echo "// to at least InitialPages * 65536 bytes before the call."
  echo "const InitialPages = $init_pages"
} > "$OUTPUTS/meta_gen.go"

# Upstream license notices. The generated module is a derivative of the
# tree-sitter runtime and the grammar (both compiled into the wasm), so
# their notices must accompany copies: ship them as files in the module
# zip, and go:embed them so the text also rides into consuming binaries
# (the linker never strips embeds from a compiled package). The dotted
# names are NOT in pkg.go.dev's recognized-filename set. This is intentional:
# the module's own LICENSE handles pkgsite detection, these are notices.
find_license() {
  local f
  for f in LICENSE LICENSE.md LICENSE.txt LICENSE-MIT.txt COPYING COPYING.md; do
    if [[ -f "$1/$f" ]]; then
      printf '%s\n' "$1/$f"
      return 0
    fi
  done
  return 1
}
core_lic=$(find_license "$TS_CORE_PATH") \
  || { echo "no license file in tree-sitter core" >&2; exit 1; }
# Subdir grammars usually inherit the repo-root license. Check the
# grammar's own directory first, then fall back to the repo root.
grammar_lic=$(find_license "$GRAMMAR_SRC") \
  || grammar_lic=$(find_license "$TS_GRAMMAR_PATH") \
  || { echo "no license file in grammar $TS_GRAMMAR_PATH: cannot redistribute" >&2; exit 1; }
cp "$core_lic" "$OUTPUTS/LICENSE.tree-sitter"
cp "$grammar_lic" "$OUTPUTS/LICENSE.grammar"

{
  echo "// Code generated by bonsai-builder. DO NOT EDIT."
  echo "//"
  echo "// Upstream license notices for the code compiled into this module."
  echo
  echo "package $WASM_PACKAGE"
  echo
  echo 'import _ "embed"'
  echo
  echo "// LicenseTreeSitter is the license of the tree-sitter runtime"
  echo "// ($(cat /sources/tree-sitter.version)) whose compiled form this package embeds."
  echo "//"
  echo "//go:embed LICENSE.tree-sitter"
  echo "var LicenseTreeSitter string"
  echo
  echo "// LicenseGrammar is the license of the grammar"
  echo "// ($(cat /sources/grammar.repo) @ $(cat /sources/grammar.version))"
  echo "// whose compiled form this package embeds."
  echo "//"
  echo "//go:embed LICENSE.grammar"
  echo "var LicenseGrammar string"
} > "$OUTPUTS/licenses_gen.go"

echo "OK: regenerated $OUTPUTS/{module_gen.go, libc_gen.go, module_gen.dat, meta_gen.go, licenses_gen.go, LICENSE.tree-sitter, LICENSE.grammar}"
EOF

WORKDIR /work
ENTRYPOINT ["/usr/local/bin/regen"]
