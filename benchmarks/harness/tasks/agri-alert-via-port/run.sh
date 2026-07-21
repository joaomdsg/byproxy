#!/usr/bin/env bash
# Run the agri-alert-via-port arms INSIDE the nullius-bench docker image.
# All arms: model claude-fable-5, effort low, context-capped (CAP) TIGHT vs the
# mounted source so a solo model cannot hold the whole frontend in context —
# forcing delegation (the discriminator this task exists to test).
#
# The port SOURCE is private and is NEVER committed. Point the harness at a
# local copy with:
#   AGRI_SOURCE_DIR=/path/to/frontend  bash run.sh all
# If AGRI_SOURCE_DIR is unset, the arms run in contract-only mode (no source to
# absorb — a weaker, public-safe fallback that does not exercise the scale lever).
#
# Usage: run.sh [plain|recursive|nullius-solo|all]   (default: all)
#   plain        solo claude -p, no plugin
#   recursive    fable leader -> nested sonnet craftsman (nullius-build)
#   nullius-solo fable leader + diet governor + haiku scouts, writes code itself
#   all          plain + recursive
# Requires: docker image `nullius-bench:latest`; CLAUDE_CODE_OAUTH_TOKEN or
#   NULLIUS_CLAUDE_CODE_OAUTH_TOKEN_FILE (never printed); host `go` for scoring.
set -euo pipefail

TASK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$TASK_DIR/../../../.." && pwd)"
PLUGIN="$ROOT/cc-nullius"
RUNS="$TASK_DIR/runs"
IMAGE="nullius-bench:latest"
MODEL="claude-fable-5"; EFFORT="low"; CAP="${CAP:-64000}"
LEADER_MODEL="claude-fable-5"
CRAFT_MODEL="claude-sonnet-5"
DOCK_TIMEOUT="${DOCK_TIMEOUT:-2400}"        # outer docker wall cap (s)
BASH_TIMEOUT_MS="${BASH_TIMEOUT_MS:-1200000}" # in-container per-command cap (ms)
SOURCE_DIR="${AGRI_SOURCE_DIR:-}"   # private port source; never committed

TOKEN_FILE="${NULLIUS_CLAUDE_CODE_OAUTH_TOKEN_FILE:-}"
if [[ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ]]; then
  [[ -n "$TOKEN_FILE" && -r "$TOKEN_FILE" ]] \
    || { echo "set CLAUDE_CODE_OAUTH_TOKEN or NULLIUS_CLAUDE_CODE_OAUTH_TOKEN_FILE" >&2; exit 3; }
  export CLAUDE_CODE_OAUTH_TOKEN="$(< "$TOKEN_FILE")"
fi

[[ -f "$TASK_DIR/FROZEN.sha256" ]] \
  && ( cd "$TASK_DIR" && sha256sum -c FROZEN.sha256 >/dev/null 2>&1 ) \
  || { echo "FROZEN.sha256 missing or mismatched — re-freeze before running" >&2; exit 3; }

command -v docker >/dev/null || { echo "docker not found" >&2; exit 3; }
docker image inspect "$IMAGE" >/dev/null 2>&1 || { echo "missing image $IMAGE" >&2; exit 3; }

if [[ -n "$SOURCE_DIR" && ! -d "$SOURCE_DIR" ]]; then
  echo "AGRI_SOURCE_DIR=$SOURCE_DIR is not a directory" >&2; exit 3
fi

WS=""; CHOME=""
prepare_ws() {   # $1 = arm → fresh worktree + container HOME
  local arm="$1"
  WS="$RUNS/$arm/work"; CHOME="$RUNS/$arm/home"
  # Go writes its module cache read-only (dirs 0555), so a plain rm -rf hits
  # "Permission denied" on the parent dirs and, under set -e, aborts the run.
  # Restore write perms on the prior run's tree before wiping it.
  [[ -e "$RUNS/$arm" ]] && chmod -R u+w "$RUNS/$arm" 2>/dev/null || true
  rm -rf "$RUNS/$arm"; mkdir -p "$WS" "$CHOME/.cache" "$CHOME/go"
  cp -r "$TASK_DIR/skeleton" "$WS/skeleton"
  cp -r "$TASK_DIR/seed" "$WS/skeleton/seed"          # binary loads ./seed at startup
  cp "$TASK_DIR/CONTRACT.md" "$TASK_DIR/prompt.md" "$WS/"
  if [[ -n "$SOURCE_DIR" ]]; then                     # stage the port source (no node_modules/.git)
    mkdir -p "$WS/source"
    rsync -a --exclude node_modules --exclude .git --exclude dist --exclude .output \
      "$SOURCE_DIR"/ "$WS/source"/ 2>/dev/null \
      || cp -r "$SOURCE_DIR"/. "$WS/source"/ 2>/dev/null || true
  fi
  [[ -n "$TOKEN_FILE" && -r "$TOKEN_FILE" ]] && install -m 600 "$TOKEN_FILE" "$CHOME/.nullius-token"
}

