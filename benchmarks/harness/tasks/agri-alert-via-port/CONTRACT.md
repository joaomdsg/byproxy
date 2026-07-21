# Agri-Alert → Via — observable contract

Port an existing **SPA agricultural-monitoring dashboard** (client-rendered
frontend + JSON API over two SQL databases) to a **server-rendered [Via](https://github.com/go-via/via)
v0.7 application** in pure Go. Via renders HTML on the server and drives the
browser as a rendering surface via Datastar (SSE); there is no hand-written
JS and no build step.

This file is the **source of truth for the target's observable behavior**.
The scorer boots your compiled binary and drives it over HTTP; it asserts the
required structure, data, and SSE signals below — it does **not** diff your
markup byte-for-byte, so any valid Via realization passes. Where a data value
is asserted, the ground truth is the seeded database (see `seed/`) and the
captured API goldens under `fixtures/`.

> All names, geography, phone numbers, and sector identifiers in this task are
> **synthetic**. The domain (an agricultural frost/disease alert dashboard) is
> generic; nothing here corresponds to any real organization or deployment.

---

## 0. Domain & data model

Two logical databases (the port may back them with any store, but the seed and
goldens are defined against these tables):

**`agri` database**
- `farmer_contacts(id INT PK, reference TEXT, name TEXT, phone_number TEXT,
  active BOOL, created_at, updated_at)` — id from a monotonic counter, never
  reused.
- `farmer_sectors(id INT PK, name TEXT, label TEXT, farmer_contact_id INT FK)`
  — a contact has 0..3 sectors; `name` is a sector cell id, `label` a free
  string ≤25 chars.
- `station_status(station_id TEXT PK, status ENUM('active','inactive',
  'needs_maintenance'), created_at, updated_at)`.
- `station_status_log(id INT PK, station_id TEXT, old_status, new_status,
  note TEXT, created_at)` — one row appended on every status change.

**`weather` database**
- `stations(station_id TEXT PK, place TEXT, latitude_deg REAL,
  longitude_deg REAL, elevation_m REAL, responsible TEXT, ...)` — station
  metadata. `place` for the admin table is looked up here by `station_id`.

The **sector grid** is a static asset `seed/sector_grid.csv` with columns:

```
cell_id,sector_name,sector_letter,sector_number,lon1,lat1,lon2,lat2,lon3,lat3,lon4,lat4,lon5,lat5
```

Each row is a 5-vertex polygon (a closed ring; first vertex repeated as last).
`seed/monitored_sectors.json` is the subset (`["<sector_name>", ...]`) that
appears on the monitoring grids. Coordinates are GeoJSON order **[lon, lat]**.

---

## 1. Authentication & sessions

Two roles, each identified by a username string:

| role      | username string | landing page      |
|-----------|-----------------|-------------------|
| manager   | `manager`       | `/`               |
| viewer    | `viewer`        | `/map`            |

**Login token.** A session token is the string `"<username>_<timestamp>_<hmac>"`
where `<hmac>` = lowercase hex of `HMAC-SHA256(secret, "<username>_<timestamp>")`.
`secret` comes from env `AGRI_SESSION_SECRET`. Validation recomputes the HMAC over
the first two parts and constant-time-compares; **there is no expiry check** (a
structurally valid token is accepted regardless of `<timestamp>`). The validated
username is what authorizes each request.

**POST `/login`** (form-encoded, field `password`):
- `password == $AGRI_MANAGER_PASSWORD` → username `manager`.
- `password == $AGRI_VIEWER_PASSWORD` → username `viewer`.
- else → **401**, re-render login with a visible error element whose text is
  exactly `login:invalid`.
- On success: set cookie **`session_token`** = the token above, attributes
  `HttpOnly; SameSite=Strict; Max-Age=604800` (`Secure` iff `$AGRI_SECURE_COOKIE`
  truthy), plus a non-HttpOnly marker cookie `session_user=<username>`. Respond
  **303** to the role's landing page.

**GET `/login`**: 200, an HTML form `<form method="post" action="/login">` with
a password input named `password`.

**Auth guard (all routes except `/login` and static assets):**
- No/invalid `session_token` → **303** redirect to `/login`.
- Valid token but **wrong role** for the route → **403**.
- Route→role: `/`, `/admin/stations`, all `/contacts*`, `/sms*` require
  `manager`; `/map` and all `/grid/*` require `viewer`.

**POST `/logout`**: clear both cookies, **303** to `/login`.

---

## 2. Pages (server-rendered HTML)

Every page includes a summary/header identifying the role and a logout control.
Titles/labels are free; the scorer keys on `id`/`data-*` hooks named below.

### GET `/` — Contacts (manager)
Renders a paginated contacts table and an add-contact form.

- Table element `id="contacts"`; one `<tr data-contact-id="{id}">` per visible
  contact, containing the HTML-escaped `reference`, `name`, the sector `label`s,
  and an `active` indicator (`data-active="true|false"`).
- **Pagination**: PAGE_SIZE = **15**, ordered by `id` ascending. Query
  `?page=N` (default 1; `<1` or non-integer → 1). A page past the end renders
  200 with zero rows. A `data-count="{total}"` attribute carries the whole-list
  total (not the page count).
- Add form `<form method="post" action="/contacts">` with inputs `reference`,
  `name`, `phone_number`, `active`, and a sector sub-form (§3).

### GET `/admin/stations` — Stations (manager)
Table `id="stations"`, one `<tr data-station-id="{station_id}">` per visible
station with the `place` (joined from the `weather` db) and current `status`
(`data-status="..."`). Same pagination contract as contacts (PAGE_SIZE 15,
`?page=N`, `data-count`). Each row has a status control that **POSTs to
`/admin/stations/{station_id}/status`** with a form field `status` whose value
is one of `active|inactive|needs_maintenance`; on success respond **303** to
`/admin/stations` and append a `station_status_log` row when the value changed.
Unknown `station_id` → **404**; an invalid `status` value → **400**.

### GET `/map` — Monitoring (viewer)
Four tabs, selected via `?tab=sectors|sporulation|thermal|battery` (default
`sectors`), each with `data-tab="<key>"` on the active panel:
- `sectors` → the base sector polygons.
- `sporulation` → sporulation grid.
- `thermal` → thermal-stress (THI) grid, with a day selector `?day=0..6`.
- `battery` → device battery points.

Each tab's panel exposes its grid data as GeoJSON (§4) at the matching
`/grid/*` endpoint AND renders a legend (§4) so the tab is meaningful without a
JS map. Tab switching and day/refresh are Datastar actions (§5).

### GET `/explorations` — Sector selector demo (either role)
A standalone `SectorSelector` (§3) with no persistence; exercises the add/remove
/validation rules in isolation.

---

## 3. Contact & sector validation

**POST `/contacts`** (create) and **PATCH `/contacts/{id}`** (edit),
form-encoded fields `reference`, `name`, `phone_number`, `active`, plus repeated
`sector_name`/`sector_label` pairs. Validate in this order; on the **first**
failure respond **400**, re-render with a visible element whose text is exactly
one of the tokens below, and **do not mutate state**:

1. `name` empty after trim → `error:name_required`.
2. `reference` empty after trim → `error:reference_required`.
3. `phone_number` not exactly **9 digits** (`^[0-9]{9}$`) → `error:phone_invalid`.
4. any sector `label` longer than 25 runes → `error:label_toolong`.
5. more than **3** sectors → `error:too_many_sectors`.
6. duplicate `sector_name` within the submission → `error:sector_duplicate`.

On success: create stores a new contact (`active` from the checkbox, next id in
sequence) and **303** to `/`; edit replaces the mutable fields (id/created_at
unchanged) and **303** to `/`. **PATCH/DELETE with a non-integer or ≤0 id →
400** with visible text `error:bad_id`; unknown id → **404**.

**DELETE `/contacts/{id}`**: remove the contact (and its sectors), **303** to
`/`. Ids are never reused.

**SectorSelector** (used on `/` add-form and `/explorations`): add requires a
non-empty `name` (else `sector:name_required`) and non-empty `label` (else
`sector:label_required`); rejects a name already selected (`sector:duplicate`),
a label >25 runes (`sector:label_toolong`), and a 4th sector
(`sector:limit`). The tokens appear in a visible element on the failing action.

---

## 4. Grids (GeoJSON + legend)

Each grid endpoint returns `Content-Type: application/json` a GeoJSON
`FeatureCollection`. Status ids are integers; `0` means "no data". Colors are
fixed: `#e7e7e7` (none), `#5CB85C` (green), `#FFD700` (gold), `#FF2500` (red).

**GET `/grid/sectors`** — one `Feature` per grid row (all cells): `Polygon`
geometry from the CSV ring, `properties = {cell_id:int, sector_name:string}`.

**GET `/grid/sporulation`** — one `Feature` per **monitored** sector:
`Polygon`, `properties = {cell_id, sector_name, max_status_id:int}` where
`max_status_id ∈ {0,2,3,4}` sourced from the sporulation feed
(`seed/feeds/sporulation.json`: `{updated_at, grid:{sector_name→status_id}}`;
missing sector → 0). Legend: 2→green, 3→gold, 4→red, else none.

**GET `/grid/thermal?day=D`** (D in 0..6, default 0) — one `Feature` per
monitored sector: `Polygon`, `properties = {cell_id, sector_name,
day0_status_id..day6_status_id}` from `seed/feeds/thermal.json`
(`{updated_at, day0_date..day6_date, thi_data:{sector_name→{day0..6:{status_id}}}}`;
missing → 0). Response also carries the seven `dayN_date` strings (e.g. in a
top-level `properties` on the collection or a sibling field). Legend for the
selected day: 2→gold, 3→red, else none.

**GET `/grid/battery`** — one `Feature` per device: `Point` at
`[longitude_deg, latitude_deg]`, `properties = {station_id, bat_volt:number,
timestamp}` from `seed/feeds/battery.json` (`{data:[{station_id, timestamp,
bat_volt, longitude_deg, latitude_deg}]}`). Legend by voltage: `>2.7` normal
(green), `2.3–2.7` low (gold), `>0–2.3` critical (red); a device whose
`timestamp` is >1 day old is "offline" (gray) and >30 days darker gray.

---

## 5. Datastar actions (SSE) — recommended, NOT directly scored

Via drives reactive updates through Datastar: actions POST to framework-generated
`/_action/*` endpoints and patches arrive over an SSE stream
(`Content-Type: text/event-stream`). Because those URLs are generated internally,
the scorer does **not** address them black-box. Instead **the scored surface is
the classic server-rendered HTTP contract** (§§1–4, 6): every page renders its
data in the initial HTML, every mutation has a plain form route (`POST`/`PATCH`/
`DELETE` → `303`) with an observable effect on a subsequent `GET`, and the grids
are JSON. Wiring the same interactions as Datastar SSE actions on top is
recommended craft (it's what makes it a *Via* app rather than raw handlers) but
is graded by the quality judge, not the pass/fail oracle.

Consequence for tab / day / refresh: `/map?tab=...` and `/grid/thermal?day=N`
must work as plain GETs returning the correct server-rendered content, so the
oracle can drive them without JS.

## 6. SMS broadcast (manager)

**POST `/sms`** (form field `message`, non-empty after trim; else 400 with
visible `sms:empty`): the server forwards the message to the configured upstream
`$AGRI_SMS_URL` (a POST with JSON `{ "message": <text> }`). On upstream success
respond **200 or 303** (with `data-sms-status="sent"` on the resulting page/SSE);
on upstream failure or timeout, status **502** (`data-sms-status="failed"`) —
**the send is never silently dropped**. In the benchmark environment
`$AGRI_SMS_URL` points at a local mock the scorer controls, so both the sent and
failed paths are exercised deterministically (the scorer asserts the upstream
actually received the message on success, and 502 on failure).

---

## 7. What the scorer checks (summary)

Boots the port binary against the seeded DBs + feeds, then drives it:
login (both roles, wrong password, wrong-role 403, unauth redirect); contacts
list pagination + `data-count`; create/edit/delete with each validation token
and the mutate-only-on-success rule; sector selector rules; stations list +
status change writing a log row; each grid's GeoJSON shape, properties, and
monitored-subset filtering; the thermal day selector; the SSE frames for tab /
day / refresh / SMS; and the auth guard on every protected route. Data values
are checked against the seeded ground truth in `fixtures/`.
