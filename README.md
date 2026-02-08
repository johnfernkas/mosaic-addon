# Mosaic LED Display Server

A Home Assistant add-on that provides a complete server for controlling one or more LED matrix displays with Tidbyt-compatible apps.

## Features

- **Multi-display support** — Control multiple LED matrices independently
- **Tidbyt-compatible apps** — Run 887+ community apps or create custom ones
- **Web dashboard** — Beautiful UI for control and app management
- **App rotation** — Auto-rotate between apps with configurable dwell time
- **Rich notifications** — Push text notifications with priorities
- **App installation** — Browse and install apps from the community index
- **Custom apps** — Upload and run your own .star apps
- **Persistent storage** — All config and installed apps saved locally

## Installation

### Home Assistant Add-ons

1. Go to **Settings → Add-ons → Add-on Store** in Home Assistant
2. Click the ⋮ menu → **Repositories**  
3. Add: `https://github.com/johnfernkas/mosaic-addon`
4. Find "Mosaic" in the list and click **Install**
5. Go to the **Logs** tab and start the add-on
6. Click **Open Web UI** to access the dashboard

### Standalone Deployment

Requires Go 1.20+:

```bash
cd mosaic-addon/mosaic
go build -o mosaic ./cmd
./mosaic
```

Access the dashboard at `http://localhost:8075`

## Configuration

### Web Dashboard