dock() {
  timeout "$DOCK_TIMEOUT" docker run --rm -i \
    --user "$(id -u):$(id -g)" \
    -e CLAUDE_CODE_OAUTH_TOKEN -e HOME=/home/agent \
    -e CLAUDE_CODE_AUTO_COMPACT_WINDOW="$CAP" \
    -e BASH_DEFAULT_TIMEOUT_MS="$BASH_TIMEOUT_MS" -e BASH_MAX_TIMEOUT_MS="$BASH_TIMEOUT_MS" \
    -e GOCACHE=/home/agent/.cache/go-build -e GOPATH=/home/agent/go \
    -v "$WS:/work" -w /work -v "$CHOME:/home/agent" \
    "$IMAGE" "$@"
}

score() {
  local arm="$1"
  echo "[$arm] scoring against the hidden semantic-contract oracle…" >&2
  bash "$TASK_DIR/score.sh" "$RUNS/$arm/work/skeleton" 2>&1 | tee "$RUNS/$arm/score.txt" || true
}

src_line() {
  [[ -n "$SOURCE_DIR" ]] && echo "The full existing SPA is in ./source/ — your DEPTH reference (large; absorb slices, never whole-read it)." \
                         || echo "No ./source/ is mounted this run — implement ./CONTRACT.md directly (contract-only mode)."
}

run_plain() {
  echo "[plain] building (solo, no plugin, capped $CAP)…" >&2
  prepare_ws plain
  local task="Port an agricultural-monitoring dashboard to a server-rendered Via v0.7 app in Go, filling ./skeleton for real. $(src_line) ./CONTRACT.md is the AUTHORITATIVE observable API you must serve (its generic hooks/tokens override any surface strings in the source). ./skeleton/seed/ holds the synthetic data your binary loads at startup. Build every route, page, grid, validation token, the status log, the Datastar SSE actions, and the SMS broadcast. Close with a from-clean go build ./... and go vet ./..."
  dock claude -p --model "$MODEL" --effort "$EFFORT" \
      --permission-mode bypassPermissions --output-format json \
      "$task" \
    > "$RUNS/plain/result.json" 2>"$RUNS/plain/err.txt" || true
  score plain
}

run_recursive() {
  echo "[recursive] $LEADER_MODEL/$EFFORT leader → $CRAFT_MODEL/$EFFORT craftsman, capped $CAP…" >&2
  prepare_ws recursive
  mkdir -p "$WS/.claude/bin" "$WS/.claude/hooks"
  cp -r "$PLUGIN/agents" "$WS/.claude/agents"
  cp "$PLUGIN/hooks/diet-governor.mjs" "$WS/.claude/hooks/"
  install -m 755 "$ROOT/bin/nullius-build" "$WS/.claude/bin/nullius-build"
  cat > "$WS/.claude/settings.json" <<'JSON'
{ "hooks": { "PreToolUse": [
  { "matcher": "Read|Grep|Glob|WebFetch|WebSearch|Bash|Edit|Write|Agent|Task",
    "hooks": [ { "type": "command",
      "command": "node \"$CLAUDE_PROJECT_DIR/.claude/hooks/diet-governor.mjs\"" } ] } ] } }
JSON
  local leader_sys="You are a nullius LEADER on the main thread. You do NOT implement this port yourself. The diet governor will DENY your own whole-file reads, sweeps, and builds — by design: dispatch nullius-scout (Agent tool, haiku) for ALL absorption, batched in parallel. Your job: (1) map the source + contract via scouts; (2) author a THIN pointer-brief; (3) delegate the full build to the nested sonnet craftsman via nullius-build; (4) confirm its close-out."
  local task="Port an agricultural-monitoring dashboard to a server-rendered Via v0.7 app in Go, filling ./skeleton for real. $(src_line) ./CONTRACT.md is the AUTHORITATIVE observable API (its generic hooks/tokens override any source surface strings). ./skeleton/seed/ holds the synthetic startup data. Do exactly this:
1. Dispatch nullius-scout to map ./CONTRACT.md and the source (routes, auth/HMAC, validation order, pagination, the cross-database station→place join, the grid GeoJSON, the map tabs, the SSE actions) — file:line anchors, not prose.
2. Write ./brief.md: a THIN pointer-brief — INTENT + acceptance bar, the CONTRACT pointers, and a TERRAIN map. No stub/TODO verdicts.
3. Run the build (this exact command):
   .claude/bin/nullius-build --model $CRAFT_MODEL --effort $EFFORT --permission-mode bypassPermissions @brief.md ./skeleton
4. Report the craftsman's STATUS and whether ./skeleton builds; flag any empty/0-byte file."
  dock claude -p --model "$LEADER_MODEL" --effort "$EFFORT" \
      --permission-mode bypassPermissions --output-format json \
      --append-system-prompt "$leader_sys" "$task" \
    > "$RUNS/recursive/result.json" 2>"$RUNS/recursive/err.txt" || true
  score recursive
}

