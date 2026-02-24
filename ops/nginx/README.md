# nginx Example

This folder contains an example `nginx` site config for deployments that:

- serve frontend files directly from a release/web root
- proxy only `/api/` to the Python backend

File:
- `ops/nginx/kiss-this-dashboard.conf`

Behavior (intentional):
- `/` serves the dashboard
- `/edit` redirects to `/edit.html`
- `/admin` and `/admin.html` redirect to `/edit.html`
- non-existent URLs return `404` (no SPA-style `/index.html` fallback)
