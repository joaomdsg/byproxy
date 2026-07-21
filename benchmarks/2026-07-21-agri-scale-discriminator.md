# 2026-07-21 — the scale discriminator: a large SPA→Via port, three ways

The `port-todo` family (now removed) never separated a solo model from a
delegating one: the source was small and fully specified, so fable-low held
it in context and nailed every trap solo. `agri-alert-via-port` restores the
**scale lever** — the port source is a real multi-screen SPA (many components,
two databases, auth/HMAC, maps, four grids, SMS) too large to hold in a
64k-capped context, so a solo arm must skim or delegate. The observable API is
a generic, synthetic [`CONTRACT.md`](harness/tasks/agri-alert-via-port/CONTRACT.md);
the *depth* lives in the mounted (private, never-committed) source.

## What ran

Three arms, all **fable-5 / effort low**, same 64k cap, same 135-file source
mounted (`AGRI_SOURCE_DIR`), scored by the same black-box 24-test oracle.

- **plain** — one solo `claude -p`, no plugin. Pays all context traffic itself.
- **recursive** — a fable-low **leader** (diet-governor-provisioned, scouts for
  bulk) authors a thin brief and delegates the whole build to a nested
  **sonnet-low craftsman** via `nullius-build`.
- **nullius-solo** — the same fable-low leader under the same diet governor +
  haiku scouts, but it **writes every Go file itself** — no nested builder.
  Isolates the method from the recursion.

## Result

| arm | score | total cost | turns | wall | breakdown |
|-----|-------|-----------|-------|------|-----------|
| plain        | 22/24 | **$8.95** | 55    | 1029s | fable $8.95 + haiku $0.00 |
| recursive    | 22/24 | **$4.88** | 7+44  | 1059s | leader $1.26 + craftsman $3.62 |
| nullius-solo | 22/24 | **$6.34** | 29    | 806s  | fable $5.72 + haiku $0.62 |

Every arm lands the **same 22/24**, failing the **same two** tests: T1
(session cookie missing `HttpOnly`) and T7 (contact `reference`/`name` not
HTML-escaped). These are contract-detail bugs, not a capacity gap — no arm's
extra spend bought a 23rd or 24th test.

## Read

- **Quality is a flat tie.** With the scale lever engaged, the delegating arms
  neither gained nor lost quality vs the solo arm — they matched it.
- **Cost ranks recursive < nullius-solo < plain.** The nullius method *alone*
  (governor keeps the leader off bulk, haiku eats it for $0.62) cuts plain's
  bill **~29%** at equal quality. Adding recursion cuts **~45%**: the fable
  leader pays just $1.26 for judgment and the sonnet craftsman absorbs the
  write-bulk at $3.62.
- **nullius-solo was fastest** (806s) and leanest in turns (29 vs plain's 55) —
  the diet's real product is fewer, cheaper turns, not more of them.
- A contract-only run (no source mounted) of nullius-solo scored the same
  22/24 at $5.37 — source added ~$1, as expected for the heavier input.

## Calibration validated

The oracle was built and frozen (`FROZEN.sha256`, 18 files) *before* any full
passing reference existed — the deferred-reference risk. It paid off: all three
independent arms converge on the identical 22/24 with the identical two
failures, and both failures are **real port gaps**, not oracle false-negatives.
No assertion needed tuning.

## Caveat

n=1 per arm — a cost-ranking signal, not a variance-bounded claim. The two
shared misses cap the ceiling at 22/24; a targeted follow-up (HttpOnly + HTML
escaping) would test whether any arm reaches 24/24 and whether the tier
separates there.
