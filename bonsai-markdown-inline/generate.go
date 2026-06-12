// Regen directive for the bonsai-markdown-inline grammar. Grammar
// pins live in build.env. build/regen.sh builds the per-grammar
// builder image on demand, then runs it against the repo to rewrite
// the *_gen.* files.
//
//go:generate ../build/regen.sh markdown-inline

package bonsaimarkdowninline
