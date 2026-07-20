# FINDINGS

The honest ledger. Every result the project measured — wins, refutations,
and the assumptions it got wrong — with sample size and caveat attached.
Most projects publish the wins and bury the graveyard. Here the graveyard
*is* the argument: the point was never "nullius is good," it was "measure
before you believe anything, including this."

**How things were measured.** Headless [harness](benchmarks/harness/)
(`run.sh`): one fresh git worktree per rep, a pinned container toolchain
so the only variable is the arm, cost captured per model from the CLI's
own usage, and scoring that never trusts a run's self-report — a hidden
suite plus a blind disclosure judge. Cost is list-price-equivalent even
under subscription auth.

---

## Confirmed — held up under measurement

- **Reads delegated to cheap throwaway contexts is the core win.** On the
  seeded-defect oracle, nullius (fable, low effort) fixed 6/6 race-clean
  at **$6.17** vs a plain fable *high-effort* run's 6/6 at **$23.34** —
  same quality, 26% of the cost. Against the same-effort baseline (plain
  fable low, **$12.36**, 4.33/6, n=3) nullius was half the cost at
  strictly better quality (5.75/6, race-clean 4/4), every rep of n=4.

- **Enforcement is load-bearing, not decorative.** The plugin's cost
  curve on the same task, tightening the diet rep by rep: $13.27 → $11.08
  → $8.09 → **$7.54** at 6/6. The one regression ($8.09 dropped to 5/6)
  came from *softening* the hunt — the signature defect shipped silently.
  Re-hardening restored 6/6. A softened brownfield hunt is a measured
  failure mode, not a hypothetical.

- **Quoted-mechanism discipline catches what testimony misses.** A
  hunter's clearance on a quoted-but-non-covering mechanism ("buffered
  chan-1 + hold/pending") let an always-true wake predicate ship (5/6).
  The fix — a REFUTED ruling on a core lens needs leader-read lines or a
  pinning test, never testimony alone — took the same run's next rep to
  6/6. Verified live again: an in-session model insisted the hook was
  inactive and the command "ran as written"; both false (it read the
  wrong config; the rewrite was silent). Self-reported green lies.

- **Prevention beats compaction** (prevention-vs-compaction A/B, same
  task/model, both capped at a 120k context window). Prevention
  (cc-nullius): 5/6 then 6/6, race-clean, $10.50 / $8.53. Compaction
  (plain, let it bloat then auto-compact): 4/6, race-*unclean*, $14.56,
  and it closed claiming "all green under -race" while three tests
  failed. Honest wrinkle: neither arm showed the *predicted* failure
  signature (a defect identified-then-lost across the compact boundary) —
  the plain arm never formed the suspicion at all. n=1 clean per arm
  (rep 2 was scrubbed on a spend limit).

- **Recursion pays on large absorption.** Porting a 154k-token Nuxt/Vue
  frontend to Go: a nested nullius session (`nullius-build`, sonnet +
  its own haiku scouts) finished at **$5.54**; a plain cold sonnet doing
  the same **timed out at $17.19**. 3.1× cheaper and the difference
  between shipping and not — the nested session's sonnet context stayed
  ~5× leaner (7.8M vs 41.2M cache-reads). n=1.

---

## Refuted — tried, measured, discarded

- **Ceremony is not rigor.** The `byproxy-v6` architecture (contracts,
  red-team critics, cold auditors, compiled gates) cost $12–14 of premium
  context per run and **never beat a plain top-tier run**; the full
  ceremony peaked at $20.56 for 3/6. Kept runnable in
  [`archive/byproxy-v6/`](archive/byproxy-v6/).

- **Judgment does not delegate downtier.** A mid-tier (sonnet) control
  plane under the full process capped at **3/6, twice** — it found the
  double-dispose both times and botched the same fix the same way.
  Detection mechanizes; interpreting the constraint took the tier.

- **Delegated writes are a tax, not a win.** The costliest flawless run
  spent $4.58 of $13.27 on 7 craftsman write-dispatches; the cheapest
  delegated zero. A per-file *size cap* that routes writes to a craftsman
  was removed entirely: a PreToolUse hook fires *after* the model already
  spent the output tokens, so denying only makes the craftsman regenerate
  the same bytes — pure double-billing.

- **Full ceremony on greenfield is pure tax.** +78% cost, 2.3× wall,
  identical 29/29 quality vs plain — until gated (see The numbers).

---

## Assumptions this project got wrong

- **Opus 4.8 output is ~$26/M, not the $75/M list price** the first
  analytical crossover used — the estimate was 3× too high. The empirical
  run overturned it, which moved the write-delegation crossover from
  ~275 lines to ~1,800 (cold). This is the reason the economics were
  measured, not reasoned: the reasoning was confidently wrong.

- **Recursive nullius is not a general cost win.** Predicted ~600-line
  crossover; on a *small* repo it was neutral (absorption tax $0.41 →
  $0.38 — the machinery's coordination overhead ≈ the haiku saving). It
  only pays when absorption is genuinely large. The benefit is quality
  and scale, not blanket cheapness.

- **The plugin's marketplace-install path was never exercised** until it
  was dogfooded, and it hid four real bugs a settings-injected test could
  not see: a duplicate-hooks manifest that blocked all hook loading; an
  `agent_type` namespace mismatch that left the craftsman ungoverned; a
  matcher that omitted `mcp__*` so the MCP gate was dead code; and a
  `grep -rn` regex that let the commonest recursive grep slip the
  wide-search deny. All fixed, all pinned by tests.

---

## Standing debts

- **Nearly every plugin-era and economics number is n=1.** Replication is
  the largest open item. The brownfield baseline is small-n, high-variance.
- **Lens generalization is unmeasured** on a disjoint brownfield
  defect set — the lenses were derived and validated on one task.
- **Interactive dynamics are unmeasured.** All benchmark evidence is
  headless single-mandate runs; the co-pilot and gap-check doctrine are
  reasoned and dogfooded, not benchmarked.
- Preregistered experiments (predictions written before the runs, so the
  runs can embarrass them): [`benchmarks/NEXT.md`](benchmarks/NEXT.md).

The lab notebook — dated, chronological, including the dead ends — is in
[`benchmarks/`](benchmarks/).
