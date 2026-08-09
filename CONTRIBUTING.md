# Contributing to KISS Startpage

KISS means simple code and simple use. Prefer a direct function and an existing
data flow over a new framework, abstraction, compatibility layer, or endpoint.

## Boundaries

- `frontend-svelte/src/App.svelte` coordinates the page and editor.
- `frontend-svelte/src/components/` contains focused UI components.
- `frontend-svelte/src/lib/theme.js` is the only owner of theme defaults,
  normalization, presets, and CSS-variable mapping.
- `backend-go/` owns authentication, persistence, validation, static serving,
  and remote icon access.
- `backend-go/icon_sources.go` is the icon catalogue; the browser does not keep
  a second provider list.
- `/opt/kiss-startpage/current` is replaceable code. `/var/lib/kiss-startpage`
  is private persistent data.

Keep saved dashboard data backward-compatible. Remove obsolete internal APIs
when every caller is migrated; do not maintain unused aliases indefinitely.

## What belongs in Git

Commit source, tests, documentation, lockfiles, generic starter data, and
installer/workflow definitions. Never commit real dashboards, users, sessions,
private icons, `.env` files, keys, credentials, backups, dependencies, build
output, or generated binaries.

## Before production

```bash
(cd backend-go && gofmt -w *.go && go test ./... && go vet ./...)
(cd frontend-svelte && npm ci --no-fund --no-audit && npm run build && npm test)
bash maintainer/smoke-local.sh
docker build -t kiss-startpage:test .
```

For this repository, publishing is production-first:

1. Back up persistent production data.
2. Deploy the tested candidate directly from the local Jarvis checkout.
3. Let the maintainer validate that exact candidate in production.
4. Commit or push to GitHub only after separate, explicit permission.

Update `CHANGELOG.md` and the `appVersion` constant for a release candidate.
See `RELEASING.md` for the short release procedure.
