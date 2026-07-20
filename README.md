# nullius

***Nullius in verba* — take nobody's word for it.**

You are not paying for a frontier model's intelligence. You are paying
for its attention span. The bulk of a premium coding run's cost is
context traffic — the model re-reading its own conversation every turn.
The judgment is a few dollars; the rest is rent.

nullius is a Claude Code methodology, plugin, and daily-driver built on
that one measured fact. The top-tier model works hands-on and holds
almost nothing: throwaway haiku scouts absorb every build, test, and
sweep in contexts billed once at a fraction of the price and then
discarded. Discovery runs through lensed hunts that must return **quoted
mechanisms** — the lock acquisition in the entrypoint's own body, the
scope argument at the fan-out call site — never claims, because every
measured miss in this project's history was a trusted-claim failure.
Confirmed defects are fixed test-first before close; a scout rerun,
never self-reported green, is the record.

Every number below was measured by the harness in this repo, scored
independently, and is reproducible. The failures are kept next to the
wins on purpose — see [`FINDINGS.md`](FINDINGS.md) for the full ledger,
including the refutations and the assumptions this project got wrong.

## What's in here

nullius exists in three forms, one lineage — each preserved because the
benchmarks that measured it are part of the record:

| | what | status |
|---|---|---|
| [`.claude/`](.claude/) | the **bare methodology** — the doctrine as a skill + `nullius-explorer`/`nullius-hunter` agents. The original form; the harness's pre-plugin arms still run it. | measured (benchmarks 1–7) |
| [`cc-nullius/`](cc-nullius/) | the **Claude Code plugin** — a diet-governor hook that *enforces* the starvation, the scout/lens-hunter/craftsman agents, and the doctrine as a skill. The recommended interactive form. | current · [ROADMAP](cc-nullius/ROADMAP.md) |
| [`bin/`](bin/) | the **daily driver** — `nullius` (a co-pilot launcher) and `nullius-build` (the recursion primitive: hand a large build to a nested nullius session so bulk absorption never touches your thread). | current |
| [`benchmarks/`](benchmarks/) | the **harness** (headless grid, containerized, independent scoring) + the dated research log + [preregistered next experiments](benchmarks/NEXT.md). | — |
| [`archive/byproxy-v6/`](archive/byproxy-v6/) | the **graveyard** — the ceremony-heavy architecture this project refuted, kept runnable. | refuted, on purpose |
| [`pi-nullius/`](pi-nullius/) | a port of the methodology to the `pi` agent. | scaffold, blocked on login |

## The numbers

Seeded-defect oracle task (6 planted concurrency bugs), headless harness,
independent scoring, blind disclosure judge:

| arm | cost | defects fixed | race-clean |
|---|---:|---:|---|
| plain fable, high effort | $23.34 | 6/6 | yes |
| **nullius, low effort** | **$6.17** | **6/6** | **yes** |
| plain fable, low effort (baseline, n=3) | $12.36 | 4.33/6 | 1/3 |
| nullius (benchmark 7, n=4 mean) | $6.23 | 5.75/6 | 4/4, 0 regressions |
| v6 ceremony, sonnet control plane | $20.56 | 3/6 | — |

Same quality as the honest top-tier run at **26% of its cost**; better
quality than the same-effort baseline at **half its cost**, every rep.
And the failed rows carry as much weight as the winners: contracts,
red-team critics, and cold auditors cost $12–14 of premium context per
run and never beat a plain top-tier run; a mid-tier control plane under
the full process capped at 3/6, twice. Tier is not a tuning knob.
Ceremony is not rigor.

Greenfield (lens-hostile by design: single-threaded double-entry ledger,
29 hidden conformance tests, frozen before any rep — n=1 each):

| arm | cost | hidden suite | turns |
|---|---:|---:|---:|
| plain fable, low effort | $3.98 | 29/29 | 25 |
| cc-nullius plugin, full process | $7.08 | 29/29 | 57 |
| **cc-nullius plugin, terrain-gated** | **$4.54** | **29/29** | **12** |

Full ceremony on empty lens terrain is pure tax (+78%). The fix is a
load-bearing gate: terrain scouts must *quote the absence* of lens
targets before ceremony stands down — and with it the plugin lands within
noise of plain while keeping the guarantee that saved the brownfield runs
(one softened hunt = the signature defect shipped silently, 5/6).

## The economics of delegation

Where should each token be spent? Measured, not assumed (the first
analytical pass assumed Opus output at list price and was 3× wrong —
[FINDINGS](FINDINGS.md) keeps the mistake):

- **Reads are the win, writes are the tax.** Delegating a *write* to a
  cheaper model only pays once the change is large enough to amortize the
  craftsman's absorption cost. With an Opus leader (measured output
  ~$26/M vs sonnet $15/M) the break-even is ~1,800 lines cold, ~130 lines
  if you hand lean context. Below that, write it yourself — a
  post-generation size-cap just double-bills.
- **Recursion pays on large absorption.** Hand a genuinely large build to
  a *nested* nullius session (`nullius-build`) whose own haiku scouts
  absorb the codebase. Porting a 154k-token Nuxt/Vue frontend to Go:

  | | plain cold sonnet | nested nullius (`nullius-build`) |
  |---|---:|---:|
  | cost | $17.19 | **$5.54** |
  | finished? | ❌ timed out | ✅ completed |
  | sonnet cache-reads | 41.2M | 7.8M |

  3.1× cheaper *and* it finished, because the craftsman's context stayed
  ~5× leaner — the starve-the-orchestrator thesis applied recursively.

## Daily-drive it

```sh
ln -sf "$(pwd)/bin/nullius" "$(pwd)/bin/nullius-build" ~/.local/bin/
claude plugin marketplace add "$(pwd)/cc-nullius" && claude plugin install nullius@nullius-local

cd ~/your/repo && nullius        # co-pilot session, wired with the doctrine + governor
```

Invoke `/nullius` on any nontrivial task; delegate large builds with
`nullius-build "<lean brief>" <dir>`. See [`bin/README.md`](bin/README.md).

Bare skill (no governor): symlink [`.claude/skills/nullius`](.claude/skills/nullius)
and [`.claude/agents/`](.claude/agents/) into `~/.claude/`.

## Reproduce it

The harness runs any task as a headless grid — fresh worktree per rep,
containerized toolchain, measured cost per model, scoring that never
trusts a run's self-report:

```sh
cd benchmarks/harness
CONTAINER=1 JUDGE=1 ./run.sh tasks/vialite-todo nullius   --reps 3
CONTAINER=1 JUDGE=1 ./run.sh tasks/vialite-todo plain     --reps 3
```

## Limits, stated plainly

The lenses were derived and validated on one seeded-defect task;
generalization to disjoint brownfield defect classes is unmeasured (the
greenfield task was built lens-hostile on purpose). **Nearly every
plugin-era and economics number is n=1** — replication is the standing
debt, tracked in [`FINDINGS.md`](FINDINGS.md). The brownfield baseline is
small-n and high-variance. Top-tier judgment only pays where quality
discriminates: on a mechanical task with a fully visible definition of
done, a plain mid-tier run is a third of the price and just as green.
Preregistered next experiments: [`benchmarks/NEXT.md`](benchmarks/NEXT.md).

Measure before you believe anything — including this README.
