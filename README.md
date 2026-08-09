# KISS Startpage

Your homelab start page, built in the browser.

## Features

- WYSIWYG layout and theming
- Drag-and-drop tabs, groups, and buttons
- Search across six icon providers, including fourteen curated Iconify collections, Dashboard Icons, local icons, website favicons, and strictly filtered Wikimedia Commons media
- Local auth and local data storage
- Docker or systemd install on Linux
- One-click EXE installer on Windows
- Mobile friendly

![KISS2](https://github.com/user-attachments/assets/ac51d41b-fb4f-454e-acee-7e43b74cd556)

## Quick Start

Requirements by method:

- Windows: Windows 10/11 or Windows Server with `winget` when Git, Node.js,
  Go, or NSSM still needs to be installed.
- Linux systemd: Debian or Ubuntu; the installer installs its build tools.
- Linux Docker: Git, curl, Docker, and Docker Compose must already work for
  the current user (or through `sudo`).

### Windows

[Download Windows installer (.exe)](https://github.com/spinninghypercube/kiss-startpage/releases/download/windows-installer-latest/kiss-startpage-bootstrap.exe)

- Run as Administrator.
- Opens on `http://<your-server-ip>:8788`.
- Installs a Windows service, Start Menu shortcuts, and an uninstaller (Settings → Apps).

Optional PowerShell one-shot:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap-windows.ps1 | iex"
```

### Linux (One-shot)

Docker:

```bash
curl -fsSL https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap-docker.sh | bash
```

Debian/Ubuntu systemd:

```bash
curl -fsSL https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap.sh | sudo bash
```

## Installer Options

| Installer | Main Script | Useful Flags |
|---|---|---|
| Windows | `ops/bootstrap-windows.ps1` | `-Port`, `-Bind`, `-InstallRoot`, `-DataDir`, `-Branch`, `-NoService`, `-NoAutoStart`, `-SkipDependencyInstall` |
| Linux systemd | `ops/bootstrap.sh` | `--port`, `--bind`, `--install-dir`, `--data-dir`, `--branch` |
| Linux Docker | `ops/bootstrap-docker.sh` | `--port`, `--dir`, `--branch` |

### Sample Commands (Defaults)

Windows EXE (default install/update):

- Download and run: `kiss-startpage-bootstrap.exe` as Administrator

Windows PowerShell (default install):

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap-windows.ps1 | iex"
```

Linux Debian/Ubuntu systemd (default install):

```bash
curl -fsSL https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap.sh | sudo bash
```

Linux Debian/Ubuntu systemd (local reverse proxy on same host):

```bash
curl -fsSL https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap.sh | sudo bash -s -- --bind 127.0.0.1
```

Linux Docker (default install):

```bash
curl -fsSL https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap-docker.sh | bash
```

Linux Docker from a downloaded checkout:

```bash
cp .env.example .env
docker compose up -d --build
curl -fsS http://127.0.0.1:8788/health
```

## Updating

Update by rerunning the installer you used initially:

- Linux systemd: `curl -fsSL https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap.sh | sudo bash`
- Linux Docker: `curl -fsSL https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap-docker.sh | bash`
- Windows PowerShell: `powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/spinninghypercube/kiss-startpage/main/ops/bootstrap-windows.ps1 | iex"`
- Windows EXE: rerun `kiss-startpage-bootstrap.exe` as Administrator

Backup and restore (Linux/systemd installs):

- Backup: `sudo bash ops/backup.sh`
- Restore: `sudo bash ops/restore.sh --backup <file> --restart-service`

Smoke test:

- `bash ops/smoke-test.sh --base-url http://127.0.0.1:8788 --username smokeadmin --password 'smoketest123'`

Windows EXE update notes:

- Yes, the EXE can update an existing Windows install in place.
- Default install path users can just rerun the EXE.
- If you use custom paths, pass the same `-InstallRoot` and `-DataDir` values when rerunning so the same instance is updated.

## Security Notes

- First visit creates the first admin account. No default shared credentials for new installs.
- For same-host reverse proxy setups, use loopback bind:
  - Linux systemd: `--bind 127.0.0.1`
  - Windows: `-Bind 127.0.0.1`
- Nginx example config is in `ops/nginx/`.
- Icon source/license metadata is public; provider preferences, icon search, and import remain restricted to an authenticated editor session.

## Where Things Live

- Linux systemd: app `/opt/kiss-startpage/current`, data `/var/lib/kiss-startpage`
- Linux Docker: persistent data in `startpage_data` volume
- Windows: app/data under `C:\ProgramData\KissStartpage` by default

## Repo Layout

- `backend-go/` API + auth + config storage + static file serving
- `frontend-svelte/` UI source and build
- `ops/` install/backup/restore/smoke-test and one-shot scripts
- `.github/workflows/windows-bootstrap-exe.yml` EXE build + release publishing

## Development

Development requires Go 1.24+ and Node.js 20.19+.

Use two terminals for live development:

```bash
cd backend-go
DASH_DATA_DIR="$(mktemp -d)" DASH_APP_ROOT="../frontend-svelte/dist" go run .
```

```bash
cd frontend-svelte
npm ci --no-fund --no-audit
npm run dev
```

Before publishing a change, run backend tests, frontend tests, and the local smoke test:

```bash
(cd backend-go && go test ./... && go vet ./...)
(cd frontend-svelte && npm ci --no-fund --no-audit && npm test)
bash maintainer/smoke-local.sh
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the maintenance rules,
[ARCHITECTURE.md](ARCHITECTURE.md) for component and data boundaries, and
[RELEASING.md](RELEASING.md) for the production-first release procedure.

## License

KISS Startpage is available under the [MIT License](LICENSE). Imported icon libraries keep their own licenses; the editor shows source-specific license information.
