---
description: Toggle the nullius diet governor for this project (on|off|status).
---

Manage the nullius diet governor. Argument: $ARGUMENTS (on | off | status; default status).

- **off**: create the marker file `.nullius-off` in the project root
  (`touch .nullius-off  #nullius:ok`). The governor allows everything
  while it exists. Warn the user the orchestrator is now unstarved.
- **on**: remove `.nullius-off` (`rm -f .nullius-off  #nullius:ok`).
- **status**: report whether `.nullius-off` exists or `NULLIUS_OFF=1` is
  set, plus current `NULLIUS_MAX_READ` / `NULLIUS_TAIL_LINES` (defaults
  250 / 30).
