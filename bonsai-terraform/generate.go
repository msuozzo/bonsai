// Regen directive for the bonsai-terraform grammar. Grammar pins live in
// build.env. build/regen.sh builds the per-grammar builder image on
// demand, then runs it against the repo to rewrite the *_gen.* files.
// Copy build.env + this file as a starting point for a new grammar.
//
//go:generate ../build/regen.sh terraform

package bonsaiterraform
