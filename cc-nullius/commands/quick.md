---
description: Toggle QUICK mode — diet-lite for trivial day-to-day tasks (on|off|status).
---

Manage nullius QUICK mode. Argument: $ARGUMENTS (on | off | status; default: on).

QUICK mode is the triviality tier: for small tasks with nothing to hunt
(a rename, a config poke, a quick question), the full ceremony and most
governor gates are friction, not rigor. In QUICK mode the governor
passes sweeps, whole reads, MCP calls, edits of any size, and heavy Bash;
only the tail-bounding rewrite stays (context stays lean either way).
The marker auto-expires after 4 hours (`NULLIUS_QUICK_TTL_H`), so a
forgotten toggle cannot silently unstarve tomorrow's defect hunt.

- **on**: `touch .nullius-quick  #nullius:ok` in the project root. Confirm
  to the user, and note the auto-expiry.
- **off**: `rm -f .nullius-quick  #nullius:ok`.
- **status**: report whether `.nullius-quick` exists and, from its mtime,
  how long until it expires.

QUICK governs the GOVERNOR, not your judgment: if a "trivial" task turns
out to touch shared state, fault paths, or exported surface, turn QUICK
off and run the hunt — the gate rules, not the label the task arrived
with.
