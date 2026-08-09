#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="/opt/kiss-startpage"
DATA_DIR="/var/lib/kiss-startpage"
SERVICE_NAME="kiss-startpage-api"
SERVICE_USER="kiss-startpage"
SERVICE_GROUP="kiss-startpage"
BIND_ADDR="0.0.0.0"
PORT="8788"
ENABLE_SERVICE=1
RUN_PREFLIGHT=1

usage() {
  cat <<USAGE
Usage: sudo $0 [options]

Options:
  --install-dir DIR      Install root (default: /opt/kiss-startpage)
  --data-dir DIR         Persistent data dir (default: /var/lib/kiss-startpage)
  --service-name NAME    systemd service name (default: kiss-startpage-api)
  --user USER            Service user (default: kiss-startpage)
  --group GROUP          Service group (default: kiss-startpage)
  --bind ADDR            Bind address (default: 0.0.0.0)
  --port PORT            Port (default: 8788)
  --no-enable            Do not enable/start the service
  --skip-preflight       Skip ops/preflight.sh
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-dir) INSTALL_DIR="${2:-}"; shift 2 ;;
    --data-dir) DATA_DIR="${2:-}"; shift 2 ;;
    --service-name) SERVICE_NAME="${2:-}"; shift 2 ;;
    --user) SERVICE_USER="${2:-}"; shift 2 ;;
    --group) SERVICE_GROUP="${2:-}"; shift 2 ;;
    --bind) BIND_ADDR="${2:-}"; shift 2 ;;
    --port) PORT="${2:-}"; shift 2 ;;
    --no-enable) ENABLE_SERVICE=0; shift ;;
    --skip-preflight) RUN_PREFLIGHT=0; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run as root (sudo)." >&2
  exit 1
fi

if [[ "$RUN_PREFLIGHT" -eq 1 ]]; then
  bash "$ROOT_DIR/ops/preflight.sh" --port "$PORT"
fi

if ! [[ "$PORT" =~ ^[0-9]+$ ]]; then
  echo "Invalid port: $PORT" >&2
  exit 1
fi

CURRENT_DIR="${INSTALL_DIR%/}/current"
PRIVATE_ICONS_DIR="${DATA_DIR%/}/private-icons"
ENV_FILE="/etc/default/${SERVICE_NAME}"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

for cmd in go node npm curl rsync; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Missing required command: $cmd" >&2
    exit 1
  fi
done

NODE_VERSION="$(node --version | sed 's/^v//')"
if [[ "$(printf '%s\n%s\n' '20.19.0' "$NODE_VERSION" | sort -V | head -1)" != "20.19.0" ]]; then
  echo "Node.js 20.19.0 or newer is required (found ${NODE_VERSION})." >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$PRIVATE_ICONS_DIR"

if ! getent group "$SERVICE_GROUP" >/dev/null; then
  groupadd --system "$SERVICE_GROUP"
fi

if ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
  useradd --system --no-create-home --gid "$SERVICE_GROUP" --shell /usr/sbin/nologin "$SERVICE_USER"
fi

BUILD_DIR="$(mktemp -d "${INSTALL_DIR%/}/.build.XXXXXX")"
trap 'rm -rf "$BUILD_DIR"' EXIT
mkdir -p "$BUILD_DIR/source/frontend-svelte" "$BUILD_DIR/source/backend-go" "$BUILD_DIR/runtime/frontend-svelte" "$BUILD_DIR/runtime/backend-go"

# Copy only build inputs. Repository metadata, tests, secrets and local runtime data
# can never enter the deployed application directory through this path.
cp "$ROOT_DIR/frontend-svelte/package.json" "$ROOT_DIR/frontend-svelte/package-lock.json" \
  "$ROOT_DIR/frontend-svelte/index.html" "$ROOT_DIR/frontend-svelte/vite.config.js" \
  "$BUILD_DIR/source/frontend-svelte/"
