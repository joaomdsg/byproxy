# Preregistration — v5 powered rerun of byproxy vs plain on `vialite-todo`

Written **before** the v5 measurement, to bind the analysis in advance and
close the rigor threats raised against benchmarks 5–6 (see the
"Threats to validity" section of `2026-07-10-v4-fable-cost-and-rigor.md`).
Registering the design up front is the fix for threat #7-adjacent
"forking-paths" concerns: the scoring rule, sample size, and hypotheses
below are fixed here and do not change after data is seen.

## Why a rerun at all

Benchmarks 5–6 were directional (n=5, one task, one tier) and carried
eight named threats. Five are measurement-design issues that only a
re-run can close; the SKILL/scorer changes shipped alongside this doc
close the other three in the harness:

| threat | status |
|---|---|
| #7 race-regressions invisible | **fixed in scorer** — `hidden.race_check` now runs the full suite under `-race` and flags new non-catcher race failures |
| #2 audit occurrence unmeasured | **fixed in harness (v4)** — `critic/builder/auditor_dispatches` counted per row |
| #8 telemetry (last result event) | **fixed** — cost is cumulative and read from the last event; turn-based reads dropped from claims |
| #1 baseline also delegates | design — addressed by container `$HOME` (both arms plain) + reporting the plain arm's subagent counts |
| #3 direction-consistency | design — v5 reports **every** rep, no cherry-picking single reps |
| #4 `caught` = keyword grep, asymmetric | design — blind judge + symmetric report instruction (below) |
| #5 inconsistent evidentiary standard | design — one preregistered α for every comparison (below) |
| #6 confounded treatment bundle | design — ablation arms (below) |

## Hypotheses (fixed in advance)

- **H1 (quality).** byproxy v4 fixes more seeded defects than plain, on the
  three non-ceiling defects (`action-not-serialized`,
  `statesess-cross-session-leak`, `sse-premature-clear`). The three ceiling
  defects (cas, subscription, ttl) are excluded from the primary endpoint —
  at ~5/5 both arms they only add noise.
- **H2 (cost).** byproxy v4 costs no more than plain at the same tier.
- **H0.** No difference on either.

Primary endpoint: **mean defects-fixed over the 3 non-ceiling defects**
(0–3 per rep). Secondary: total fixed /6, silent /6, cost, wall, and
`race_check.data_race_detected` rate.

## Design

- **n ≥ 15 per arm** (was 5). Powers a 1-defect effect on the 3-defect
  primary endpoint out of the near-ceiling noise floor. Sequential stop:
  evaluate at n=15; if the primary-endpoint 95% CI still straddles 0 and
  |Δ| < 0.5 defect, stop and report "no detectable effect at this power"
  rather than chasing significance.
- **Containerized, both arms** (`CONTAINER=1`): fixed CLI + Go toolchain +
  throwaway `$HOME`, so *neither* arm inherits host `~/.claude`. Closes
  threat #1's "plain isn't plain on the host".
- **One α = 0.05, two-sided, every comparison.** No calling p≈0.10 "robust"
  in one place and p≈0.18 "clean" in another (threat #5). Report effect
  sizes + CIs, not just p.
- **Report every rep** in `results.jsonl`; no single-rep probes quoted as
  trend (threat #3).

### Arms (confound decomposition — threat #6)

The v4 byproxy arm is a bundle: guard layer + forced system-prompt
injection + mandated RISKS report + no-human self-answer loop. Plain gets
none of it. To attribute the effect rather than measure the bundle:

| arm | guard layer | report instruction | purpose |
|---|---|---|---|
| `plain` | no | **symmetric minimal** (see below) | baseline |
| `byproxy` | full v4 | full RISKS format | treatment |
| `plain+report` | no | full RISKS format | isolates the report-format confound |
| `byproxy-noaudit` | guard minus cold auditor | full | isolates the audit's contribution |

`plain+report` and `byproxy-noaudit` are the two ablations that matter;
add more only if the primary comparison shows an effect worth decomposing.

### Symmetric reporting + blind judge (threat #4)

- **Symmetric report instruction.** Both arms are asked, in identical
  words, to end with a short "what I changed / known-unfixed issues"
  section. Today only byproxy is told to report, yet `caught` is compared
  across arms — the disclosure metric is rigged. v5 gives both the same
  ask (implemented as a shared prompt suffix in `run.sh`, applied to every
  arm).
- **Blind report judge replaces the keyword grep** for `caught`. A third
  headless run receives each rep's report **and diff with the arm label
  stripped**, and rules per defect: disclosed / not, fixed-claim / not —
  no keyword substring. `score.sh`'s keyword `caught` stays as a cheap
  secondary, explicitly labelled lower-trust. (Blind-judge tool is the one
  piece of new tooling v5 needs; until it exists, `caught` claims are
  reported as "keyword-grep, provisional".)

## What counts as a result

- **fixed** — unchanged: each catcher passes in isolation under `-race`.
  This is the trustworthy endpoint and is not up for reinterpretation.
- **quality win** earns the word only if the primary endpoint's 95% CI
  excludes 0 at n≥15 under the α above.
- **cost win** earns the word under the same α, reported with the outlier
  kept in (no dropping the one expensive rep to make p cross a line).
- A `data_race_detected` on any rep of an arm is reported as a hard defect
  of that arm regardless of functional pass rate.

## Reproduce (once auth is set — see harness/README)

```sh
export BYPROXY_ANTHROPIC_API_KEY=sk-ant-api03-...
docker build -t byproxy-bench:latest benchmarks/harness
CONTAINER=1 ORCH_MODEL=claude-fable-5 ORCH_EFFORT=low \
  ./run.sh tasks/vialite-todo byproxy --reps 15
CONTAINER=1 PLAIN_MODEL=claude-fable-5 PLAIN_EFFORT=low \
  ./run.sh tasks/vialite-todo plain --reps 15
```

Until v5 runs, benchmarks 5–6 remain the standing (directional, contested)
record; nothing here is a result yet — it is the contract for producing one.
