// Package enumerate is the mechanical finding layer of the deterministic
// orchestrator: it turns source files into typed Candidates, one per site a lens
// matched. It never judges (DEFECT/CORRECT is a downstream forced choice); it only
// finds, and it can only find patterns that are actually present in source read from
// disk, so a Candidate cannot be fabricated.
//
// Two lens backends, per DESIGN-deterministic.md step 4:
//
//   - QueryLens — a compiled tree-sitter .scm query (SHAPE properties). A malformed
//     query errors at compile (NewQuery), which is the first mechanical gate.
//   - WalkLens  — a Go-side CST walk (ORDER / LIFECYCLE properties that .scm cannot
//     express: statement order, dominance, before/after). Parameterized in Go.
//
// Lenses come from two sources that COMPOSE, never replace:
//
//   - Baseline — the always-on coverage floor per language. Model-derived lenses may
//     only ADD to it; a failed derivation degrades to baseline-only. This is the rule
//     that keeps recall honest when the model authors the finder.
//   - Derived  — authored at recon by the (weak, local) model, either by filling
//     PARAMETERS into a pre-verified Template (tier 1) or as free-form .scm (tier 2).
//     Every derived lens passes the witness Gate before it is trusted: it must match
//     its own positive snippet and must NOT match its negative snippet. A lens that
//     cannot demonstrate what it catches is vacuous and is dropped (rule-5 for lenses).
package enumerate
