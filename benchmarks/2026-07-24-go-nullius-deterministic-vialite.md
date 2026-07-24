# go-nullius deterministic harness vs vialite-todo (6 injected defects)

**Date:** 2026-07-24 · **Driver:** `cmd/nullius-det` (deterministic state machine)
· **Models:** smart=minimax-m2.7, fast=qwen3.6 (local llama.cpp, $0) · **Mode:**
read-only audit (no craftsman → find→judge→confirm→plan, no writes) · **Wall:** 77.7s.

First run of the deterministic go-nullius harness against a real injected-defect
bench (the same `harness/tasks/vialite-todo` skeleton the claude-p arms use).
Task was generic/class-level (not defect-enumerating) but did mention "optimistic
concurrency (CAS)".

## Score: recall 1/6, precision 2/5

| Injected defect | file / symbol | class | result |
|---|---|---|---|
| cas-lost-update | stateapp.go / Update | lost-update | **✅ caught exactly** (correct root cause + fix plan: retry loop on ErrCASConflict) |
| statesess-cross-session-leak | statesess.go / Update | isolation | ⚠️ symbol flagged, **wrong reason** (ruled a CAS-rev issue, not the `sess=nil` fan-out) |
| subscription-overwake | ctx.go / subscribed | wake-predicate | ❌ missed |
| sse-premature-clear | sse.go / drainQueue | fault-survival | ❌ missed |
| ttl-sweeps-connected | runtime.go / removeExpiredContexts | lifecycle | ❌ missed |
| action-not-serialized | action.go / runAction | serialization | ❌ missed |

**False positives (3 of 5 confirmed):** `backplanetest/conformance.go:265,272`
(CAS calls in a test-HELPER file — not `*_test.go`, so the filter missed it);
`crypto.go:224` (a genuine swallowed-`json.Unmarshal` smell, but not an injected
defect — and the judge ruled on the json error under a `cas-method-call` lens →
off-lens free-association).

## Root cause: Recon single-theme fixation

Recon derived only 3 lenses, ALL CAS-themed (`cas-write-lens` witness-failed;
`cas-method-call` + `cas-deferred-method-call` accepted). Because the task said
"optimistic concurrency (CAS)", the weak smart model fixated on that one theme and
was blind to the other 5 defect classes. A single recon pass over 63 files picks
ONE risk theme; the always-on baseline floor (`stmt-after-return` only) is far too
thin to cover the rest. **The harness is currently a single-lens sniper, not a
broad auditor — coverage is bottlenecked at Recon.**

## What worked

- The one catch was clean and precise, with a correct fix plan.
- `cas-method-call` was PROMOTED to the on-disk lens library (promotion works live).
- Plan-overlap dedup collapsed 5 confirmed → 4 plan targets.
- The refuter-evidence gate REFUTED one weak candidate (conformance.go:282).
- Fail-closed held throughout; no drain (read-only).
- Pair-discrimination / smart-escalation / audit-reentry did not trigger (no
  same-lens CANT_TELL ties; read-only so no drain/re-entry).

## Fable counsel (2026-07-24) — reframed

Fable's key correction: this is **not primarily a recon problem — the coverage
floor specced in DESIGN:57-59 (~5 mechanical class lenses) was never built.**
`defaults.go` ships only `stmt-after-return`. And the template library is
**call-expression-only**, so recon was *expressively incapable* of 3 of the 5
misses (tautology, statement-order, guard-condition) regardless of breadth.
Multi-theme recon buys more CAS-shaped lenses, not the missing classes.

Where the lens-circularity line is: a lens is circular iff its content is
*instance-identifying* (fixture names, the seeded literal, a shape tuned to the
bug). A lens encoding a defect *class* from the pre-existing taxonomy, using only
language/stdlib vocabulary (`Lock`, `defer`, `len(x)>=0`), running unchanged on any
codebase, is legitimate baseline — `stmt-after-return` already is one.

**Ranked fixes (fable):**
1. **Baseline build-out (1c)** — 5 mechanical WALK lenses (not query floods):
   `bool-tautology` (const-fold `len>=0`/`x==x`/`uint<0` → over-wake miss),
   `lock-without-release-in-fn`, `clear-before-use-in-fn` (sibling-order → sse miss),
   `nil-literal-arg` (→ sess=nil, judge/pair discriminates), `write-to-guarded-
   field-without-lock` (→ action.go miss). Covers 4/5, immune to model failure,
   promotable every run.
2. **Fix 3 — lens/judge coherence** — RESTORE DESIGN:29 filter-1 (decisive line must
   contain the lens mechanism token / fall in the candidate span; `validLine` was
   silently weakened to in-window+nonblank). Constrain `judgePrompt` to the lens's
   concern. Off-lens finds become `off_lens_note` LEADS (no confirm/plan/promotion) —
   **critical: off-lens confirms currently poison `promoteConfirmed`** (crypto.go
   would have reinforced `cas-method-call` for a json error).
3. **Fix 2 — testsupport classification** at machine level (`BuildTerrain`/Mandate,
   not cmd): `*_test.go`→test; imports `"testing"`→testsupport; dir/pkg ending `test`
   or ∈{testdata,mocks,fakes,stubs}→testsupport. Exclude from Enumerate, KEEP in
   digest (labeled), record in Notes (no silent exclusion). Suffix/segment match only
   (`latest/`, `contest.go` traps). Do first — an afternoon, removes 2/3 FPs.
4. **Sharded recon (1b)** — one recon call per fixed lens class (theme imposed per
   call → fixation structurally impossible), per-shard fail-closed, ~3000 tok/shard,
   class-prefixed IDs. **Deferred by fable's own ranking: needs new non-call templates
   (condition-shape, assignment-shape) + a richer terrain digest (imports +
   go/select/send/mutex-field counts + uncapped fn list) first, else shards steer
   blind.**

Also flagged (not yet built): judge's enclosing-function window can't see
cross-function facts (sess=nil's meaning lives at the callee) — the designed
CST-slice callee-summary feeding `judgePrompt` is the unlock for D2 recall.

Reproduce: seed `harness/tasks/vialite-todo/skeleton` into a git dir, build
`cmd/nullius-det`, run over the non-`_test.go` `.go` files with a class-level task.

## Re-run after increment 1 (2026-07-24, same task/files)

After landing `bool-tautology` baseline + test-support exclusion + lens/judge
coherence (Evidence gate):

- **Precision 1/1** (was 2/5). All 3 FPs eliminated: 4 test-support files excluded
  from the hunt (conformance.go, vt/vt.go, …), logged in Notes; the off-lens
  crypto.go-style ruling can no longer confirm.
- **subscription-overwake (#2) now CAUGHT** — `ctx.go:457 subscribed:
  len(ctx.lastReads) >= 0 is always true`, by the always-on `bool-tautology`
  baseline, INDEPENDENT of recon. Missed entirely in run 1.
- **cas-lost-update (#1) NOT caught this run** — recon is stochastic and derived no
  usable CAS lens this time (run 1's catch was a recon fluke). Confirms fable's
  thesis: recon-dependent catches are luck; baseline lenses are the durable coverage.

Net per-run recall still 1/6, but the caught defect is now a deterministic,
100%-precision baseline catch that recurs every run, not a recon fluke. The
remaining 3 baseline lenses (clear-before-use, nil-literal-arg,
write-to-guarded-field) will add durable coverage for the sse / sess-leak /
action classes. 292.9s (slower than run 1's 77.7s — recon on the flaky smart
endpoint dominated).
