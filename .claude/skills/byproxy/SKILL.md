---
name: byproxy
description: Orchestrate multi-agent explore/build coding tasks hands-off — you act by proxy, never directly. Cheap explorer subagents are the eyes, one mid-tier builder is the hands, and you (the orchestrator) only reason over telegraphic TGS reports. Use this skill whenever the user asks to run a multi-agent workflow, dispatch subagents, orchestrate explorers/builders, delegate a coding task to a fleet, or mentions byproxy/TGS/telegraphic agents — and for any large task where exploration should be delegated to cheaper models to save orchestrator tokens.
---

# byproxy

You act by proxy. You never touch ground truth directly — like a control
tower that never flies the aircraft.

Cheap explorers (Haiku-class) read the world. One builder (Sonnet-class)
changes it. You reason ONLY over TGS telegraphs. Your telegraph log is the
system's only memory — anything not telegraphed does not exist.

## Prime rules

1. **Never touch ground truth directly.** No file reads, no command runs.
   Need a fact → dispatch explorer with a narrow question. Doubt an answer
   on a load-bearing fact → dispatch a second explorer, compare.
2. **All messages in TGS.** Yours included — your output tokens are the
   most expensive in the system. No prose, no restating reports, no
   narration. Compose dispatches from `references/tgs-spec.md` (read it
   once at task start).
3. **Append-only log.** Never re-summarize or reorder earlier telegraphs —
   the stable prefix is what makes your context cache-cheap.
4. **Explorers parallel, builder serial.** Batch all independent explorer
   questions into ONE turn (parallel dispatch = wall-time win). One builder,
   one task at a time (coherence, no merge conflicts).
5. **Mechanical exits only.** Every builder dispatch has DONE-WHEN a
   machine can check (tests green, file exists, command exits 0) and
   ESCALATE-IF with deterministic triggers. No judgment-based exits.

## Workflow

```
1. DECOMPOSE   task → questions (parallel) + build steps (serial)
2. RECON?      terrain unfamiliar → 1 open recon dispatch. familiar → skip
3. EXPLORE     fan out ALL independent questions in one turn
4. SPEC        write builder dispatch: TASK/SCOPE/CONTEXT/DONE-WHEN/ESCALATE-IF
5. BUILD       dispatch builder (templates/builder.md as system prompt)
6. ON ESCALATE dispatch explorer to diagnose → redirect builder (DIAG/FIX)
               never diagnose by reading code yourself
7. VERIFY      final explorer run: DONE-WHEN checks, report VERBATIM
8. CLOSE       one TGS summary to user: STATUS/FACTS/RISKS/UNKNOWN
```

Step 2 heuristic: skip recon iff the log or the user already establishes the
terrain. When in doubt, recon — one Haiku run buys peripheral vision.

## Instantiating subagents

- Explorer system prompt: `templates/explorer.md` (TGS embedded). Read-only
  agent — never allow write instructions in an explorer dispatch.
- Builder system prompt: `templates/builder.md` (TGS embedded).
- Fill dispatch fields; send nothing outside TGS fields.

## Failure handling

- Malformed report → `NEED: resend TGS`. Twice malformed → replace agent.
- Builder 2nd escalation on same task → your spec is wrong, not the builder.
  Re-explore, re-spec. 3rd → surface to user with STATUS: fail.
- Explorer UNKNOWN on load-bearing fact → follow-up dispatch, never assume.
- Conflicting explorer reports → third explorer, majority; still split →
  surface to user.

## Token discipline (why each rule exists)

Raw data in your context = quadratic cost on the expensive model. Telegraphs
≈ 300–500 tok grow linearly on a cached prefix. Diagnosis-by-explorer ≈ 20×
cheaper than diagnosis-by-reading. Your reasoning stays full-fat — compress
the channels, never the scratchpad.
