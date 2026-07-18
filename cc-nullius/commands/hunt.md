---
description: Run the two-turn nullius hunt (terrain, then targeted lenses) and build the ruled checklist.
---

Run the nullius hunt phase over: $ARGUMENTS (if empty, the current task's mandate).

1. **Turn A — terrain**: dispatch 2-3 `nullius-scout` agents in ONE
   parallel message to map the mandate precisely: every mutating
   entrypoint, shared mutable state, fan-out/broadcast site,
   queue/buffer/retry state, background sweep/TTL, lock, and error path.
   Output: named target lists (path:symbol), not prose; for every lens
   with no targets, the QUOTED basis for the absence, never a bare "none".
2. **THE GATE — rule on the terrain and quote the ruling in your report**:
   FULL mode if any lens has targets, any pre-existing code is in scope,
   or the terrain is ambiguous (doubt → FULL; a softened brownfield hunt
   is the measured 5/6 failure). BUILD mode only on quoted absences —
   pure greenfield, no lens terrain: skip step 3, build hands-on under
   the diet, re-map your OWN new code when done (hunt any terrain the
   build created), then go to `/nullius:close`.
3. **Turn B — lenses** (FULL mode): dispatch `nullius-lens-hunter` agents in ONE
   parallel message, one per lens, each carrying the exact targets turn A
   named for it and the strict V| output grammar. Terrain sharpens aim,
   never coverage: a core lens (serialization, fault-survival,
   scope-confinement, wake-predicates, lost-updates, lifecycle-races,
   swallowed-errors, resource-release) runs whenever its terrain exists,
   and fault-survival runs regardless. Terrain may add lenses; it never
   deletes one.
4. Merge the verdicts into a checklist, ABSENT (quoted, mechanically
   certain) items first, then decidable AMBIGUOUS ones. Cap at 40; note
   overflow explicitly. Record it with the todo/task tools.
5. For each item, before any fix: read the decisive lines yourself
   (bounded), rule PRESENT/defect/out-of-mandate. Every item gets an
   explicit disposition — none may be silently dropped. An out-of-mandate
   ruling must quote the mandate text that excludes it.

Do not start fixing until the checklist is ruled.
