#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Usage: $0 X.Y.Z" >&2
  exit 1
fi

cd "$REPO_ROOT"

if [[ "$(git branch --show-current)" != "main" ]]; then
  echo "Run releases from main." >&2
  exit 1
fi
if [[ -n "$(git status --porcelain)" ]]; then
  echo "Working tree is not clean." >&2
  exit 1
fi

git fetch origin main --tags
if [[ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]]; then
  echo "Local main must exactly match origin/main." >&2
  exit 1
fi

APP_VERSION="$(sed -nE 's/^[[:space:]]*appVersion[[:space:]]*=[[:space:]]*"([^"]+)".*/\1/p' backend-go/main.go | head -1)"
if [[ "$APP_VERSION" != "$VERSION" ]]; then
  echo "backend-go/main.go has appVersion $APP_VERSION, expected $VERSION." >&2
  exit 1
fi
if ! grep -q "^## \\[$VERSION\\]" CHANGELOG.md; then
  echo "CHANGELOG.md has no [$VERSION] release section." >&2
  exit 1
fi

TAG="v$VERSION"
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Tag already exists: $TAG" >&2
  exit 1
fi

(
  cd backend-go
  go test ./...
  go vet ./...
)
(
  cd frontend-svelte
  npm ci --no-fund --no-audit
  npm test
)
bash maintainer/smoke-local.sh

git tag -a "$TAG" -m "Release $TAG"
echo "Created $TAG. Push it explicitly with: git push origin $TAG"
