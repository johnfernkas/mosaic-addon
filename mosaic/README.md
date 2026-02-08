# Mosaic

Home Assistant add-on for LED matrix displays. Run Tidbyt-compatible apps on DIY hardware.

## Features

- **Tidbyt-compatible** — Run apps from the [Tidbyt community repo](https://github.com/tidbyt/community)
- **DIY hardware support** — Interstate 75W, ESP32+HUB75, and more
- **Native HA integration** — Brightness control, automations, notifications
- **Multi-display** — Support multiple matrices with independent rotations

## Supported Hardware

| Device | Format | Status |
|--------|--------|--------|
| Pimoroni Interstate 75W | Raw RGB | ✅ Primary target |
| Tidbyt Gen1/Gen2 | WebP | 🚧 Planned |
| ESP32 + HUB75 | Raw RGB | 🚧 Planned |

## Installation

### As a Local Add-on

1. Copy this repository to `/addons/mosaic-addon` on your Home Assistant host
2. Go to **Settings → Add-ons → Add-on Store**
3. Click the menu (⋮) → **Check for updates**
4. Find "Mosaic" in Local add-ons and install

### From Repository (coming soon)

Add this repository URL to your Home Assistant add-on store:
```
https://github.com/johnfernkas/mosaic-addon
```

## Configuration

```yaml
log_level: info
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Service info |
| `/api/status` | GET | Health check |
| `/frame` | GET | Raw RGB frame for LED clients |
| `/frame/preview` | GET | PNG preview (browser) |

## Development

### With Docker (recommended)

```bash
# Build and run
docker compose up --build

# Access the dashboard
open http://localhost:8075
```

### Without Docker

```bash
# Install dependencies (macOS)
brew install go libwebp pango cairo

# Download Go modules
go mod tidy

# Build
go build -o mosaic ./cmd/mosaic

# Run
./mosaic
```

### Testing

```bash
# Check status
curl http://localhost:8075/api/status

# Get raw frame (for LED clients)
curl -I http://localhost:8075/frame
```

## License

MIT
