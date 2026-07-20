# .claude — the bare methodology

The original, plugin-free form of nullius: the doctrine as a skill plus
the two absorption agents. No governor hook — nothing *enforces* the diet
here; the model follows the doctrine because it's asked to. This is what
the harness's pre-plugin arms (`nullius`, `nullius-rev2`, `fable-lean`)
run, and what benchmarks 1–7 measured.

- [`skills/nullius/SKILL.md`](skills/nullius/SKILL.md) — the doctrine.
- [`agents/nullius-explorer.md`](agents/nullius-explorer.md) — read/search/
  build/verify absorption scout (used 9× across the harness arms).
- [`agents/nullius-hunter.md`](agents/nullius-hunter.md) — the fan-in
  hunting scout for the rev2 design.

For the enforced, current form, use the plugin in
[`../cc-nullius/`](../cc-nullius/) — its diet-governor hook makes the
starvation mechanical rather than voluntary. This directory doubles as
this repo's own dogfood config.

Bare install into your user config:

```sh
ln -s "$(pwd)/skills/nullius"            ~/.claude/skills/nullius
ln -s "$(pwd)/agents/nullius-explorer.md" ~/.claude/agents/
```
