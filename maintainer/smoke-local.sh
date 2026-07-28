#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_ROOT="$(mktemp -d)"
PORT="${KISS_SMOKE_PORT:-18789}"
PID=""

cleanup() {
  if [[ -n "$PID" ]]; then
    kill "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
  rm -rf -- "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

(
  cd "$REPO_ROOT/frontend-svelte"
  npm run build
)

(
  cd "$REPO_ROOT/backend-go"
  go build -buildvcs=false -o "$TMP_ROOT/kissdash-go" .
)

DASH_BIND=127.0.0.1 \
DASH_PORT="$PORT" \
DASH_DATA_DIR="$TMP_ROOT/data" \
DASH_PRIVATE_ICONS_DIR="$TMP_ROOT/data/private-icons" \
DASH_DEFAULT_CONFIG="$REPO_ROOT/startpage-default-config.json" \
DASH_APP_ROOT="$REPO_ROOT/frontend-svelte/dist" \
  "$TMP_ROOT/kissdash-go" >"$TMP_ROOT/server.log" 2>&1 &
PID=$!

if ! bash "$REPO_ROOT/ops/smoke-test.sh" \
  --base-url "http://127.0.0.1:$PORT" \
  --username smokeadmin \
  --password temporary-smoke-password; then
  cat "$TMP_ROOT/server.log" >&2
  exit 1
fi

BACKUP_DATA="$TMP_ROOT/backup-data"
BACKUP_OUT="$TMP_ROOT/backups"
mkdir -p "$BACKUP_DATA/private-icons"
printf '{}\n' > "$BACKUP_DATA/dashboard-config.json"
printf '{}\n' > "$BACKUP_DATA/users.json"
printf '<svg xmlns="http://www.w3.org/2000/svg"/>\n' > "$BACKUP_DATA/private-icons/test.svg"

bash "$REPO_ROOT/ops/backup.sh" --data-dir "$BACKUP_DATA" --out-dir "$BACKUP_OUT"
BACKUP_ARCHIVE="$(find "$BACKUP_OUT" -maxdepth 1 -type f -name 'kiss-startpage-backup-*.tar.gz' -print -quit)"
[[ -n "$BACKUP_ARCHIVE" ]]
[[ "$(stat -c '%a' "$BACKUP_ARCHIVE")" == "600" ]]
tar -tzf "$BACKUP_ARCHIVE" | grep -q 'dashboard-config.json'
tar -tzf "$BACKUP_ARCHIVE" | grep -q 'users.json'
