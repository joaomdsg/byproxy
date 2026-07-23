# go-nullius — deterministic orchestrator design (fable/xhigh advisor, 2026-07-23)

Design for a pure-Go deterministic driver for weak local models. Reasoned from the
advisor context pack (advisor-pack-v2.md). Verbatim advisor output below.

> **AUTHORITY:** the verbatim advisor body records the *original* reasoning; where it
> and the **AMENDMENTS** section (end of file) disagree, AMENDMENTS win. In particular
> the body's `go/ast`/`go/parser`/`go/types` substrate is **superseded** by tree-sitter
> (see Amendment 1 — settled). Read AMENDMENTS first for current decisions.

---

**Premise disputed:** on a weak model the model should never *find* anything. Go finds;
the model only *discriminates between two concrete alternatives it is shown*. Every
open-ended judgment ("is this a defect?") is where it over-flags. Every contract should be
a forced choice over evidence the orchestrator already extracted. The "leader" as a smart
conversational entity dissolves — what survives is three narrow stateless smart calls
(lens selection, plan authoring, report writing). No leader loop → no leader context to diet.

## Q1. STATE MACHINE — 9 phases, linear + one bounded loop

    Index → Orient → Enumerate → Judge → Corroborate → Plan → Drain → Audit → [defer] Close
    (Go)   (smart×1)  (go/ast)   (fast×N) (Go+fast+esc) (smart×k)(Go+craft)(Go)   (Go,+smart×1)

