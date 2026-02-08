# Changelog

All notable changes to this project will be documented in this file.

## [0.2.0] - 2026-02-08

### Added
- **App Rotation System**
  - Automatic cycling through apps with configurable dwell time
  - Per-app dwell time override
  - Enable/disable individual apps in rotation
  - Skip to next app button
  
- **App Management API**
  - List installed apps (`GET /api/apps`)
  - Browse community apps (`GET /api/apps/community`)
  - Search community apps (`GET /api/apps/community/search?q=...`)
  - Install from community (`POST /api/apps/install`)
  - Uninstall apps (`DELETE /api/apps/{id}`)
  - Save app config (`PUT /api/apps/{id}/config`)

- **Display Control API**
  - Brightness control (`PUT /api/display/brightness`)
  - Power on/off (`PUT /api/display/power`)
  - Rotation enabled toggle (`PUT /api/rotation/enabled`)
  - Skip to next app (`POST /api/display/skip`)

- **Notification System**
  - Push text notifications (`POST /api/notify`)
  - Priority levels: low, normal, high, sticky
  - Show specific app temporarily (`POST /api/show`)

- **Enhanced Web Dashboard**
  - Live preview with auto-refresh
  - Brightness slider
  - Power and rotation toggles
  - App management UI
  - Skip button

- **Configuration Persistence**
  - Settings saved to `/data/config.json`
  - Survives add-on restarts

### Changed
- Server now starts with full initialization (display, apps, rotation)
- Version bumped to 0.2.0

## [0.1.2] - 2026-02-07

### Added
- Basic Pixlet integration working
- Web dashboard with live preview
- Frame endpoint for LED matrix clients
- Sample clock app

## [0.1.0] - 2026-02-06

### Added
- Initial add-on skeleton
- Basic HTTP server
- HA ingress support
