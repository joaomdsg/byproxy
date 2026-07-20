# cc-nullius — day-to-day readiness tracker

Blind-spot audit 2026-07-18, after the prevention-vs-compaction A/B.
Ranked by expected daily pain. Status: OPEN / IN-PROGRESS / DONE (date).

| # | Blind spot | Why it hurts daily | Countermeasure | Status |
|---|---|---|---|---|
| 1 | All evidence is headless single-mandate runs; interactive dynamics unmeasured (mid-task scope changes, user-driven reads, interrupted hunts) | Doctrine has no re-gating rule when the mandate shifts mid-session | Dogfood + add a "mandate shifted → re-run the gate" rule once failure modes are observed, not guessed. Interactive doctrine shipped 2026-07-19: post-terrain GAP CHECK (one batch, cap 4, three-part qualification, asked in the Turn-B dispatch turn so hunts never block), one-way-door rule (block only on irreversible choices; else PROVISIONAL), user-answers-are-testimony (intent binds, facts verified), mandate-shift re-entry (scope-changing answer → delta Turn A + re-rule gate), gap ledger + close ratification. Research-hardened 2026-07-19 (4-agent web sweep): (a) overlap VERIFIED — background dispatches run detached from the leader loop; AskUserQuestion blocks only the leader and is unavailable to subagents (Claude Code docs; Cursor 2.4 ships the same pattern; LangGraph interrupts freeze siblings — avoided); (b) reversibility = 4-layer escape analysis (read / undo-artifact→PROVISIONAL / escapes-worktree→BLOCK / unclassifiable→BLOCK), from Krakovna reachability + Anthropic auto-mode tiers + buf-breaking-class detectors; "reversibility decays" → flush at close; (c) silence splits by layer, lazy-consensus style (Apache/Rust FCP precedent): layer-2 stands-unless-objected with evaporation, layer-3 never silent-ratifies; (d) ledger salience = recite at tail of every update (goal-restating measurably cuts drift, p<0.05; edge placement is the robust lost-in-the-middle fix; close-time re-read alone is confirmation theater — 85-95% of rechecks confirm rather than correct). Remaining: pure dogfood (does it FEEL right in real sessions) | PARTIAL 2026-07-19 |
| 2 | No triviality tier: governor + ceremony engage on every task, incl. quick questions and 3-line fixes | Grep denials and steering noise on tasks with nothing to hunt; only escape is the blunt `.nullius-off` | QUICK mode: `.nullius-quick` marker (4h auto-expiry) set/cleared by `/nullius:quick`; governor drops to diet-lite (sweeps+reads allowed, Bash still tail-bounded); SKILL gets a QUICK ruling at the gate | DONE 2026-07-18 |
| 3 | Lens library is concurrency-shaped and Go-flavored; empty terrain on TS/SQL/infra work | Every hunt outside Go-like domains is wasted or BUILD-gated | Lens-derivation method → always-on lens library (bench-9 rev3 design, unbuilt); derive lenses per domain from misses | OPEN (blocked on bench-9 rev3) |
| 4 | MCP tools bypass the diet: bulk server responses land unbounded in the leader | One browser-HTML or Drive fetch can out-bloat everything the governor prevents | Main-thread MCP calls with bulk-verb names steered to scouts (subagents reach MCP via ToolSearch); `NULLIUS_MCP_OK=1` or QUICK mode to disable | DONE 2026-07-19 (gate logic 07-18 was DEAD — hooks.json matcher omitted `mcp__*`, so the hook never fired for MCP; test-drive caught it; matcher + wiring-guard test added, verified end-to-end via stats file) |
| 10 | Craftsman tests-first + boundary gates never bound under plugin install | The plugin's core safety feature was fully bypassed — craftsman edited source ungoverned | `agent_type` arrives namespaced `nullius:nullius-craftsman` under marketplace install (bare only via the harness's `.claude/agents` copy); a `===` fell through to the subagent passthrough. Fixed to regex `/(^\|:)nullius-craftsman$/`; verified end-to-end | DONE 2026-07-19 |
| 5 | No cross-session persistence: same terrain re-absorbed every session in the same repo | Session 2+ pays full absorption for unchanged code | Terrain cache `.nullius/terrain.md` (commit-stamped); Turn A validates freshness via git diff and delta-scouts only the drift; close refreshes it | DONE 2026-07-18 (doctrine) |
| 6 | Repos without test infra silently weaken tests-first + the close record | Close degrades to build+vet without saying so | Close must NAME the missing suite as a RISK (close.md step 3 + SKILL step 6) | DONE 2026-07-18 |
| 7 | No in-plugin cost/dispatch telemetry | Flying blind on the economics the whole project is about | Governor counts denies/rewrites/dispatches per session (`/tmp/nullius-stats-<session>`); `/nullius:diet status` reports them | DONE 2026-07-18 |
| 8 | Wrong REFUTED on trusted hunter testimony (measured: wake-predicate 5/6) | A quoted-but-non-covering mechanism clears a real defect | REFUTED on a core lens needs leader-read lines or a pinning test | DONE 2026-07-18 (validated same day: 6/6) |
| 9 | Mechanical cross-file renames route through craftsman + tests-first | Churn for a mechanical change | QUICK mode now covers this; a rename-aware Edit exemption would be cleaner | PARTIAL (via #2) |
| 11 | Governor always-on, but the SKILL body loads only on invocation → a session can be starved yet not run the method | "Installed" did not mean "doctrine applied"; relied on the model reaching for `/nullius` | SessionStart hook injects a ~70-token pointer (not the ~5k body) nudging the orchestrator to invoke the skill on nontrivial work; respects the off switches; verified a fresh session receives and reads it | DONE 2026-07-19 |

Measured record backing the ranks: benchmarks/NEXT.md, the repo README
table, and the memory ledger. Nothing here is speculative except #1 by
design — it is listed to force measurement before doctrine.

## Crossover measurement (2026-07-19) — write-delegation threshold

Empirical (Docker, no plugin, Opus leader) settled where delegating a
write beats writing it. **Measured Opus 4.8 output ≈ $26/M** (not the
$75/M list price the first analytical pass assumed — the run overturned
it), sonnet $15/M → output gap only ~$11/M (1.7x). Cold-absorption tax
A ≈ $0.41. Crossover `S* = A/gap`: **~1,800 lines cold-dispatch, ~130
lean-dispatch.** Direct proof: cold sonnet cost $30/M-output > Opus's
$27/M writing it one-shot at ~250 lines — cold delegation already loses.
Outcome: removed the `MAX_WRITE`/`MAX_EDIT` hook size-caps (a post-
generation deny double-bills), moved the delegate-or-write decision into
the doctrine (Division of labor), kept tests-first + boundary gates.
Suite 31 green.
