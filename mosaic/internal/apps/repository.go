package apps

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/johnfernkas/mosaic-addon/internal/pixlet"
)

// App represents an installed app
type App struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Summary     string            `json:"summary" yaml:"summary"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Author      string            `json:"author" yaml:"author"`
	Category    string            `json:"category,omitempty" yaml:"category,omitempty"`
	Path        string            `json:"path" yaml:"path"`
	Config      map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
	SchemaJSON  []byte            `json:"schema_json,omitempty" yaml:"-"`
	Source      string            `json:"source" yaml:"source"` // "local", "community", "custom"
	Installed   time.Time         `json:"installed" yaml:"installed"`
}

// CommunityApp represents an app from the community index
type CommunityApp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Author   string `json:"author"`
	Category string `json:"category"`
	FileName string `json:"file_name,omitempty"`
}

// CommunityIndex represents the bundled community apps index
type CommunityIndex struct {
	Updated time.Time      `json:"updated"`
	Count   int            `json:"count"`
	Apps    []CommunityApp `json:"apps"`
}

// Repository manages installed apps
type Repository struct {
	mu sync.RWMutex

	dataDir        string
	appsDir        string
	communityIndex *CommunityIndex
	installed      map[string]*App
}

// NewRepository creates a new app repository
func NewRepository(dataDir string) (*Repository, error) {
	appsDir := filepath.Join(dataDir, "apps")
	
	// Create directories if needed
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating apps directory: %w", err)
	}

	r := &Repository{
		dataDir:   dataDir,
		appsDir:   appsDir,
		installed: make(map[string]*App),
	}

	// Load community index
	if err := r.loadCommunityIndex(); err != nil {
		log.Printf("Warning: failed to load community index: %v", err)
	}

	// Discover installed apps
	if err := r.discoverApps(); err != nil {
		log.Printf("Warning: failed to discover apps: %v", err)
	}

	return r, nil
}

// List returns all installed apps
func (r *Repository) List() []*App {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apps := make([]*App, 0, len(r.installed))
	for _, app := range r.installed {
		apps = append(apps, app)
	}
	return apps
}

// Get returns an installed app by ID
func (r *Repository) Get(id string) *App {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.installed[id]
}

// GetPath returns the path to an app's .star file
func (r *Repository) GetPath(id string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if app, ok := r.installed[id]; ok {
		return app.Path
	}
	return ""
}

// Install downloads and installs an app from the community
func (r *Repository) Install(id string) (*App, error) {
	// Check if already installed
	if existing := r.Get(id); existing != nil {
		return existing, nil
	}

	// Find in community index
	community := r.GetCommunityApp(id)
	if community == nil {
		return nil, fmt.Errorf("app %q not found in community index", id)
	}

	// First, get the directory listing to find the .star file
	listURL := fmt.Sprintf("https://api.github.com/repos/tidbyt/community/contents/apps/%s", id)
	listResp, err := http.Get(listURL)
	if err != nil {
		return nil, fmt.Errorf("listing app directory: %w", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("app directory not found: %s", listResp.Status)
	}

	var files []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("parsing directory listing: %w", err)
	}

	// Find the .star file
	var starFile string
	for _, f := range files {
		if strings.HasSuffix(f.Name, ".star") {
			starFile = f.Name
			break
		}
	}
	if starFile == "" {
		return nil, fmt.Errorf("no .star file found in app %s", id)
	}

	// Download the .star file
	url := fmt.Sprintf("https://raw.githubusercontent.com/tidbyt/community/main/apps/%s/%s", id, starFile)
	log.Printf("Downloading app from: %s", url)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading app: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}

	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading app source: %w", err)
	}

	// Create app directory
	appDir := filepath.Join(r.appsDir, id)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, fmt.Errorf("creating app directory: %w", err)
	}

	// Write star file
	starPath := filepath.Join(appDir, id+".star")
	if err := os.WriteFile(starPath, source, 0644); err != nil {
		return nil, fmt.Errorf("writing app file: %w", err)
	}

	// Extract schema from app
	renderer := pixlet.NewRenderer(64, 32)
	schemaJSON, err := renderer.GetSchema(starPath)
	if err != nil {
		log.Printf("Warning: could not extract schema for %s: %v", id, err)
		// Continue without schema - not all apps have one
	}

	// Create app entry
	app := &App{
		ID:         id,
		Name:       community.Name,
		Summary:    community.Summary,
		Author:     community.Author,
		Category:   community.Category,
		Path:       starPath,
		SchemaJSON: schemaJSON,
		Source:     "community",
		Installed:  time.Now(),
	}

	// Write metadata
	metaPath := filepath.Join(appDir, "app.json")
	metaData, _ := json.MarshalIndent(app, "", "  ")
	os.WriteFile(metaPath, metaData, 0644)

	r.mu.Lock()
	r.installed[id] = app
	r.mu.Unlock()

	log.Printf("Installed community app: %s", id)
	return app, nil
}

// InstallFromSource installs an app from provided source code
func (r *Repository) InstallFromSource(id, name string, source []byte) (*App, error) {
	// Remove existing if present
	if existing := r.Get(id); existing != nil {
		r.Uninstall(id)
	}

	// Create app directory
	appDir := filepath.Join(r.appsDir, id)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, fmt.Errorf("creating app directory: %w", err)
	}

	// Write star file
	starPath := filepath.Join(appDir, id+".star")
	if err := os.WriteFile(starPath, source, 0644); err != nil {
		return nil, fmt.Errorf("writing app file: %w", err)
	}

	// Create app entry
	if name == "" {
		name = id
	}
	app := &App{
		ID:        id,
		Name:      name,
		Summary:   "Custom uploaded app",
		Path:      starPath,
		Source:    "custom",
		Installed: time.Now(),
	}

	// Write metadata
	metaPath := filepath.Join(appDir, "app.json")
	metaData, _ := json.MarshalIndent(app, "", "  ")
	os.WriteFile(metaPath, metaData, 0644)

	// Add to installed
	r.mu.Lock()
	r.installed[id] = app
	r.mu.Unlock()

	log.Printf("Installed app: %s", id)
	return app, nil
}

// Uninstall removes an installed app
func (r *Repository) Uninstall(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, ok := r.installed[id]
	if !ok {
		return fmt.Errorf("app %q not installed", id)
	}

	// Remove directory
	appDir := filepath.Dir(app.Path)
	if err := os.RemoveAll(appDir); err != nil {
		return fmt.Errorf("removing app directory: %w", err)
	}

	delete(r.installed, id)
	log.Printf("Uninstalled app: %s", id)
	return nil
}

// SaveConfig saves config for an app
func (r *Repository) SaveConfig(id string, config map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, ok := r.installed[id]
	if !ok {
		return fmt.Errorf("app %q not installed", id)
	}

	app.Config = config

	// Write config file
	configPath := filepath.Join(filepath.Dir(app.Path), "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// ListCommunity returns the community apps index
func (r *Repository) ListCommunity() []CommunityApp {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.communityIndex == nil || r.communityIndex.Apps == nil {
		return []CommunityApp{}  // Return empty slice, not nil
	}
	return r.communityIndex.Apps
}

// GetCommunityApp returns a community app by ID
func (r *Repository) GetCommunityApp(id string) *CommunityApp {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.communityIndex == nil {
		return nil
	}

	for _, app := range r.communityIndex.Apps {
		if app.ID == id {
			return &app
		}
	}
	return nil
}

// SearchCommunity searches the community index
func (r *Repository) SearchCommunity(query string) []CommunityApp {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.communityIndex == nil {
		return []CommunityApp{}  // Return empty slice, not nil
	}

	query = strings.ToLower(query)
	var results []CommunityApp

	for _, app := range r.communityIndex.Apps {
		if strings.Contains(strings.ToLower(app.Name), query) ||
			strings.Contains(strings.ToLower(app.Summary), query) ||
			strings.Contains(strings.ToLower(app.ID), query) {
			results = append(results, app)
		}
	}

	return results
}

func (r *Repository) loadCommunityIndex() error {
	// Try bundled index first (in /app/data, not /data which is a volume mount)
	paths := []string{
		"/app/data/community-apps.json",  // Bundled in Docker image
		filepath.Join(r.dataDir, "community-apps.json"),  // User-provided
	}
	
	var data []byte
	var err error
	var loadedPath string
	
	for _, path := range paths {
		log.Printf("Trying community index: %s", path)
		data, err = os.ReadFile(path)
		if err == nil {
			loadedPath = path
			break
		}
	}
	
	if err != nil {
		log.Printf("Failed to read community index from any location")
		// Generate minimal index for testing
		r.communityIndex = &CommunityIndex{
			Updated: time.Now(),
			Count:   0,
			Apps:    []CommunityApp{},
		}
		log.Println("No community index found, starting with empty index")
		return nil
	}
	log.Printf("Read %d bytes from community index at %s", len(data), loadedPath)

	var index CommunityIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("parsing community index: %w", err)
	}

	r.communityIndex = &index
	log.Printf("Loaded community index: %d apps", index.Count)
	return nil
}

func (r *Repository) discoverApps() error {
	entries, err := os.ReadDir(r.appsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appID := entry.Name()
		appDir := filepath.Join(r.appsDir, appID)

		// Look for .star file
		starPath := filepath.Join(appDir, appID+".star")
		if _, err := os.Stat(starPath); os.IsNotExist(err) {
			// Try any .star file
			files, _ := filepath.Glob(filepath.Join(appDir, "*.star"))
			if len(files) > 0 {
				starPath = files[0]
			} else {
				continue
			}
		}

		// Load metadata if available
		app := &App{
			ID:     appID,
			Name:   appID,
			Path:   starPath,
			Source: "local",
		}

		metaPath := filepath.Join(appDir, "app.json")
		if data, err := os.ReadFile(metaPath); err == nil {
			json.Unmarshal(data, app)
		}

		// Load config if available
		configPath := filepath.Join(appDir, "config.json")
		if data, err := os.ReadFile(configPath); err == nil {
			var config map[string]string
			if json.Unmarshal(data, &config) == nil {
				app.Config = config
			}
		}

		// Parse app header for metadata
		if source, err := os.ReadFile(starPath); err == nil {
			r.parseAppHeader(app, source)
		}

		// Extract schema if not already loaded
		if len(app.SchemaJSON) == 0 {
			renderer := pixlet.NewRenderer(64, 32)
			if schemaJSON, err := renderer.GetSchema(starPath); err == nil && len(schemaJSON) > 0 {
				app.SchemaJSON = schemaJSON
				log.Printf("Extracted schema for app: %s", appID)
			}
		}

		r.installed[appID] = app
		log.Printf("Discovered app: %s (%s)", app.Name, appID)
	}

	return nil
}

func (r *Repository) parseAppHeader(app *App, source []byte) {
	lines := strings.Split(string(source), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `"""`) && !strings.HasPrefix(line, "#") {
			if line != "" && !strings.HasPrefix(line, "load(") {
				break // Past header
			}
			continue
		}

		// Parse docstring format: """Key: Value"""
		if strings.HasPrefix(line, `"""`) {
			continue
		}

		// Parse header comments
		line = strings.TrimPrefix(line, "# ")
		line = strings.TrimPrefix(line, "#")
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "applet", "name":
			if app.Name == app.ID {
				app.Name = value
			}
		case "summary":
			app.Summary = value
		case "description":
			app.Description = value
		case "author":
			app.Author = value
		}
	}
}
