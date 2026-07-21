# agri-alert-via-port

Port a large, client-rendered **SPA agricultural-monitoring dashboard** to a
**server-rendered [Via](https://github.com/go-via/via) v0.7 app in pure Go**,
scored by a black-box semantic-contract oracle.

## Why this task exists

The earlier small-port family failed to separate a solo model from a delegating
one: the source was small and fully specified, so a solo model held it in
context and nailed every trap. This task restores the **scale lever** — the port source is a
real multi-screen frontend (many components, two databases, auth, maps, grids)
too large to hold in a capped context, so a solo arm must skim or delegate. The
observable contract is generic and synthetic; the *depth* lives in the mounted
source.

## Layout

| path | role | committed? |
|------|------|-----------|
| `CONTRACT.md` | authoritative observable API the port must serve | yes |
| `seed/` | synthetic startup data (2 DBs, sector grid, 3 feeds) | yes |
| `fixtures/` | data goldens derived from the seed | yes |
| `skeleton/` | compiling Via v0.7 stub the arm fills in (`module agrialert`) | yes |
| `hidden/` | the semantic-contract oracle (boots the binary, drives it) | yes |
| `prompt.md` | the arm's brief | yes |
| `score.sh` / `run.sh` / `meta.env` | scorer + two-arm runner + metadata | yes |
| `source/` | the **private** port source, mounted at run time | **no (git-ignored)** |
| `reference/` | optional local reference port | **no (git-ignored)** |

Everything committed is **synthetic and generic** — no real organization,
person, geography, phone, or sector. The private source is never committed; the
runner mounts it from `AGRI_SOURCE_DIR`.

## Run

```sh
docker build -t nullius-bench:latest benchmarks/harness      # once
export NULLIUS_CLAUDE_CODE_OAUTH_TOKEN_FILE=/path/to/token   # never printed
AGRI_SOURCE_DIR=/path/to/frontend bash run.sh all            # plain + recursive
AGRI_SOURCE_DIR=/path/to/frontend bash run.sh nullius-solo   # method, no builder
```

Arms: `plain` (solo, no plugin), `recursive` (fable leader → nested sonnet
craftsman via `nullius-build`), `nullius-solo` (fable leader + diet governor +
haiku scouts, writes the code itself).

**The port source is private and is never committed.** The runner mounts it
from `AGRI_SOURCE_DIR`; point it at your own local copy of the frontend.
Without `AGRI_SOURCE_DIR` the arms run contract-only (public-safe, but the scale
lever is not exercised). Score a built attempt directly:

```sh
bash score.sh skeleton      # -> SCORE n/24  (a bare skeleton scores 0)
```

## Scoring

`score.sh` builds the arm's app, boots it against the synthetic seed, and runs
the hidden oracle: 24 tests over auth/roles, contacts (pagination + create/edit/
delete + validation tokens + HTML-escape + id-non-reuse), stations (join +
status change), the four grids (GeoJSON shape, monitored subset, spot values),
grid auth, and the SMS broadcast (sent + failed paths). Assertions key on the
contract's generic hooks, not markup, so any valid Via realization passes. The
Datastar SSE layer is recommended craft graded by the quality judge, not the
pass/fail oracle (see `CONTRACT.md` §5).

## Result (2026-07-21, n=1/arm, source mounted)

| arm | score | total cost | turns | wall |
|-----|-------|-----------|-------|------|
| plain        | 22/24 | $8.95 | 55   | 1029s |
| recursive    | 22/24 | $4.88 | 7+44 | 1059s |
| nullius-solo | 22/24 | $6.34 | 29   | 806s  |

Flat quality tie; cost ranks recursive < nullius-solo < plain. The method alone
cuts plain ~29%, recursion ~45%, at equal quality. All three fail the same two
tests — T1 (session cookie missing `HttpOnly`) and T7 (contact fields not
HTML-escaped) — real port gaps, not oracle false-negatives. Full writeup:
[`benchmarks/2026-07-21-agri-scale-discriminator.md`](../../../2026-07-21-agri-scale-discriminator.md).

## Calibration

The oracle was frozen (`FROZEN.sha256`) *before* any complete passing reference
existed — a deliberately deferred reference. The first run validated it: three
independent arms converge on the identical 22/24 with the identical two
failures, both real gaps, so no assertion needed tuning.
