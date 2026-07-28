# Releasing

KISS Startpage uses semantic versions where practical. `backend-go/main.go` is the version source of truth.

## Repository contents

Commit source code, tests, documentation, lockfiles, the generic starter config, installers, and workflows. Never commit production dashboard data, user/session files, private icons, `.env`, credentials, backups, dependencies, build directories, or generated executables.

## Release checklist

1. Add the user-visible changes under a dated `CHANGELOG.md` section for the new version.
2. Set `appVersion` in `backend-go/main.go` to the same version.
3. Merge the tested change to `main` and ensure GitHub CI is green.
4. Run `bash maintainer/release.sh X.Y.Z`.
5. Push explicitly:

   ```bash
   git push origin main
   git push origin vX.Y.Z
   ```

6. Confirm the tag workflow created the GitHub Release and attached both Windows installer filenames.
7. Back up persistent data before updating a production instance.

The rolling Windows installer remains a prerelease convenience asset. Generated executables belong in Actions/Releases, not in Git history.
