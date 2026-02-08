# Mosaic

LED matrix display server for Home Assistant. Run Tidbyt-compatible apps on DIY displays like the Interstate 75W.

## Features

- **App Rotation** — Cycle through multiple apps with configurable timing
- **Community Apps** — Install apps from the Tidbyt community
- **Web Dashboard** — Manage displays and apps via browser
- **HA Integration** — Control via Home Assistant entities and automations
- **Multi-Display** — Support multiple LED matrices

## Quick Start

1. Install this add-on
2. Open the web UI (click "Open Web UI" on the add-on page)
3. Install some apps from the community browser
4. Add apps to your rotation

## API

The add-on exposes a REST API for integration with Home Assistant or custom clients.

### Status
```
GET /api/status
→ {status, version, current_app, brightness, power, rotation_enabled, display}
```

### Display Control
```
GET /api/display
PUT /api/display/brightness {brightness: 0-100}
PUT /api/display/power {power: true/false}
POST /api/display/skip
```

### Rotation
```
GET /api/rotation
PUT /api/rotation {apps: [...]}
PUT /api/rotation/enabled {enabled: true/false}
POST /api/rotation/apps {app_id: "..."}
DELETE /api/rotation/apps/{id}
```

### Apps
```
GET /api/apps
GET /api/apps/{id}
GET /api/apps/community
GET /api/apps/community/search?q=...
POST /api/apps/install {app_id: "..."}
DELETE /api/apps/{id}
PUT /api/apps/{id}/config {...}
```

### Notifications
```
POST /api/notify {text, color, duration, priority}
POST /api/show {app_id, duration}
```

### Frame Endpoint (for LED clients)
```
GET /frame
Headers: X-Frame-Width, X-Frame-Height, X-Frame-Count, X-Frame-Delay-Ms, X-Dwell-Secs, X-Brightness, X-App-Name
Body: Raw RGB bytes
```

## Hardware Support

- **Interstate 75W** — Primary target, uses `/frame` endpoint
- **Tidbyt** — WebP output (coming soon)
- **Any HUB75 matrix** — With appropriate controller

## Configuration

| Option | Description |
|--------|-------------|
| `log_level` | Logging verbosity (trace/debug/info/warning/error/fatal) |

## Port

The add-on runs on port **8176** (internal and external).

## Support

- [GitHub Issues](https://github.com/johnfernkas/mosaic-addon/issues)
- [Documentation](https://github.com/johnfernkas/mosaic-addon)