The dashboard (available at the add-on's Web UI) provides all controls:

- **Display Preview** — Real-time rendering of the current app
- **Display Controls** — Power, brightness, rotation toggle, skip button
- **Apps in Rotation** — Manage which apps display in sequence
- **Installed Apps** — View installed apps, configure them, add/remove from rotation
- **Community Apps** — Browse and install from 887 community apps
- **Upload Custom App** — Upload your own `.star` apps
- **Configure** — Edit app configuration values

### App Configuration

If an app has configuration options:

1. Go to **App Browser → Configure** tab
2. Select the app from the dropdown
3. Edit the configuration fields
4. Click **Save Configuration**

Configuration is automatically applied to the app when it runs.

## API Reference

### Displays API

#### List all displays
```
GET /api/displays
```

Response: Array of display objects
```json
[
  {
    "id": "display_1",
    "name": "Living Room",
    "width": 64,
    "height": 32,
    "brightness": 80,
    "power": true,
    "rotation_enabled": true,
    "current_app": "weather"
  }
]
```

#### Register a display
```
POST /api/displays
```

Request body:
```json
{
  "id": "display_1",
  "name": "Living Room",
  "width": 64,
  "height": 32
}
```

#### Get display info
```
GET /api/displays/{displayID}
```

#### Update display
```
PUT /api/displays/{displayID}
```

Request: Any of `brightness` (0-100), `power` (boolean)

#### Set brightness
```
PUT /api/displays/{displayID}/brightness
```

Request: `{"brightness": 80}`

#### Set power state
```
PUT /api/displays/{displayID}/power
```

Request: `{"power": true}`

#### Skip to next app
```
POST /api/displays/{displayID}/skip
```

### Rotation API

#### Get rotation config
```
GET /api/displays/{displayID}/rotation
```

Response:
```json
{
  "enabled": true,
  "apps": [
    {"id": "weather", "name": "Weather"}
  ]
}
```

#### Set rotation enabled
```
PUT /api/displays/{displayID}/rotation
```

Request: `{"enabled": true}`

#### Add app to rotation
```
POST /api/displays/{displayID}/rotation/apps
```

Request: `{"app_id": "weather"}`

#### Remove app from rotation
```
DELETE /api/displays/{displayID}/rotation/apps/{appID}
```

### Apps API

#### List installed apps
```
GET /api/apps
```

Response: Array of app objects
```json
[
  {
    "id": "weather",
    "name": "Weather",
    "summary": "Weather forecast",
    "author": "Community",
    "source": "community",
    "config": {},
    "installed": "2024-02-08T12:00:00Z"
  }
]
```

#### Get app info
```
GET /api/apps/{appID}
```

#### Save app configuration
```
PUT /api/apps/{appID}/config
```

Request: JSON object with config keys/values
```json
{
  "location": "New York",
  "units": "celsius"
}
```

#### Install app
```
POST /api/apps/install
```

Request: `{"app_id": "weather"}`

#### Upload custom app
```
POST /api/apps/upload
```

Request:
```json
{
  "id": "my_app",
  "name": "My Custom App",
  "source": "<base64-encoded .star file contents>"
}
```

#### Uninstall app
```
DELETE /api/apps/{appID}
```

#### List community apps
```
GET /api/apps/community
```

Query parameters:
- `limit` (optional, default 100) — Max results
- `offset` (optional, default 0) — Pagination offset

#### Search community apps
```
GET /api/apps/community/search?q=weather
```

### Notifications API

#### Push text notification
```
POST /api/notify
```

Request:
```json
{
  "text": "Hello!",
  "duration": 10,
  "color": "#FFFFFF",
  "priority": "normal",
  "display_id": "display_1"
}
```

Parameters:
- `text` — Message to display
- `duration` — How many seconds to show (default: 10)
- `color` — Hex color (default: #FFFFFF)
- `priority` — "low", "normal", "high", or "sticky" (default: normal)
- `display_id` — Target display (optional; uses first display if omitted)

#### Show app temporarily
```
POST /api/show
```

Request:
```json
{
  "app_id": "weather",
  "duration": 30,
  "display_id": "display_1"
}
```

### Frame Data API

#### Get raw frame data (for LED matrix clients)
```
GET /frame?display={displayID}
```

Returns raw RGB pixel data as binary. Headers include:
- `X-Frame-Width` — Display width in pixels
- `X-Frame-Height` — Display height in pixels
- `X-Frame-Count` — Number of animation frames
- `X-Frame-Delay-Ms` — Milliseconds between frames
- `X-Dwell-Secs` — Seconds to show this app
- `X-Brightness` — Brightness percentage
- `X-App-Name` — Current app name

## Usage Examples

### Use with Home Assistant Integration

1. Install [mosaic-integration](https://github.com/johnfernkas/mosaic-integration)
2. Add in Home Assistant config:
   ```yaml
   mosaic:
     url: http://localhost:8075
   ```

3. Use the `mosaic.push_text` service:
   ```yaml
   service: mosaic.push_text
   data:
     text: "Hello from Home Assistant!"
     duration: 10
     color: "#00FF00"
     display_id: "display_1"
   ```

### Push notifications via curl
```bash
curl -X POST http://localhost:8075/api/notify \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Temperature: 72°F",
    "duration": 5,
    "display_id": "display_1"
  }'
```

### Display a specific app
```bash
curl -X POST http://localhost:8075/api/show \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "weather",
    "duration": 30,
    "display_id": "display_1"
  }'
```

### Set brightness
```bash
curl -X PUT http://localhost:8075/api/displays/display_1/brightness \
  -H "Content-Type: application/json" \
  -d '{"brightness": 50}'
```

## Local Development

```bash
cd mosaic-addon/mosaic
docker compose up --build
# Open http://localhost:8075
```

Edit files and restart the container to see changes.

## Architecture

- **Go server** (`internal/server`) — REST API and WebSocket support
- **App repository** (`internal/apps`) — Install, manage, render Tidbyt apps via Pixlet
- **Display drivers** (`internal/display`) — Hardware abstraction for LED matrices
- **Dashboard** (`internal/server/dashboard.go`) — Single-page web app
- **Web UI** — Modern dark theme with real-time preview

## Troubleshooting

### Apps not installing
- Check add-on logs: **Add-ons → Mosaic → Logs**
- Ensure community index loads: Check dashboard for app count
- Try uploading a custom app to test the system

### Display not showing anything
- Verify display is registered: Check `/api/displays`
- Check LED matrix hardware connections
- Try the boot animation by refreshing dashboard

### Push_text not working for multi-display
- Specify `display_id` in the request
- If omitted, uses the first registered display

## License

MIT
