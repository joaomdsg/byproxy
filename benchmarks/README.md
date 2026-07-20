# benchmarks — the lab notebook

Chronological record of every benchmark, dead ends included. Each dated
writeup states what was asked, what ran, and what it showed — several
exist to record a *refutation*. Consolidated results and caveats live in
the top-level [`FINDINGS.md`](../FINDINGS.md); this is the primary record
behind them.

- [`harness/`](harness/) — the runner (`run.sh`): headless grid, one
  worktree per rep, pinned container, per-model cost, independent scoring.
  See [`harness/README.md`](harness/README.md).
- [`NEXT.md`](NEXT.md) — preregistered experiments: predictions written
  *before* the runs, so the runs can embarrass them.

## The notebook, in order

| # | date | what it measured |
|---|---|---|
| 1 | 2026-07-08 | [`SliceOps.RemoveAt`](2026-07-08-via-removeat.md) — an S-size task |
| 2 | 2026-07-08 | [chunked resumable uploads](2026-07-08-via-chunked-uploads.md) — an L-size task |
| 3 | 2026-07-08 | [headless, unbiased, n=3 per arm](2026-07-08-headless-p34.md) |
| 4 | 2026-07-09 | [four-arm grid on a discriminating task](2026-07-09-headless-p24-grid.md) |
| 5 | 2026-07-10 | [a quality task — the guard layer's turn to be refuted](2026-07-10-vialite-todo-sonnet.md) |
| 6 | 2026-07-10 | [v4 guard layer on fable, cost attribution, rigor audit](2026-07-10-v4-fable-cost-and-rigor.md) |
| 7 | 2026-07-13 | [the sonnet control plane refuted twice, and fable-lean found](2026-07-13-sonnet-refuted-fable-lean.md) |
| 8 | 2026-07-14 | [the effort ladder, and craft parity once the judge can see the tests](2026-07-14-effort-ladder-craft-parity.md) |

Later work (the cc-nullius plugin, the terrain gate, prevention-vs-
compaction, the delegation-crossover economics, and the recursive
`nullius-build` measurement) is recorded in [`FINDINGS.md`](../FINDINGS.md)
and [`NEXT.md`](NEXT.md).
