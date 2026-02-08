package pixlet

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"tidbyt.dev/pixlet/render"
	"tidbyt.dev/pixlet/runtime"
)

// Frame represents a rendered frame with metadata
type Frame struct {
	Images   []image.Image
	DelayMs  int
	MaxAge   int
	AppName  string
	Width    int
	Height   int
}

// Renderer handles Pixlet app rendering
type Renderer struct {
	width  int
	height int
	timeout time.Duration
}

// NewRenderer creates a new Pixlet renderer
func NewRenderer(width, height int) *Renderer {
	return &Renderer{
		width:   width,
		height:  height,
		timeout: 30 * time.Second,
	}
}

// RenderApp renders a .star app file with the given config
func (r *Renderer) RenderApp(appPath string, config map[string]string) (*Frame, error) {
	// Read the app source
	src, err := os.ReadFile(appPath)
	if err != nil {
		return nil, fmt.Errorf("reading app file: %w", err)
	}

	// Get app ID from filename
	appID := filepath.Base(appPath)
	if ext := filepath.Ext(appID); ext != "" {
		appID = appID[:len(appID)-len(ext)]
	}

	// Create the applet
	applet, err := runtime.NewApplet(appID, src)
	if err != nil {
		return nil, fmt.Errorf("creating applet: %w", err)
	}

	// Run with timeout
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	roots, err := applet.RunWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("running applet: %w", err)
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("applet returned no roots")
	}

	// Paint the roots to images
	images := render.PaintRoots(true, roots...)

	// Get delay from first root
	delayMs := 50
	maxAge := 0
	if len(roots) > 0 {
		if roots[0].Delay > 0 {
			delayMs = int(roots[0].Delay)
		}
		if roots[0].MaxAge > 0 {
			maxAge = int(roots[0].MaxAge)
		}
	}

	frame := &Frame{
		Images:  images,
		DelayMs: delayMs,
		MaxAge:  maxAge,
		AppName: appID,
		Width:   r.width,
		Height:  r.height,
	}

	// Default delay if not set
	if frame.DelayMs == 0 {
		frame.DelayMs = 50
	}

	return frame, nil
}

// RenderAppFromSource renders a .star app from source code
func (r *Renderer) RenderAppFromSource(appID string, src []byte, config map[string]string) (*Frame, error) {
	applet, err := runtime.NewApplet(appID, src)
	if err != nil {
		return nil, fmt.Errorf("creating applet: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	roots, err := applet.RunWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("running applet: %w", err)
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("applet returned no roots")
	}

	images := render.PaintRoots(true, roots...)

	delayMs := 50
	maxAge := 0
	if len(roots) > 0 {
		if roots[0].Delay > 0 {
			delayMs = int(roots[0].Delay)
		}
		if roots[0].MaxAge > 0 {
			maxAge = int(roots[0].MaxAge)
		}
	}

	frame := &Frame{
		Images:  images,
		DelayMs: delayMs,
		MaxAge:  maxAge,
		AppName: appID,
		Width:   r.width,
		Height:  r.height,
	}

	return frame, nil
}

// ImagesToRGB converts images to raw RGB bytes
func ImagesToRGB(images []image.Image) []byte {
	if len(images) == 0 {
		return nil
	}

	// Get dimensions from first image
	bounds := images[0].Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Allocate buffer for all frames
	frameSize := width * height * 3
	pixels := make([]byte, frameSize*len(images))

	for frameIdx, img := range images {
		offset := frameIdx * frameSize
		
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				idx := offset + (y*width+x)*3
				
				// Convert from 16-bit to 8-bit color
				pixels[idx] = byte(r >> 8)
				pixels[idx+1] = byte(g >> 8)
				pixels[idx+2] = byte(b >> 8)
			}
		}
	}

	return pixels
}

// GetSchema extracts the schema from a .star app
func (r *Renderer) GetSchema(appPath string) ([]byte, error) {
	src, err := os.ReadFile(appPath)
	if err != nil {
		return nil, fmt.Errorf("reading app file: %w", err)
	}

	appID := filepath.Base(appPath)
	if ext := filepath.Ext(appID); ext != "" {
		appID = appID[:len(appID)-len(ext)]
	}

	applet, err := runtime.NewApplet(appID, src)
	if err != nil {
		return nil, fmt.Errorf("creating applet: %w", err)
	}

	return applet.SchemaJSON, nil
}
