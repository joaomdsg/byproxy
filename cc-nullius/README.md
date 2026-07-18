# cc-nullius — starved-orchestrator plugin for Claude Code

The nullius methodology as a Claude Code plugin: the main-thread context
pays only for judgment; bulk absorption, lens hunts, implementation, and
verification run in cheap throwaway subagent contexts. Cumulative session
cost is quadratic in retained context — starving the orchestrator attacks
the growth term directly.

## What it ships

- **Diet governor** (`hooks/diet-governor.mjs`, PreToolUse, zero-dep node): on the main
  thread only (subagent calls, identified by `agent_id` — the reliable
  discriminator; `agent_type` can appear on the main thread in `--agent`
  sessions — pass untouched):
  - denies Grep/Glob/WebFetch/WebSearch → steers to scouts;
  - denies whole reads of files > `NULLIUS_MAX_READ` (250) lines and exact
    duplicate reads (ledger keyed on path+range+mtime, per session; deny
    reason names the narrower-range escape for post-compaction re-reads);
    binaries (extension or NUL sniff) are exempt from the line cap —
    newline bytes in a PNG are not lines;
  - denies heavy Bash (builds/tests/linters/package installs) and
    unbounded wide searches (incl. `grep -R`/`--recursive`);
  - bounds other single-line Bash with `2>&1 | tail -n 30`, preserving the
    command's exit code via `exit "${PIPESTATUS[0]}"`; commands with `#`
    comments or trailing `;` are left unrewritten/normalized (both broke
    the `{ …; }` wrapper); the rewrite carries NO permission decision, so
    the user's normal permission prompt still applies;
  - allows main-thread Edit of SOURCE only when small (≤ `NULLIUS_MAX_EDIT`,
    25 changed lines — measured: a 12-line cap routed defect-sized fixes to
    sonnet craftsmen at 2× run cost) — bigger edits steer to craftsman;
    docs/config edits
    are exempt (consistent with Write); Write allowed for
    tests (any size), scaffolding/config (not source), API skeletons
    (decl-dense: ≥1 exported decl per 30 lines), and small files
    (≤ `NULLIUS_MAX_WRITE`, 120 lines) — implementation bulk steers to
    craftsman;
  - boundary gate: edits that remove/rename exported symbols are DENIED in
    craftsman hands (design decisions escalate up) and ALLOWED on the main
    thread at any size — boundary-shaping is the orchestrator's, exempt
    from the small-edit cap;
  - craftsman tests-first gate: inside nullius-craftsman, the first
    Edit/Write must touch a test file or SOURCE edits are denied
    (scaffolding/config/docs are exempt — nothing to test).

  Known limitation: mechanical cross-file renames route >12-line call-site
  batches through craftsman and its tests-first gate — churn for a
  mechanical change. Current answer is the blunt `.nullius-off` toggle.
  - Escapes: `#nullius:ok` in a command, `.nullius-off` file, `NULLIUS_OFF=1`.
- **Agents**: `nullius-scout` (haiku — absorption, terrain maps, and the
  close-out record: full suite + linters + exported-surface diff, ≤40-line
  anchored testimony), `nullius-lens-hunter` (haiku — one lens over named
  targets, strict PRESENT/ABSENT/AMBIGUOUS grammar), `nullius-craftsman`
  (sonnet — last resort for one pinned indivisible change, tests-first,
  public-API surface frozen). There is no judge tier: verification is
  absorption; every ruling is the orchestrator's.
- **Skill** `nullius`: the orchestrator doctrine (two-turn hunt —
  terrain then targeted lenses, with the terrain ruling as a load-bearing
  GATE: quoted lens targets → FULL mode, core lenses never deselected;
  quoted absences on pure greenfield → BUILD mode, ceremony stands down
  and the leader builds under the diet alone; doubt → FULL. Capped ruled
  checklist with no line left unruled; out-of-mandate has a cost; close
  via scout record + surface diff in every mode).
- **Commands**: `/nullius:hunt`, `/nullius:close`, `/nullius:diet on|off|status`.
- **Tests**: `node --test hooks/diet-governor.test.mjs` — behavioral suite piping synthetic
  PreToolUse payloads through the governor (rewrite validity, exit-code
  preservation, no auto-approve, gates, ledger, craftsman markers).

## Lessons from the measured runs baked in

| Measured failure (benchmarks + pi-nullius runs) | Countermeasure here |
|---|---|
| Leaders ignore system-prompt "should hunt" | `/nullius:hunt` mechanizes the fan-out; skill makes it step 0 |
| 168-item checklist flood → churn, 2/6 fixes | 40-item cap, mechanically-certain ABSENTs ranked first |
| Only mechanically-decided findings got fixed | lens-hunter's strict quote-or-AMBIGUOUS grammar |
| Rubber-stamp adjudication (74/370 free dismissals) | no-line-left-unruled dispositions; out-of-mandate rulings must quote the mandate |
| Green DoD, hidden 0/6: leader deleted public `AppendToHead` | craftsman public-surface freeze + surface diff in the close-scout record |
| rep 3: softened hunt went 5/6 — sse-premature-clear silent again | two-turn hunt: terrain sharpens aim, core lenses never deselected, fault-survival unconditional |
| greenfield-ledger: full process on empty lens terrain = +78% cost, 2.3× wall, identical 29/29 quality vs plain | terrain gate: quoted absences → BUILD mode (ceremony stands down, diet stays); doubt → FULL |
| Context bloat from re-reads and command floods | duplicate-read ledger, tail-bounding, whole-read cap |
| Judgment delegated downtier capped at 3/6 | orchestrator rules on decisive lines itself; agents never decide |
| 7 craftsman dispatches = $4.58 of a $13.27 run (rep 1); published consensus: delegated writes are the tax | edit cap 12→25; craftsman demoted to last resort — intelligence fans out, writes stay home |

## Install (local)

```
/plugin marketplace add /home/jgonc/Personal/repos/nullius/cc-nullius
/plugin install nullius@nullius-local
```

Best paired with a top-tier orchestrator model (`/model`) — the whole
economics is judgment uptier, bulk downtier.
