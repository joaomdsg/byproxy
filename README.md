# byproxy

You act by proxy. You never touch the code directly.

## The problem nobody wants to name

Every AI coding tool in 2026 is obsessed with context engineering — curating what the model sees so it produces better output. Embeddings, RAG pipelines, dynamic rule loading, memory systems. All of it answers one question: *what should the model see?*

Two questions go unanswered. **Who should do the work?** — because the expensive model burns tokens reading files it could have delegated. And **what survives between steps?** — because chat history evaporates and the model re-derives what it already knew.

byproxy answers both. One orchestrator reasons and decides; it never reads a file or runs a command itself. Cheap explorer agents are its eyes. A single builder is its hands. Everything they exchange is a terse, schema-bound telegraph — and that telegraph log is the system's only memory.

## The architecture

This is the **control plane / data plane** split from networking, applied to coding agents. The orchestrator is the control plane: it holds no payload, it decides where things go. The agents are the data plane: they move the bytes.

- **You (the orchestrator).** Expensive model. Reason over telegraphs, decide, dispatch. Never touch ground truth. Doubt a load-bearing fact → dispatch a second explorer and compare.
- **Explorers.** Haiku-class, read-only, disposable. One narrow question each, then they cease to exist. They report facts and quote machine output verbatim — they never judge. The cheapest model in the loop must never decide what information survives.
- **Builder.** One Sonnet-class agent, serial, the only writer. Executes a precise spec with a mechanical exit condition, and escalates the moment a deterministic trigger fires rather than circling.

**Why it's cheaper.** Raw file contents in the orchestrator's context cost quadratically on the most expensive model. Telegraphs are ~300–500 tokens and grow linearly on a cache-friendly append-only prefix. Diagnosis-by-explorer is ≈20× cheaper than diagnosis-by-reading. Compress the channels; never compress the scratchpad.

## TGS — the telegraph protocol

All inter-agent messages use **TGS** (Telegraphic-Grok-Schema): labeled fields, no prose, grok-dense inside fields, machine output quoted verbatim, absence reported explicitly (`UNKNOWN:` is mandatory). A dispatch to the builder looks like:

```
TASK: fix off-by-one token count lex.go:31
SCOPE: lex.go only
CONTEXT: comment lex.go:29 claims intentional skip — preserve intent or update it
DONE-WHEN: TestParseHeader + TestComment green
ESCALATE-IF: 2 consecutive fail runs | fix requires touching parse.go | diff >30 lines
```

Full spec: [`references/tgs-spec.md`](.claude/skills/byproxy/references/tgs-spec.md).

## The builder's discipline: RYGB

The builder doesn't write code freehand. It runs the **Red → Yellow → Green → Blue** TDD cycle, one full cycle per logical unit:

- 🔴 **Red** — write the failing test; confirm it fails for the right reason.
- 🟡 **Yellow** — an explorer critiques the test (trivial-pass? missing edge cases? wrong failure?). The orchestrator judges the findings and redirects.
- 🟢 **Green** — minimal implementation to pass; nothing more.
- 🔵 **Blue** — an explorer audits coverage and correctness. The orchestrator classifies each finding as pure cleanup (redirect the builder) or a new cycle (queue it in the telegraph log). Never interleave.

Yellow and Blue are read-only *findings*; the verdict is always the orchestrator's. That is byproxy's core rule applied to TDD: the cheapest model reports, the expensive model decides.

## What's in this repo

```
.claude/skills/byproxy/
  SKILL.md                 # the orchestrator's operating instructions
  references/tgs-spec.md   # the TGS message protocol
  templates/
    explorer.md            # read-only subagent system prompt
    builder.md             # single-writer subagent system prompt
template/
  CONVENTIONS.md           # questions that produce reasoning, not bare rules —
                           # feeds project facts into dispatch CONTEXT fields
```

## Getting started

byproxy is a **Claude Code skill** (Claude-specific for now — it relies on subagent dispatch).

```sh
git clone https://github.com/joaomdsg/byproxy.git
ln -s "$(pwd)/byproxy/.claude/skills/byproxy" ~/.claude/skills/byproxy
```

Then in any project, invoke `/byproxy` (or just describe a multi-agent coding task) and hand it the work.

## The philosophy in one sentence

Reason from above, delegate the reading down, let one hand do the writing, and keep the only record you trust in the telegraph log — because the expensive model should spend its tokens thinking, not fetching.
