# nullius-build brief — the leader→builder handoff

The measured failure this shape prevents: **value-passing.** A leader
compressed reachable code (the reference tree, sitting right there in the
builder's working dir) into prose verdicts — "stub the grid / stub SMS /
stub the map" — and the reference-blind builder obeyed a summary it could
not check, shipping a worse port than a solo run under the same mandate.

The fix the prior art converges on (Anthropic: workers do their own
retrieval; Manus: filesystem-as-memory / lazy fetch; A2A: pass a pointer,
worker fetches; Cognition: share context, don't compress it):

> **When the source is reachable by the builder, do NOT summarize it.**
> Hand pointers; let the builder read the actual code for all depth. The
> brief carries no build-vs-stub verdicts — that call is made against the
> source, by whoever can see it.

---

## DEFAULT: the THIN brief (use whenever the builder can reach the source)

Three parts, and nothing that pre-decides depth:

### 1. INTENT + acceptance bar
What is being built and why (2–4 lines), ending with the concrete,
checkable definition of done the close-out enforces (e.g. "`go build ./...`
and `go test ./...` pass; every surface in TERRAIN exists as real code, or
as a stub whose unreachable dependency is named").

### 2. CONTRACT  (verbatim)
The target API / framework / interface to build against, and the module/
package layout. Copy any framework guidance in verbatim — the builder has
no other source for it. This is the one thing you *do* inline, because it
is not "reachable source" the builder can go read.

### 3. TERRAIN  (a `file:line` pointer-map — never pasted code, never a verdict)
What exists to port, as pointers the builder will open itself:
- pages / entrypoints — `path:line`
- components / units — `path:line`
- routes / handlers — `path:line`
- data models / types — `path:line`
- hard-logic hotspots (algorithms, computations, protocols, integrations)
  — `path:line` **only**; do NOT annotate them build-or-stub.

### The DEPTH RULE (state it; agents mis-judge effort without one)
> Implement every hotspot for REAL. STUB only what depends on something
> genuinely unreachable in the build env (an external service that won't
> resolve, real infra/DB with no credentials). A stub with no named,
> verified blocker is a defect.

---

## FALLBACK: the HOTSPOT LEDGER (only when the source is NOT reachable)

If the builder cannot open the source (not mounted, a different repo, an
external system), you must pass the specifics — but as **pointers-plus-
defaults the builder verifies, never verdicts it obeys.** Add a table:

| hotspot | source `file:line` (or best description) | DEFAULT | reason (required iff STUB) |
|---|---|---|---|
| sector-grid computation | `utils/useSectorList.ts:12` | **BUILD** | — |
| THI grid GeoJSON | `server/api/thigrid.ts:1` | **BUILD** | — |
| SMS broadcast | `server/api/send_sms_to_all.ts:8` | **STUB** | posts to `sapc-alert_service:3000`, unreachable in build env |
| MySQL persistence | `server/repo/mysql.ts:1` | **STUB** | needs a live DB; no DSN in build env |

- **DEFAULT = BUILD** unless the reason names a real external/infra blocker
  (same DEPTH RULE as above; "stub for breadth/time" is forbidden).
- The builder re-adjudicates every row against whatever source it *can*
  reach and records overrides under ASSUMED.

---

## CLOSE  (both shapes)
The exit checks the builder's close-out scout runs FROM CLEAN and reports
verbatim: build + vet + tests + the project's linters (`-race` where
concurrency is touched). Self-reported "it compiles" is never the record;
an empty/0-byte file or missing package declaration is a broken build.
