# Releasing

All modules release in lockstep: the root runtime and every
`bonsai-<lang>` module get the same `vX.Y.Z` tag, every release, whether
or not a given module changed. Every cross-module `require` points at
exactly that version.

As a result, the version number carries no grammar information. The upstream
grammar and lib pins live in each module's `build.env` and in the generated
`GrammarVersion` and `TreeSitterVersion` constants, respectively.

## Patch or minor?

Roughly abides by semver. Here are a few scenarios and how we map them to
version bumps:

| change                                            | release |
| ------------------------------------------------- | ------- |
| toolchain bump (wasm2go, binaryen, wasi-sdk, Go)¹ | patch   |
| runtime bugfix                                    | patch   |
| tree-sitter grammar or core bump                  | minor   |
| new language module²                              | minor   |
| runtime API addition                              | minor   |
| anything breaking (pre-1.0)                       | minor   |

¹ Provided the regenerated output is behavior-identical.

² New modules debut at the release's version.

## Process

1. Bump every cross-module require to the new version:

   ```sh
   v=vX.Y.Z
   for m in bonsai-*/; do
     go mod edit -require=github.com/msuozzo/bonsai@"$v" "${m%/}/go.mod"
   done
   go mod edit -require=github.com/msuozzo/bonsai/bonsai-markdown-inline@"$v" \
     bonsai-markdown/go.mod
   ```

2. Commit, point `main` at it, and push.

3. Tag the root and every language module at `main` and push.