# nullius method WITHOUT the nested builder: same fable-5/low leader, same diet
# governor + scout agents as `recursive`, but the leader implements the port
# ITSELF (scouts for bulk absorption/verification, surgical edits by hand). No
# nullius-build call. Isolates whether the sonnet craftsman (recursion) is what
# does the work, or the nullius method alone on fable suffices.
run_nullius_solo() {
  echo "[nullius-solo] $MODEL/$EFFORT leader builds directly (governor+scouts, no builder), capped $CAP…" >&2
  prepare_ws nullius-solo
  mkdir -p "$WS/.claude/hooks"
  cp -r "$PLUGIN/agents" "$WS/.claude/agents"
  cp "$PLUGIN/hooks/diet-governor.mjs" "$WS/.claude/hooks/"
  cat > "$WS/.claude/settings.json" <<'JSON'
{ "hooks": { "PreToolUse": [
  { "matcher": "Read|Grep|Glob|WebFetch|WebSearch|Bash|Edit|Write|Agent|Task",
    "hooks": [ { "type": "command",
      "command": "node \"$CLAUDE_PROJECT_DIR/.claude/hooks/diet-governor.mjs\"" } ] } ] } }
JSON
  local leader_sys="You are a nullius leader working the task HANDS-ON. You implement this port YOURSELF — you read the decisive code, design, and write every Go file and edit directly with Edit/Write. The diet governor will DENY your own whole-file reads, wide sweeps, builds, tests, and vet — by design: dispatch nullius-scout (Agent tool, haiku) for ALL bulk absorption and for every build/test/vet run, batched in parallel when independent; their capped reports are your trusted record. Read surgically only the few files you edit. Do NOT delegate the implementation to any nested builder — there is none; the code is yours to write."
  local task="Port an agricultural-monitoring dashboard to a server-rendered Via v0.7 app in Go, filling ./skeleton for real — YOU write the code. $(src_line) ./CONTRACT.md is the AUTHORITATIVE observable API you must serve (its generic hooks/tokens override any source surface strings). ./skeleton/seed/ holds the synthetic startup data your binary loads. Build every route, page, grid, validation token, the status log, and the SMS broadcast. Dispatch nullius-scout to map the source+contract and to run the from-clean go build ./... and go vet ./... at close — never run them yourself. Close with the scout's verbatim build+vet record."
  dock claude -p --model "$MODEL" --effort "$EFFORT" \
      --permission-mode bypassPermissions --output-format json \
      --append-system-prompt "$leader_sys" "$task" \
    > "$RUNS/nullius-solo/result.json" 2>"$RUNS/nullius-solo/err.txt" || true
  score nullius-solo
}

mkdir -p "$RUNS"
case "${1:-all}" in
  plain)        run_plain ;;
  recursive)    run_recursive ;;
  nullius-solo) run_nullius_solo ;;
  all)          run_plain; run_recursive ;;
  *) echo "usage: run.sh [plain|recursive|nullius-solo|all]" >&2; exit 2 ;;
esac
echo "done — results under $RUNS/{plain,recursive,nullius-solo}/" >&2
