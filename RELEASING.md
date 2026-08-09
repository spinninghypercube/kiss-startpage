# Releasing

KISS Startpage uses semantic versions where practical. `backend-go/main.go` is the version source of truth.

## Repository contents

Commit source code, tests, documentation, lockfiles, the generic starter config, installers, and workflows. Never commit production dashboard data, user/session files, private icons, `.env`, credentials, backups, dependencies, build directories, or generated executables.

## Release checklist

1. Add the user-visible changes under a dated `CHANGELOG.md` section for the new version.
2. Set `appVersion` in `backend-go/main.go` to the same version.
3. Run all checks from `CONTRIBUTING.md` locally.
4. Back up persistent production data.
5. Deploy the candidate from the local Jarvis checkout and let the maintainer validate it in production.
6. Wait for separate, explicit permission to publish this exact candidate.
7. Commit and push `main`, then confirm GitHub CI is green.
8. Run `bash maintainer/release.sh X.Y.Z` and push the tag explicitly:

   ```bash
   git push origin vX.Y.Z
   ```

9. Confirm the tag workflow created the GitHub Release and attached both Windows installer filenames.

The rolling Windows installer remains a prerelease convenience asset. Generated executables belong in Actions/Releases, not in Git history.
