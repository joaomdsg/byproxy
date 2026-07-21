#!/usr/bin/env bash
# Score an arm's agri-alert-via-port attempt against the hidden semantic-contract
# oracle. The oracle boots the arm's compiled binary and drives it over HTTP,
# asserting the observable behavior in CONTRACT.md (data values are checked
# against the synthetic seed ground truth in fixtures/).
#
# Usage: score.sh <target-dir>
#   <target-dir> — dir containing the arm's Go app (go.mod + main.go);
#                  defaults to ./skeleton relative to this script.
# Output (last line): SCORE <passed>/<total>
set -u

TASK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_DIR="${1:-$TASK_DIR/skeleton}"

# Total = number of TestT* functions in the hidden suite (no drift vs SCORE_MAX).
TOTAL="$(grep -rhoE '^func TestT[0-9]+' "$TASK_DIR/hidden"/*.go 2>/dev/null | sort -u | wc -l)"
[ "$TOTAL" -gt 0 ] 2>/dev/null || TOTAL=1

TARGET_DIR="$(cd "$TARGET_DIR" && pwd)" || { echo "SCORE 0/$TOTAL"; exit 1; }

# Config the arm's binary reads (see CONTRACT.md §1). Synthetic, fixed.
export AGRI_SEED_DIR="$TASK_DIR/seed"
export AGRI_SESSION_SECRET="bench-secret-agri-alert"
export AGRI_MANAGER_PASSWORD="manager-pass"
export AGRI_VIEWER_PASSWORD="viewer-pass"
export AGRI_SECURE_COOKIE="0"

BIN="$(mktemp -d)/app"
trap 'rm -rf "$(dirname "$BIN")"' EXIT

# A non-compiling submission scores 0.
if ! (cd "$TARGET_DIR" && go build -o "$BIN" .); then
  echo "build failed in $TARGET_DIR" >&2
  echo "SCORE 0/$TOTAL"
  exit 1
fi

# Smoke-launch on a free port so TARGET_URL is available; the hidden suite
# boots its own process per test from TARGET_BIN (and owns AGRI_SMS_URL, which
# it points at an in-test mock so both the sent and failed paths are testable).
PORT="$(python3 - <<'EOF'
import socket
s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()
EOF
)"
PORT="$PORT" AGRI_SMS_URL="http://127.0.0.1:9/none" "$BIN" & APP_PID=$!
trap 'kill "$APP_PID" 2>/dev/null; rm -rf "$(dirname "$BIN")"' EXIT
export TARGET_URL="http://127.0.0.1:$PORT"
export TARGET_BIN="$BIN"
export TARGET_DIR

for _ in $(seq 1 50); do
  curl -fsS "$TARGET_URL/login" >/dev/null 2>&1 && break
  sleep 0.1
done

OUT="$(cd "$TASK_DIR/hidden" && go test -count=1 -v ./... 2>&1)"
STATUS=$?
echo "$OUT"

PASSED="$(echo "$OUT" | grep -c '^--- PASS: TestT')"
echo "SCORE $PASSED/$TOTAL"
[ "$STATUS" -eq 0 ]
