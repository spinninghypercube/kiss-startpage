# Changelog

All notable changes to this project should be documented in this file.

The format is based on Keep a Changelog and this project uses semantic versioning where practical.

## [Unreleased]

## [2.8.0] - 2026-08-09

### Added
- Added one searchable icon picker across all enabled providers and Iconify collections
- Added regression coverage for mobile safe areas, KISS branding, static assets, authentication sessions, and serialized configuration saves
- Added a concise contributor guide and production-first release procedure

### Changed
- Consolidated theme defaults, presets, normalization, and CSS variables into one module
- Simplified drag-and-drop to update configuration directly from Sortable events
- Serialized editor saves so delayed responses cannot overwrite newer changes
- Updated Svelte, Vite, Playwright, and Vitest; removed unused icon-generation and drag dependencies
- Staged Linux deployments now install only a small runtime allowlist after successful builds
- Static hashed assets now use immutable caching while HTML remains revalidated
- Docker and Linux tooling now consistently require Node.js 20.19 or newer
- Removed obsolete provider-specific icon import endpoints
- Installation documentation and example nginx/systemd files now use the same paths and port defaults

### Fixed
- Theme-derived group borders and corner radii now remain consistent in view mode, Tab Settings, Theme Editor, group boxes, and Add group
- Successfully imported icon search results remain visibly selected until another icon is chosen
- Username changes update every active session; password changes revoke other sessions; removed users can no longer retain access
- Session cookies automatically use `Secure` behind HTTPS and carry an explicit lifetime
- Linux and Docker installers now fail clearly when the deployed service stays unhealthy
- Windows updates build beside the active runtime, preserve dependency ownership records, and restore the previous runtime after a failed health check

## [2.7.2] - 2026-08-04

### Added
- Added durable KISS Startpage branding with SVG, Safari touch icon, and installable web-app icons
- Added a standalone web-app manifest and mobile browser metadata
- Added a built-in KISS Brand theme based on the cyan, graphite, and near-black logo palette

### Changed
- Frontend builds now regenerate the complete KISS icon set from its tracked SVG source
- iPhone browser and installed web-app layouts now respect all safe areas, including the Dynamic Island and home indicator
- Safari status and overscroll surfaces now follow the active startpage background when themes change
- Theme preset normalization now preserves tab hover and group border colors
- Group border colors now render on group boxes in both view and edit modes
- The Theme Editor divider now follows the configured group border color
- The outer Tab Settings and Theme Editor panel now uses the configured group border color
- Tab Settings and Add group panels now use the same corner radius as group boxes
- The built-in KISS Brand preset now matches the production Custom theme 3 palette

## [2.7.1] - 2026-07-28

### Changed
- Simplified icon results to use the button modal's existing scroll area, plain provider headings, and a single empty-state message
- Hidden inactive and empty provider result groups until a real text or website-icon search runs
- Website icon discovery now accepts external and internal URLs without an explicit scheme

### Fixed
- External website icons now default to HTTPS while scheme-less internal URLs default to HTTP

## [2.7.0] - 2026-07-27

### Added
- Parallel search across Iconify, selfh.st/icons, Dashboard Icons, private local icons, website favicons, and strictly filtered Wikimedia Commons media
- Per-user server-side provider preferences with all providers enabled by default
- Website favicon discovery and safe local embedding
- Wikimedia search restricted to CC0/public-domain files without attribution or other declared restrictions
- Optional provider, license, and source provenance on imported button icons

### Changed
- Removed the provider and Iconify collection dropdowns in favor of account-persistent provider checkboxes
- Expanded the curated Iconify search from ten to fourteen collections with Bootstrap Icons, Devicon, Heroicons, and Remix Icon
- Results are grouped per provider and remain usable when another provider is unavailable
- Icon picker controls and result groups now consistently use active theme colors

### Security
- Added download limits, redirect limits, MIME checks, and conservative SVG validation for new remote import providers

## [2.6.0] - 2026-07-27

### Changed
- Icon search now presents selfh.st and Iconify as providers, with all ten Iconify libraries bundled into a dedicated collection selector
- Neutral, primary, warning, danger, upload, and icon-search controls now derive their colors from the active theme
- Icon source and license metadata can be read without an editor session so the catalogue no longer silently falls back to an incomplete hardcoded list

### Fixed
- The editor now shows an explicit retry state if the icon catalogue cannot be loaded
- Plain modal and editor action buttons no longer retain Bulma's fixed white, blue, or red palette when switching themes

## [2.5.1] - 2026-07-27

### Fixed
- Backup archives are now created with owner-only permissions because they can contain dashboard configuration and authentication data

## [2.5.0] - 2026-07-27

### Added
- Broad icon source registry with Material Design Icons, Material Symbols, Tabler, Lucide, Phosphor, Font Awesome 6 Solid, Fluent UI, and Carbon
- Icon source and license metadata endpoint for the editor
- Brave/Chromium reload and tab-switch regression coverage
- Lightweight architecture, repository, CI, licensing, and release rules

### Changed
- Startpage groups and buttons now use stable keyed rendering
- Configuration loading retries once and shows an explicit Retry state instead of silently displaying demo data
- Versioned releases validate the application version and generate GitHub release notes

### Fixed
- Duplicate or missing group/button IDs are normalized without dropping content
- Smoke tests now parse compact JSON correctly

## [2.4.0] - 2026-03-19

### Added
- Per-tab setting to choose whether dashboard links open in the current tab or a new tab

### Changed
- Tab setting toggles now render with the switch before the label for cleaner alignment in the editor

## [1.3.0] - 2026-02-23

### Added
- Starter default config seeded from the maintainer's `Default` tab plus an empty `Extra tab`
- Docker Compose quick-start (`docker-compose.yml`, `.env.example`)
- Debian/Ubuntu install tooling (`ops/install.sh`) and preflight checks (`ops/preflight.sh`)
- Backup/restore scripts (`ops/backup.sh`, `ops/restore.sh`)
- Smoke test script (`ops/smoke-test.sh`)
- Release helper script (`maintainer/release.sh`) and README deployment docs
- First-run admin account bootstrap flow (no default credentials for new installs)
- Admin `Reset to Starter` action
- Backend static file serving fallback for single-service deployments
- Admin drag-and-drop reordering for tabs, groups, and buttons (desktop + touch)
- Tab theme preset manager with built-in presets plus save/load for per-tab custom presets

### Changed
- New installs now start with `Default` + `Extra tab` starter tabs from `startpage-default-config.json`
- Group/button reorder arrow mini-actions were replaced with drag handles in the admin editor