- **Index** (Go): parse whole module (go/parser+go/types) → Index{Files,Funcs,Types,CallGraph,SharedState}. → Orient.
- **Orient** (smart×1): task + mechanical terrain digest → typed `{lenses:[]LensID (from FIXED library), focus_pkgs, risk_note}`. Model picks WHICH analyzers, cannot invent. Invalid JSON after 2 retries → run ALL lenses. → Enumerate.
- **Enumerate** (Go): run selected AST lenses → `Candidate{ID,Lens,File,Line,Fn,DiskSnippet,ContextSnippet,MechanicalFacts}`. Snippet read from disk by Go → fabrication impossible. Cap 30/lens by mechanical severity. candidates>0→Judge; ==0→Close.
- **Judge** (fast×N parallel, per candidate): forced-choice → `{answer: DEFECT|CORRECT|CANT_TELL, decisive_line: int (∈ offered), because: ≤160c}`. Line out of range → auto CANT_TELL. → Corroborate.
- **Corroborate** (Go+fast): three filters on every DEFECT: (1) decisive-line validity check (repurposed quote-gate — line must contain the lens's expected mechanism token); (2) pair-discrimination for same lens+shape (D2 vs FP): one fast call showing BOTH snippets — "identical shape; which is scope-safe, name the arg carrying scope + where it comes from"; identical undifferentiated answers → both CANT_TELL → escalate smart; (3) refuter `{stands:bool, refuting_line:int|null}`. Smart escalation ≤3/run. confirmed>0→Plan; ==0→Audit.
- **Plan** (smart, 1/defect): candidate + corroboration + enclosing func text → `{target,intent≤300,test_name,test_sketch≤600,blast_radius}`. No writes. → Drain.
- **Drain** (Go+craftsman, AS BUILT): snapshot→craftsman writes→empty-diff=FAIL→`go build ./...`→touched-pkg -race→fail⇒revert+1 retry⇒2nd fail⇒mark FAILED, keep revert, advance. → Audit.
- **Audit** (Go): re-run each fixed defect's lens over CHANGED FILES ONLY + exported-surface diff. New candidate in changed files → back to Judge for THOSE ONLY, auditRound++.
- **Close**: unconditionally reached — the return path, also via `defer` on error/panic/cancel.

**Deterministic termination (not a turn cap):** monotone shrinking set — auditRound≤2 AND re-judge only candidates in Drain-touched files AND any Finding.Key already ruled is auto-skipped (key set only grows). Terminates in ≤2 extra rounds regardless of model behavior; round-3 findings → UNRESOLVED in report. Termination is a property of the data structure.

## Q2. MODEL CONTRACT
- Transport: one request = one JSON answer. GBNF grammar-constrained (llama.cpp supports it — strongest local "forbid anything else"). NO tools in any judgment request. MaxTokens per schema (verdict 400, plan 1500, report 3000). Strict decode (DisallowUnknownFields), 2 retries w/ parse error appended, then schema-default fallback (CANT_TELL / all-lenses / templated report). Model physically cannot act — no tools; all side effects are Go's.
    type Caller interface { Ask(ctx, tier Tier, prompt string, grammar GBNF, out any) error }
- **Judge** (fast): input = 2-sentence lens desc + numbered-line snippet + MechanicalFacts ("arg 2 line 41 is literal nil; binds to param scope *Session per go/types") + 2-4 fixed contrastive exemplars (generic defect/correct shapes). Asymmetric prior: "enumerator over-generates; most are CORRECT; DEFECT requires naming the exact failing line."
- **PairDiscriminate** (fast): both snippets A/B + scope question → `{a:SAFE|UNSAFE, b:..., a_scope_source_line, b_scope_source_line}`.
- **Refute** (fast): `{stands, refuting_line}`.
- **Orient/Plan/Report** (smart): typed as above; Report is DECORATIVE — Go emits templated report from ledger on any failure. Never load-bearing.
- **Craftsman**: stays a subprocess (edit-compile-fix is genuinely iterative), one-step mandate (step + enclosing file), RESULT: DONE|FAILED, verified by build regardless (never trusted).

## Q3. MACHINE-FINDS / MODEL-JUDGES (~90% of hunt is AST-enumerable)
Fully AST-enumerable (model only classifies):
- **Tautology (D1):** constant-fold boolean exprs under go/types — `len(x)>=0`, `x==x`, `uint<0`, `||true-arm`. D1 caught purely mechanically; verdict pre-seeded DEFECT, model asked only to confirm decisive line, disagreement escalates (never dismisses).
- **Nil-scope-arg (D2/FP):** enumerate calls passing nil/zero to a pointer param where MOST sites pass a derived value (NO name-matching needed). Both D2 AND FP enumerate — correct: enumeration is RECALL, judgment is PRECISION.
- Unlocked-shared-state (callgraph: pkg var/field written from goroutine/handler w/o dominating Lock), cleared-before-flush (statement-order), ignored-error, shadowed-return, defer-in-loop — all mechanical, over-approximate is fine.

Needs semantic judgment: is a nil arg intentional (broadcast-all) vs leak — exactly D2 vs FP. go/types supplies the discriminating facts (statesess nil'd param = `scope *Session` per-session broadcast; stateapp nil'd = `app *App` app-wide) + callee doc + sibling call sites. Model answers a forced contrastive question, not raw shape-match.

Designs out all three at once: over-flag (model can't grow candidate set; DEFECT survives 3 independent filters); fabrication (Go supplies quote — nothing to fabricate); D2-vs-FP (recall mechanical, precision = easiest forced choice).

## Q4. TERMINATION & CLOSE
Close = the run's exit, via defer off `context.Background()` (runs even on cancel/panic):
    defer func(){ if r:=recover();r!=nil{err=...}; rep=o.mechanicalClose(context.Background(), err) }()
mechanicalClose pure Go: gofmt -l, go vet ./..., go build ./..., go test ./... -count=1 (touched -race), git diff --stat, git status --porcelain, exported-surface diff vs Index snapshot, ledger dump w/ dispositions, verbatim exit codes. Then optional smart prose call; templated fallback. Zero turns from any budget, own 10-min wall clock → cannot be starved. Mode 4 impossible: even a mid-Drain crash produces a close record; Drain never leaves a broken build resident.
Budgets PER-PHASE not global: Budget{ModelCalls, WallClock}; exhaustion → record PHASE_TRUNCATED and ADVANCE, never abort.

## Q5. WEAK-MODEL ROBUSTNESS (transport/orchestrator, all Go)
1. Output ceiling: switch to STREAMING; hard-kill past MaxTokens (belt) + per-schema byte ceiling (suspenders).
2. Repetition kill: rolling 256-byte window, best-repeated-substring ratio >0.9 for 3 windows → cancel, retry once at temp+0.3, then fallback. INPUT guard: prompt assembly returns (prompt,tokens,error), errors at the wall (the 2.86M blowup was unbounded prompt assembly).
3. Context accountant: fit(parts,budget) per-call token count; budget per class (judge 6k, plan 16k, report 24k); structured truncation (drop exemplars→context lines→NEVER candidate snippet). Stateless calls → no growing conversation.
4. Schema enforcement: GBNF + strict-decode + 2 retries + typed fallback (degrade toward doing LESS, safely).
5. Slot hygiene: sem(4); per-request wall clock (fast 120s, smart 300s); craftsman keeps idle watchdog + 32KB cap + 12-turn cap w/ revert-on-cap.
6. No model-reachable side effects: judge/refute/plan/report carry no tools; craftsman only touches disk, in a snapshot, verified by build. Modes 1&3 unrepresentable (no leader loop, no optional lens, no tool to choose).

## Q6. KEEP vs SCRAP — replace, don't wrap, keep the seams
Hybrid worth having: "deterministic orchestrator + ONE agentic role (craftsman) as bounded subprocess + build flag swapping the whole orchestrator for old Loop on frontier."
- **Scrap** (local path): leader agent loop, all 3 deny-and-steer gates (no target once model has no tools), Unfinished() nudge, Editor sweep (no resident leader ctx), MaxTurns global cap, HuntTool as model-driven activity.
- **Keep verbatim**: quote-gate (→decisive-line validator), Drain (the template for everything), Finding/Ruling/Step ledger + Finding.Key (the termination proof's data structure), scout/craftsman subprocess runner + sem(4) + watchdog, CloseTool checklist (→mechanicalClose function), V| polarity (→CORRECT/DEFECT enum).
- **Keep behind flag**: `--driver=agentic|deterministic`. Frontier=Loop.Run, local=state machine. Shared: transport, ledger, drain, close. Also the cleanest methodology A/B.
Why NOT wrap agentic inner steps for judge/orient: any inner agentic step re-imports the gate problem in miniature. Agency only pays for WRITING code (craftsman); everywhere else it was an expensive way to phrase a question Go can phrase exactly.

## BUILD ORDER (fastest path to D1+D2 caught, FP skipped, always closes)
1. **defer mechanicalClose** around whatever driver runs (½ day; kills mode 4 NOW, even before new driver — promote CloseTool checklist to a plain func).
2. **Transport hardening** (streaming + output ceiling + repetition kill + prompt-size wall) — small, protects everything, kills mode 6.
3. **Caller.Ask + strict-decode + GBNF** for Verdict/Pair/Refute — typed-call substrate.
4. **Two lenses only**: tautology (D1) + nil-scope-arg (D2/FP) as tree-sitter `.scm` queries over gotreesitter (Amendment 1; NOT go/ast) → Candidates w/ MechanicalFacts. Syntactic facts come from the query captures; semantic facts (e.g. the nil'd param's declared type — D2 `*Session` vs FP `*App`) come from a fast-agent summary of the extracted CST slice (Amendment 2), not go/types. Skip index/callgraph generality; per-file parse of focus pkgs suffices.
5. **State-machine skeleton**: Index-lite→Enumerate(2)→Judge→Corroborate(line-check+pair+refute)→reuse Plan/Drain→Audit(bounded)→Close. Hardcode lens selection (skip Orient) for first run.
6. **Acceptance run**: success = ledger shows D1 DEFECT-fixed, D2 DEFECT-fixed, FP CORRECT-ruled, build green in close record. Achievable steps 1-5, no smart calls except Plan.
7. Generalize: Orient, remaining lenses, smart escalation, audit re-entry, --driver flag + A/B.

Steps 1-2 land even if the rest is debated — they fix verified failure modes in the CURRENT architecture.

---

## AMENDMENTS (user direction, 2026-07-23)

**MULTI-LANGUAGE, not Go-specific.** The harness must be language/format-agnostic from the start; `go/ast`+`go/types` would make it Go-only. Decisions:

1. **Enumeration substrate = tree-sitter, non-CGO — SETTLED (2026-07-23).** A lens = a named tree-sitter query (`.scm`), one query-set per language; uniform CST API across languages. Syntactic lenses (tautology, nil-arg patterns) are plain query matches. **Route: `github.com/odvcencio/gotreesitter`** — a pure-Go tree-sitter (parser + full `.scm` query engine; per-language `XxxLanguage()` grammar accessors, Go/Python/JS/Rust incl.). Self-verified: 0 C files, 0 `import "C"`, builds to a statically-linked binary under `CGO_ENABLED=0`, and a smoke test parsed Go + ran `.scm` queries catching the D1 tautology (`len(x) >= 0`) and D2 nil-arg (`f(ctx, nil, key)`) shapes. The cgo bindings (smacker, tree-sitter/go-tree-sitter) and the WASM-under-`wazero` fallback are NOT needed. API shape: `grammars.GoLanguage()` → `gts.NewParser(lang).Parse(src)` → `gts.NewQuery(scm, lang).Execute(tree)` returning `[]QueryMatch{Captures:[]QueryCapture{Name string; Node *gts.Node}}`; node text = `src[n.StartByte():n.EndByte()]`; node kind = `n.Type(lang)`. (Version pinned in go.mod; deliberately not restated here — go.mod is the source of truth.)
2. **Semantic layer = fast-agent summarizer EVERYWHERE** (chosen). Tree-sitter is syntactic only; the type/binding facts go/types would give (e.g. D2's nil'd param is `scope *Session` vs FP's `app *App`) come uniformly from a fast agent reading the specific CST slice (callee sig + enclosing fn, extracted by node byte-ranges) and summarizing into a typed schema. Low-fabrication (summarizing REAL extracted text), bounded (one slice). One uniform code path, no per-language semantic providers. `SemanticProvider` iface with a go/types opt-in was REJECTED in favor of uniformity.
3. **Build scope = SAFE WINS FIRST** (chosen). Landed this session (both language-agnostic, both fix verified failure modes in the CURRENT agentic arch):
   - **defer MechanicalClose** (leader/close.go + cmd/main.go): pure-Go close-out record (detected suite + git diff/status + 0-byte find, verbatim exit codes) runs in a `defer` off context.Background() whenever no clean close-out ran → close can never be starved by the turn cap. Kills failure mode 4.
   - **Transport prompt-size wall + output default cap** (api/openai.go): refuse to SEND a request whose estimated prompt exceeds maxPromptTokens=200k (the 2.86M-token blowup was unbounded INPUT assembly); default MaxTokens=8192 when absent. Kills failure mode 6's input side. DEFERRED (needs switch off the non-streaming client): streaming + mid-generation output ceiling + repetition kill-switch.

**Progress (2026-07-23):** safe wins landed and the working tree was cleaned — the abandoned agentic prompt-gate experiment (read-gate/hunt-gate/scout-unification) was discarded back to HEAD, junk binary removed + gitignored. Tree green (gofmt/vet/build/test all pass). Substrate settled (Amendment 1).

**Next:** build the `internal/enumerate` package (build-order step 4) — gotreesitter-backed enumerator, the 2 `.scm` lenses (tautology + nil-arg), typed `Candidate` output — proven over the vialite skeleton: D1+D2 enumerate, FP enumerates too (recall is mechanical; the D2-vs-FP precision call is the fast-agent semantic step, Amendment 2). Then step 5 (state-machine skeleton) and step 6 (acceptance run).
