package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/johnfernkas/mosaic-addon/internal/rotation"
)

// Config represents the Mosaic server configuration
type Config struct {
	mu sync.RWMutex

	path string

	// Server settings
	Port     string `json:"port"`
	LogLevel string `json:"log_level"`

	// Display defaults
	DefaultWidth  int `json:"default_width"`
	DefaultHeight int `json:"default_height"`
	DefaultDwell  int `json:"default_dwell_ms"`
	Brightness    int `json:"brightness"`

	// Rotation settings
	RotationEnabled bool              `json:"rotation_enabled"`
	Apps            []rotation.AppEntry `json:"apps"`

	// Display power
	PowerOn bool `json:"power_on"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Port:            "8176",
		LogLevel:        "info",
		DefaultWidth:    64,
		DefaultHeight:   32,
		DefaultDwell:    10000, // 10 seconds
		Brightness:      80,
		RotationEnabled: true,
		PowerOn:         true,
		Apps:            []rotation.AppEntry{},
	}
}

// Load loads config from a file
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			if err := cfg.Save(); err != nil {
				return nil, fmt.Errorf("creating default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// Save saves the config to disk
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.path == "" {
		return fmt.Errorf("config path not set")
	}

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// SetBrightness updates brightness and saves
func (c *Config) SetBrightness(brightness int) error {
	c.mu.Lock()
	c.Brightness = brightness
	c.mu.Unlock()
	return c.Save()
}

// SetPower updates power state and saves
func (c *Config) SetPower(on bool) error {
	c.mu.Lock()
	c.PowerOn = on
	c.mu.Unlock()
	return c.Save()
}

// SetRotationEnabled updates rotation state and saves
func (c *Config) SetRotationEnabled(enabled bool) error {
	c.mu.Lock()
	c.RotationEnabled = enabled
	c.mu.Unlock()
	return c.Save()
}

// SetApps updates the app rotation list and saves
func (c *Config) SetApps(apps []rotation.AppEntry) error {
	c.mu.Lock()
	c.Apps = apps
	c.mu.Unlock()
	return c.Save()
}

// GetApps returns a copy of the apps list
func (c *Config) GetApps() []rotation.AppEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]rotation.AppEntry, len(c.Apps))
	copy(result, c.Apps)
	return result
}