cp -a "$ROOT_DIR/frontend-svelte/src" "$ROOT_DIR/frontend-svelte/public" "$BUILD_DIR/source/frontend-svelte/"
cp "$ROOT_DIR/backend-go/go.mod" "$ROOT_DIR"/backend-go/*.go "$BUILD_DIR/source/backend-go/"
cp "$ROOT_DIR/startpage-default-config.json" "$BUILD_DIR/runtime/startpage-default-config.json"

echo "[1/4] Building frontend (Svelte/Vite)"
(
  cd "$BUILD_DIR/source/frontend-svelte"
  npm ci --no-fund --no-audit
  npm run build
  cp -a dist "$BUILD_DIR/runtime/frontend-svelte/dist"
)

echo "[2/4] Building backend (Go)"
(
  cd "$BUILD_DIR/source/backend-go"
  go build -buildvcs=false -o "$BUILD_DIR/runtime/backend-go/kissdash-go" .
  chmod 755 "$BUILD_DIR/runtime/backend-go/kissdash-go"
)

echo "[3/4] Installing the complete runtime candidate"
if [[ "$ENABLE_SERVICE" -eq 1 ]] && systemctl is-active --quiet "$SERVICE_NAME"; then
  systemctl stop "$SERVICE_NAME"
fi
mkdir -p "$CURRENT_DIR"
rsync -a --delete "$BUILD_DIR/runtime/" "$CURRENT_DIR/"

echo "[4/4] Applying ownership and runtime permissions"

chown -R root:root "$CURRENT_DIR"
chmod -R a+rX "$CURRENT_DIR"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$DATA_DIR"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$PRIVATE_ICONS_DIR"
chmod 750 "$DATA_DIR"
chmod 750 "$PRIVATE_ICONS_DIR"

set_env_value() {
  local key="$1"
  local value="$2"
  if grep -q "^${key}=" "$ENV_FILE" 2>/dev/null; then
    sed -i "s|^${key}=.*|${key}=${value}|" "$ENV_FILE"
  else
    printf '%s=%s\n' "$key" "$value" >> "$ENV_FILE"
  fi
}

if [[ ! -f "$ENV_FILE" ]]; then
  cat > "$ENV_FILE" <<ENV
# KISS Startpage runtime settings
# DASH_SESSION_TTL=315360000
# DASH_ICON_INDEX_TTL=21600
# DASH_ICON_SEARCH_MAX_LIMIT=30
ENV
fi
set_env_value DASH_BIND "$BIND_ADDR"
set_env_value DASH_PORT "$PORT"
set_env_value DASH_DATA_DIR "$DATA_DIR"
set_env_value DASH_PRIVATE_ICONS_DIR "$PRIVATE_ICONS_DIR"
set_env_value DASH_DEFAULT_CONFIG "$CURRENT_DIR/startpage-default-config.json"
set_env_value DASH_APP_ROOT "$CURRENT_DIR/frontend-svelte/dist"
chmod 640 "$ENV_FILE"

cat > "$UNIT_FILE" <<UNIT
[Unit]
Description=KISS Startpage API
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
WorkingDirectory=${CURRENT_DIR}
EnvironmentFile=-${ENV_FILE}
Environment=DASH_DEFAULT_CONFIG=${CURRENT_DIR}/startpage-default-config.json
Environment=DASH_APP_ROOT=${CURRENT_DIR}/frontend-svelte/dist
Environment=DASH_PRIVATE_ICONS_DIR=${PRIVATE_ICONS_DIR}
ExecStart=${CURRENT_DIR}/backend-go/kissdash-go
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
if [[ "$ENABLE_SERVICE" -eq 1 ]]; then
  systemctl enable --now "$SERVICE_NAME"
  systemctl restart "$SERVICE_NAME"
  HEALTH_HOST="$BIND_ADDR"
  if [[ "$HEALTH_HOST" == "0.0.0.0" || "$HEALTH_HOST" == "::" || "$HEALTH_HOST" == "localhost" ]]; then
    HEALTH_HOST="127.0.0.1"
  fi
  HEALTH_URL="http://${HEALTH_HOST}:${PORT}/health"
  HEALTHY=0
  for _ in $(seq 1 15); do
    if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
      HEALTHY=1
      break
    fi
    sleep 1
  done
  if [[ "$HEALTHY" -ne 1 ]]; then
    echo "Service failed its health check: $HEALTH_URL" >&2
    systemctl --no-pager --full status "$SERVICE_NAME" >&2 || true
    exit 1
  fi
fi

echo
if [[ "$ENABLE_SERVICE" -eq 1 ]]; then
  echo "Installed and started: ${SERVICE_NAME}"
else
  echo "Installed service unit: ${SERVICE_NAME} (not started)"
fi
echo "App dir: ${CURRENT_DIR}"
echo "Data dir: ${DATA_DIR}"
echo "Private icons dir: ${PRIVATE_ICONS_DIR}"
echo "Open: http://${BIND_ADDR}:${PORT}/ and /edit"
echo "First visit /edit: create the first admin account"
