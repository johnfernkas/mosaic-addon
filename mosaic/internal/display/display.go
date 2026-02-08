package display

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/johnfernkas/mosaic-addon/internal/apps"
	"github.com/johnfernkas/mosaic-addon/internal/config"
	"github.com/johnfernkas/mosaic-addon/internal/pixlet"
	"github.com/johnfernkas/mosaic-addon/internal/rotation"
)

// Status represents display connection status
type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
)

// FrameData contains the current frame for clients
type FrameData struct {
	Pixels     []byte
	Width      int
	Height     int
	FrameCount int
	DelayMs    int
	DwellSecs  int
	Brightness int
	AppName    string
	UpdatedAt  time.Time
}

// Display represents a single LED matrix display
type Display struct {
	mu sync.RWMutex

	ID         string `json:"id"`
	Name       string `json:"name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	ClientType string `json:"client_type"`

	config   *config.Config
	apps     *apps.Repository
	rotation *rotation.Manager
	renderer *pixlet.Renderer

	// Current frame state
	currentFrame *FrameData

	// Control
	stopCh chan struct{}
}

// NewDisplay creates a new display manager
func NewDisplay(id, name string, width, height int, cfg *config.Config, appRepo *apps.Repository) *Display {
	d := &Display{
		ID:       id,
		Name:     name,
		Width:    width,
		Height:   height,
		config:   cfg,
		apps:     appRepo,
		rotation: rotation.NewManager(time.Duration(cfg.DefaultDwell) * time.Millisecond),
		renderer: pixlet.NewRenderer(width, height),
		stopCh:   make(chan struct{}),
	}

	// Set initial state from config
	d.rotation.SetEnabled(cfg.RotationEnabled)
	d.rotation.SetBrightness(cfg.Brightness)

	// Load apps from config
	d.rotation.SetApps(cfg.GetApps())

	// Set callback for when rotation advances
	d.rotation.OnAdvance(func(app rotation.AppEntry) {
		d.renderApp(app)
	})

	// Render initial frame
	d.renderStartupScreen()

	return d
}

// Start begins the rotation loop
func (d *Display) Start() {
	go d.rotation.Run()

	// Trigger initial render if we have apps
	if app := d.rotation.CurrentApp(); app != nil {
		d.renderApp(*app)
	}
}

// Stop stops the display manager
func (d *Display) Stop() {
	d.rotation.Stop()
	close(d.stopCh)
}

// GetFrame returns the current frame data
func (d *Display) GetFrame() *FrameData {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.currentFrame == nil {
		return d.generateTestPattern()
	}

	// Return copy
	return &FrameData{
		Pixels:     d.currentFrame.Pixels,
		Width:      d.currentFrame.Width,
		Height:     d.currentFrame.Height,
		FrameCount: d.currentFrame.FrameCount,
		DelayMs:    d.currentFrame.DelayMs,
		DwellSecs:  d.config.DefaultDwell / 1000,
		Brightness: d.rotation.GetBrightness(),
		AppName:    d.currentFrame.AppName,
		UpdatedAt:  d.currentFrame.UpdatedAt,
	}
}

// SetBrightness updates display brightness
func (d *Display) SetBrightness(brightness int) error {
	d.rotation.SetBrightness(brightness)
	return d.config.SetBrightness(brightness)
}

// GetBrightness returns current brightness
func (d *Display) GetBrightness() int {
	return d.rotation.GetBrightness()
}

// SetPower turns the display on/off
func (d *Display) SetPower(on bool) error {
	if !on {
		// Turning off: stop rotation and render blank
		d.rotation.SetEnabled(false)
		d.renderBlankScreen()
	} else {
		// Turning on: restore rotation based on config and render current app
		d.rotation.SetEnabled(d.config.RotationEnabled)
		// Force a re-render of current app
		if app := d.rotation.CurrentApp(); app != nil {
			d.renderApp(*app)
		} else {
			// No apps in rotation, render startup screen
			d.renderStartupScreen()
		}
	}
	return d.config.SetPower(on)
}

// IsPowerOn returns power state
func (d *Display) IsPowerOn() bool {
	return d.config.PowerOn
}

// SetRotationEnabled enables/disables rotation
func (d *Display) SetRotationEnabled(enabled bool) error {
	d.rotation.SetEnabled(enabled)
	return d.config.SetRotationEnabled(enabled)
}

// IsRotationEnabled returns rotation state
func (d *Display) IsRotationEnabled() bool {
	return d.rotation.IsEnabled()
}

// Skip advances to the next app
func (d *Display) Skip() {
	d.rotation.Skip()
}

// GetRotationApps returns apps in rotation
func (d *Display) GetRotationApps() []rotation.AppEntry {
	return d.rotation.GetApps()
}

// SetRotationApps sets apps in rotation
func (d *Display) SetRotationApps(apps []rotation.AppEntry) error {
	d.rotation.SetApps(apps)
	return d.config.SetApps(apps)
}

// AddToRotation adds an app to rotation
func (d *Display) AddToRotation(appID string) error {
	app := d.apps.Get(appID)
	if app == nil {
		return fmt.Errorf("app %q not installed", appID)
	}

	entry := rotation.AppEntry{
		ID:      appID,
		Name:    app.Name,
		Path:    app.Path,
		Config:  app.Config,
		Enabled: true,
	}

	d.rotation.AddApp(entry)

	// Update config
	return d.config.SetApps(d.rotation.GetApps())
}

// RemoveFromRotation removes an app from rotation
func (d *Display) RemoveFromRotation(appID string) error {
	if !d.rotation.RemoveApp(appID) {
		return fmt.Errorf("app %q not in rotation", appID)
	}
	return d.config.SetApps(d.rotation.GetApps())
}

// ShowApp temporarily shows a specific app
func (d *Display) ShowApp(appID string, durationSecs int) error {
	app := d.apps.Get(appID)
	if app == nil {
		return fmt.Errorf("app %q not installed", appID)
	}

	// Pause rotation temporarily
	wasEnabled := d.rotation.IsEnabled()
	d.rotation.SetEnabled(false)

	// Render the app
	entry := rotation.AppEntry{
		ID:     appID,
		Name:   app.Name,
		Path:   app.Path,
		Config: app.Config,
	}
	d.renderApp(entry)

	// Resume after duration
	if durationSecs > 0 {
		go func() {
			time.Sleep(time.Duration(durationSecs) * time.Second)
			d.rotation.SetEnabled(wasEnabled)
		}()
	}

	return nil
}

// RenderSource renders inline Starlark source
func (d *Display) RenderSource(appID string, source []byte, config map[string]string) error {
	frame, err := d.renderer.RenderAppFromSource(appID, source, config)
	if err != nil {
		return err
	}

	pixels := pixlet.ImagesToRGB(frame.Images)
	d.setFrame(frame, pixels)
	return nil
}

// PushText displays text notification
func (d *Display) PushText(text string, color string, durationSecs int, priority rotation.Priority) {
	// Create a simple text app source
	if color == "" {
		color = "#fff"
	}

	source := fmt.Sprintf(`
load("render.star", "render")

def main():
    return render.Root(
        child = render.Box(
            width = 64,
            height = 32,
            color = "#000",
            child = render.WrappedText(
                content = %q,
                font = "tom-thumb",
                color = %q,
                align = "center",
            ),
        ),
    )
`, text, color)

	notification := rotation.Notification{
		ID:       fmt.Sprintf("text-%d", time.Now().UnixNano()),
		Text:     text,
		Source:   source,
		Duration: time.Duration(durationSecs) * time.Second,
		Priority: priority,
	}

	d.rotation.PushNotification(notification)

	// For now, render immediately
	d.RenderSource("notification", []byte(source), nil)
}

// renderApp renders an app and updates the frame
func (d *Display) renderApp(app rotation.AppEntry) {
	frame, err := d.renderer.RenderApp(app.Path, app.Config)
	if err != nil {
		log.Printf("Error rendering app %s: %v", app.ID, err)
		d.renderErrorScreen(app.Name, err)
		return
	}

	pixels := pixlet.ImagesToRGB(frame.Images)
	d.setFrame(frame, pixels)
}

func (d *Display) setFrame(frame *pixlet.Frame, pixels []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	frameCount := 1
	if len(frame.Images) > 0 {
		frameCount = len(frame.Images)
	}

	d.currentFrame = &FrameData{
		Pixels:     pixels,
		Width:      d.Width,
		Height:     d.Height,
		FrameCount: frameCount,
		DelayMs:    frame.DelayMs,
		DwellSecs:  d.config.DefaultDwell / 1000,
		Brightness: d.rotation.GetBrightness(),
		AppName:    frame.AppName,
		UpdatedAt:  time.Now(),
	}
}

func (d *Display) renderStartupScreen() {
	source := `
load("render.star", "render")

def main():
    return render.Root(
        child = render.Box(
            width = 64,
            height = 32,
            color = "#111",
            child = render.Column(
                expanded = True,
                main_align = "center",
                cross_align = "center",
                children = [
                    render.Text(
                        content = "MOSAIC",
                        font = "6x13",
                        color = "#0ff",
                    ),
                    render.Text(
                        content = "Ready",
                        font = "tom-thumb",
                        color = "#888",
                    ),
                ],
            ),
        ),
    )
`
	d.RenderSource("startup", []byte(source), nil)
}

func (d *Display) renderBlankScreen() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// All black pixels
	pixels := make([]byte, d.Width*d.Height*3)
	d.currentFrame = &FrameData{
		Pixels:     pixels,
		Width:      d.Width,
		Height:     d.Height,
		FrameCount: 1,
		DelayMs:    50,
		DwellSecs:  0,
		Brightness: 0,
		AppName:    "off",
		UpdatedAt:  time.Now(),
	}
}

func (d *Display) renderErrorScreen(appName string, err error) {
	source := fmt.Sprintf(`
load("render.star", "render")

def main():
    return render.Root(
        child = render.Box(
            width = 64,
            height = 32,
            color = "#300",
            child = render.Column(
                expanded = True,
                main_align = "center",
                cross_align = "center",
                children = [
                    render.Text(
                        content = "ERROR",
                        font = "tom-thumb",
                        color = "#f00",
                    ),
                    render.Marquee(
                        width = 60,
                        child = render.Text(
                            content = %q,
                            font = "tom-thumb",
                            color = "#faa",
                        ),
                    ),
                ],
            ),
        ),
    )
`, appName)
	d.RenderSource("error", []byte(source), nil)
}

func (d *Display) generateTestPattern() *FrameData {
	pixels := make([]byte, d.Width*d.Height*3)

	for y := 0; y < d.Height; y++ {
		for x := 0; x < d.Width; x++ {
			idx := (y*d.Width + x) * 3
			r := byte((x * 255) / d.Width)
			g := byte((y * 255) / d.Height)
			b := byte(128)

			pixels[idx] = r
			pixels[idx+1] = g
			pixels[idx+2] = b
		}
	}

	return &FrameData{
		Pixels:     pixels,
		Width:      d.Width,
		Height:     d.Height,
		FrameCount: 1,
		DelayMs:    50,
		DwellSecs:  0,
		Brightness: 80,
		AppName:    "test-pattern",
		UpdatedAt:  time.Now(),
	}
}
