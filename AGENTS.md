# KISS Startpage rules of the road

These rules apply to the entire repository.

## Architecture

- Keep the browser UI in `frontend-svelte/`.
- Keep authentication, persistence, validation, and external icon-provider access in `backend-go/`.
- Add icon providers and Iconify collections through the backend registries and `/api/icons/sources`; the Svelte client only renders the server catalogue and per-user provider preferences.
- Preserve backward compatibility for saved dashboard configuration within minor releases.
- Treat `/opt/kiss-startpage/current` as replaceable application code and `/var/lib/kiss-startpage` as persistent runtime data.

## Repository boundaries

- Commit source, lockfiles, generic starter data, installer scripts, tests, documentation, and workflows.
- Never commit real dashboard configuration, users, sessions, private icons, `.env` files, credentials, keys, backups, build output, or dependencies.
- Publish generated installers and binaries as GitHub Actions artifacts or GitHub Release assets, never as tracked files.

## Required checks

Before publishing a change, run:

```bash
cd backend-go && gofmt -w *.go && go test ./... && go vet ./...
cd ../frontend-svelte && npm ci --no-fund --no-audit && npm run build && npm test
cd .. && bash maintainer/smoke-local.sh
```

Update `CHANGELOG.md` for user-visible changes. Releases must use a lowercase `vX.Y.Z` tag matching `appVersion` in `backend-go/main.go`.

## Production-first publishing gate

- Build and test changes locally in Jarvis.
- Deploy the candidate directly from the local checkout to production without using GitHub as the deployment source.
- Stop and wait for the maintainer to validate the candidate on production.
- Push commits, tags, or releases to GitHub only after the maintainer gives separate explicit permission for that exact candidate.
