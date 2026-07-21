# Task: port the Agri-Alert dashboard to Via v0.7

You are porting an existing **client-rendered SPA** — an agricultural-monitoring
dashboard (a Nuxt/Vue frontend backed by a JSON API over two SQL databases) —
to a **server-rendered [Via](https://github.com/go-via/via) v0.7 application in
pure Go**. Via renders HTML on the server and drives the browser via Datastar
(SSE); no hand-written JS, no build step.

## What you are given

- **`source/`** — the full existing SPA (the behavior you are porting). It is
  large; treat it as your **depth reference**: go to it for the exact rules
  (validation order, pagination, the cross-database station→place lookup, the
  HMAC session-token scheme, the grid GeoJSON math, the map tabs). Do **not**
  try to hold it all in context at once — absorb the slices you need.
- **`CONTRACT.md`** — the **authoritative observable API** your port must
  expose. Where the source's surface strings differ (language, role names,
  error text), **the contract wins**: emit the contract's hooks and tokens
  (`manager`/`viewer`, `error:phone_invalid`, `data-tab`, the `/grid/*` routes,
  the grid feature `properties`, etc.). The source supplies depth; the contract
  supplies the surface the scorer checks.
- **`seed/`** — the deterministic synthetic data your binary loads at startup:
  `seed.sql` (two databases `agri` + `weather`), `sector_grid.csv`,
  `monitored_sectors.json`, and `feeds/{sporulation,thermal,battery}.json`.
  Read `AGRI_SEED_DIR` (default `./seed`) for the path.
- **`skeleton/`** — a compiling Via v0.7 module stub (`module agrialert`, `go`
  pinned, `require github.com/go-via/via v0.7.0`). Build your port **here**.

## What to build

A single Go binary that, on `go build ./...`, produces a server which:
1. Loads the seed at startup (any store; the scorer only observes HTTP).
2. Serves every route, page, action, and grid in `CONTRACT.md`, with the auth
   guard, roles, pagination, validation tokens, status-log-on-change, grid
   GeoJSON + legends, the thermal day selector, and the Datastar SSE actions.
3. Reads config from env: `AGRI_SESSION_SECRET`, `AGRI_MANAGER_PASSWORD`,
   `AGRI_VIEWER_PASSWORD`, `AGRI_SECURE_COOKIE`, `AGRI_SMS_URL`, `AGRI_SEED_DIR`,
   and `PORT` (listen address `:$PORT`, default from the harness).

## Rules

- **The contract is the target, the source is the truth for depth.** A behavior
  the contract underspecifies, resolve by reading the source and record the
  call. Never invent semantics the source contradicts.
- **Build for real.** Stub a piece ONLY if it depends on something genuinely
  unreachable here (there is nothing: the SMS upstream is a mock the scorer
  controls via `AGRI_SMS_URL`; everything else is in `seed/`). A `TODO` on a
  reachable feature is a defect.
- **Every asserted datum must be reachable from the initial server-rendered
  HTML**, not only via SSE — SSE is progressive enhancement.
- Close by proving a from-clean `go build ./...` and `go vet ./...` pass.

Report STATUS / FACTS / RISKS / UNKNOWN / ASSUMED.
