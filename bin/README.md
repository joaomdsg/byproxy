# nullius daily driver

The scripted, recursion-enabled way to run nullius on real SWE work. Wraps
your live Claude Code session (it doesn't replace it) and adds the one thing
the plugin structurally can't: **recursion** — handing a large build to a
nested nullius session so bulk absorption stays in cheap throwaway contexts.

## Install

```sh
ln -sf "$(pwd)/bin/nullius"       ~/.local/bin/nullius
ln -sf "$(pwd)/bin/nullius-build" ~/.local/bin/nullius-build
# and the plugin (the doctrine + governor + agents), once:
claude plugin marketplace add "$(pwd)/cc-nullius" && claude plugin install nullius@nullius-local
```

## Daily use

```sh
cd ~/some/repo
nullius            # drops you into a nullius-wired co-pilot session
```

You stay in the loop. Invoke `/nullius` on any nontrivial task — the doctrine
runs the two-turn hunt (terrain → gate → gap-check → lensed hunt), you rule and
fix, and it closes with a scout record. The diet governor keeps your thread
lean throughout.

When a task needs a **large, self-contained build**, don't write it in your
(expensive) thread and don't cold-dispatch it — delegate to a nested nullius
session:

```sh
nullius-build "Port the pages/ and components/ under ./reference to Go+via.
  Contract: <the target API>. Absorb the Vue specifics via scouts." ./out
```

That runs a sonnet craftsman that spins up its OWN haiku scouts to absorb the
code, so neither your thread nor the craftsman's holds the bulk. It prints the
craftsman's STATUS/FACTS/RISKS report plus a cost line. Give it a **lean brief**
(intent + contract + pointers to what to read) — not the code itself; "guide
lean, don't duplicate."

## When recursion pays (measured 2026-07-20)

| build | in-thread / cold sonnet | nullius-build (nested) |
|---|---|---|
| small change (~25–130 lines) | cheaper — just write it | neutral, skip it |
| large-absorption build (154k-token port) | plain cold sonnet **$17.19, timed out** | **$5.54, completed** |

The win scales with how much code must be understood: the nested session's
sonnet context stayed ~5× leaner (7.8M vs 41.2M cache-reads) because haiku held
the bulk. Neutral on small work; decisive on large.

## Commands

- `nullius [dir]` — launch the co-pilot (verifies wiring, then `claude`).
- `nullius --check` — verify plugin + PATH + git wiring, don't launch.
- `nullius-build "<brief>" [dir]` — delegate a large build to a nested session.
  `@file` / `-` read the brief from a file / stdin. `--model`, `--max-read`,
  `--dry-run`.
