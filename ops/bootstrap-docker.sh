#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/spinninghypercube/kiss-this-dashboard.git"
BRANCH="main"
PORT="8788"
INSTALL_DIR=""

usage() {
  cat <<USAGE
One Shot Install (Docker Compose)

Usage: curl -fsSL <this script> | bash [-s -- [options]]

Options:
  --port PORT           Host port to expose (default: 8788)
  --dir DIR             Clone/install directory (default: /opt/kiss-this-dashboard-docker if root, otherwise ~/kiss-this-dashboard-docker)
  --branch NAME         Git branch or tag to install (default: main)
  --repo URL            Git repo URL (default: upstream GitHub repo)
  -h, --help            Show this help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --port) PORT="${2:-}"; shift 2 ;;
    --dir) INSTALL_DIR="${2:-}"; shift 2 ;;
    --branch) BRANCH="${2:-}"; shift 2 ;;
    --repo) REPO_URL="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "$INSTALL_DIR" ]]; then
  if [[ "${EUID}" -eq 0 ]]; then
    INSTALL_DIR="/opt/kiss-this-dashboard-docker"
  else
    INSTALL_DIR="${HOME}/kiss-this-dashboard-docker"
  fi
fi

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_cmd git
require_cmd docker

DOCKER_PREFIX=()
if ! docker info >/dev/null 2>&1; then
  if command -v sudo >/dev/null 2>&1 && sudo docker info >/dev/null 2>&1; then
    DOCKER_PREFIX=(sudo)
  else
    echo "Docker daemon is not reachable for the current user." >&2
    echo "Run as a user in the docker group, or rerun with sudo." >&2
    exit 1
  fi
fi

if "${DOCKER_PREFIX[@]}" docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD=("${DOCKER_PREFIX[@]}" docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD=("${DOCKER_PREFIX[@]}" docker-compose)
else
  echo "Docker Compose is required (docker compose plugin or docker-compose)." >&2
  exit 1
fi

if [[ -e "$INSTALL_DIR" && ! -d "$INSTALL_DIR/.git" ]]; then
  echo "Target path exists and is not a git checkout: $INSTALL_DIR" >&2
  exit 1
fi

if [[ -d "$INSTALL_DIR/.git" ]]; then
  echo "[bootstrap-docker] Reusing existing checkout: $INSTALL_DIR"
  git -C "$INSTALL_DIR" fetch --tags origin
  git -C "$INSTALL_DIR" checkout "$BRANCH"
  git -C "$INSTALL_DIR" pull --ff-only origin "$BRANCH"
else
  mkdir -p "$(dirname "$INSTALL_DIR")"
  echo "[bootstrap-docker] Cloning ${REPO_URL} (${BRANCH}) to $INSTALL_DIR"
  git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
fi

ENV_FILE="$INSTALL_DIR/.env"
if [[ -f "$ENV_FILE" ]]; then
  if grep -q '^KISS_PORT=' "$ENV_FILE"; then
    sed -i "s/^KISS_PORT=.*/KISS_PORT=${PORT}/" "$ENV_FILE"
  else
    printf '\nKISS_PORT=%s\n' "$PORT" >> "$ENV_FILE"
  fi
else
  printf 'KISS_PORT=%s\n' "$PORT" > "$ENV_FILE"
fi

echo "[bootstrap-docker] Starting container build and app"
(
  cd "$INSTALL_DIR"
  "${COMPOSE_CMD[@]}" up -d --build
)

if command -v curl >/dev/null 2>&1; then
  echo "[bootstrap-docker] Waiting for health endpoint"
  for _ in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
fi

host_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
host_ip="${host_ip:-127.0.0.1}"
echo
echo "One Shot Install (Docker) complete."
echo "Open:  http://${host_ip}:${PORT}/"
echo "Edit:  http://${host_ip}:${PORT}/edit"
echo "App dir: $INSTALL_DIR"

