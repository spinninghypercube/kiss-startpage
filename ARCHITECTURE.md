# Architecture

KISS Startpage is a small self-hosted application with a replaceable application directory and a separate persistent data directory.

## Runtime

```text
Browser
  │ HTTP
  ▼
Nginx :80
  │ reverse proxy
  ▼
Go service :8788
  ├─ serves the built Svelte application
  ├─ authenticates editor requests
  ├─ reads and atomically writes dashboard JSON
  └─ searches/imports icons from registered providers

Application: /opt/kiss-startpage/current
Persistent data: /var/lib/kiss-startpage
```

The Svelte client owns presentation, edit interactions, and compatibility normalization. The Go service owns trust boundaries: authentication, filesystem writes, external HTTP requests, validation, and provider allowlists.

There is one owner for each cross-cutting concern: `theme.js` owns theme data
and CSS variables, `startpage-common.js` owns dashboard compatibility and the
small API client, and `icon_sources.go` owns the provider catalogue. Keep
components focused on rendering and user interaction; do not duplicate these
registries or defaults in components.

## Configuration

The public startpage reads `GET /api/config`. Editor writes require an authenticated session. The server writes JSON atomically so a failed write cannot leave a partial config file.

The frontend must show an explicit retry state when the API cannot be reached. `startpage-default-config.json` seeds a new server-side data directory; it is not a runtime fallback for an existing dashboard.

Tabs, groups, and buttons require stable IDs. Client normalization preserves IDs and deterministically repairs missing or duplicate IDs before keyed rendering.

## Icons

`backend-go/icon_sources.go` is the source of truth for providers, curated Iconify collections, and license metadata. The editor retrieves the non-sensitive catalogue through public `GET /api/icons/sources`. Authenticated provider preferences are stored per user in `users.json`; they are runtime data and never repository content.

One search request queries every relevant enabled provider concurrently and returns independent result groups, so a remote provider failure cannot hide successful results from other providers. Providers without applicable input are omitted: text providers require two search characters and website discovery requires an external or internal URL. Scheme-less external website URLs default to HTTPS; scheme-less internal URLs default to HTTP. The registered providers are Iconify, selfh.st/icons, Dashboard Icons, the private local icon directory, URL favicon discovery, and strictly filtered CC0/public-domain Wikimedia Commons media.

Search and import requests use one generic search endpoint and one generic
import endpoint through the Go service. Remote image sizes and formats are
validated and untrusted SVG content is rejected. Wikimedia imports prefer the
original SVG after conservative validation and fall back to a PNG thumbnail
when needed. Imported image data and optional provenance metadata are embedded
in the saved button configuration, so a rendered dashboard does not depend on
the provider remaining available.

## Deliberate security choices

The default session lifetime is ten years and the minimum password length is
four characters. These are explicit convenience choices for the current
single-user deployment, not accidental defaults. Internet-facing deployments
should override `DASH_SESSION_TTL`, use a strong password, bind to loopback
behind HTTPS, and apply access controls at the reverse proxy or firewall.

## Releases

`main` is expected to stay releasable, but this small project intentionally does not require pull requests or branch protection. GitHub Actions run the same build and regression checks on every pushed branch. Versioned production deployments use tags rather than an arbitrary `main` snapshot.
