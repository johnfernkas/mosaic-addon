package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/johnfernkas/mosaic-addon/internal/apps"
	"github.com/johnfernkas/mosaic-addon/internal/config"
	"github.com/johnfernkas/mosaic-addon/internal/display"
	"github.com/johnfernkas/mosaic-addon/internal/rotation"
)

const (
	DefaultWidth  = 64
	DefaultHeight = 32
)

type Server struct {
	router   *chi.Mux
	config   *config.Config
	apps     *apps.Repository
	displays map[string]*display.Display
}

// New creates a new Mosaic server
func New(dataDir string) (*Server, error) {
	// Load config
	cfg, err := config.Load(dataDir + "/config.json")
	if err != nil {
		return nil, err
	}

	// Create app repository
	appRepo, err := apps.NewRepository(dataDir)
	if err != nil {
		return nil, err
	}

	s := &Server{
		router:   chi.NewRouter(),
		config:   cfg,
		apps:     appRepo,
		displays: make(map[string]*display.Display),
	}

	s.setupRoutes()

	return s, nil
}

// NewSimple creates a server with defaults (for backwards compatibility)
func NewSimple() *Server {
	s, err := New("/data")
	if err != nil {
		log.Printf("Failed to create server with data dir, using minimal setup: %v", err)
		// Fallback to minimal server
		return newMinimalServer()
	}
	return s
}

func newMinimalServer() *Server {
	s := &Server{
		router:   chi.NewRouter(),
		displays: make(map[string]*display.Display),
	}
	s.setupRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RealIP)

	// Web dashboard
	s.router.Get("/", s.handleDashboard)

	// API endpoints
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/status", s.handleStatus)
		
		// Multi-display endpoints
		r.Get("/displays", s.handleListDisplays)
		r.Post("/displays", s.handleRegisterDisplay)
		r.Get("/displays/{displayID}", s.handleGetDisplayByID)
		r.Put("/displays/{displayID}", s.handleUpdateDisplay)
		r.Put("/displays/{displayID}/brightness", s.handleSetDisplayBrightness)
		r.Put("/displays/{displayID}/power", s.handleSetDisplayPower)
		r.Post("/displays/{displayID}/skip", s.handleDisplaySkip)
		r.Get("/displays/{displayID}/rotation", s.handleGetDisplayRotation)
		r.Put("/displays/{displayID}/rotation", s.handleSetDisplayRotation)
		r.Post("/displays/{displayID}/rotation/apps", s.handleAddToDisplayRotation)
		r.Delete("/displays/{displayID}/rotation/apps/{appID}", s.handleRemoveFromDisplayRotation)
		
		// Legacy single display endpoints (use default display)
		r.Get("/display", s.handleGetDisplay)
		r.Put("/display/brightness", s.handleSetBrightness)
		r.Put("/display/power", s.handleSetPower)
		r.Post("/display/skip", s.handleSkip)
		
		// Rotation (legacy - default display)
		r.Get("/rotation", s.handleGetRotation)
		r.Put("/rotation", s.handleSetRotation)
		r.Put("/rotation/enabled", s.handleSetRotationEnabled)
		r.Post("/rotation/apps", s.handleAddToRotation)
		r.Delete("/rotation/apps/{appID}", s.handleRemoveFromRotation)
		
		// Apps
		r.Get("/apps", s.handleListApps)
		r.Get("/apps/community", s.handleListCommunity)
		r.Get("/apps/community/search", s.handleSearchCommunity)
		r.Post("/apps/install", s.handleInstallApp)
		r.Post("/apps/upload", s.handleUploadApp)
		r.Delete("/apps/{appID}", s.handleUninstallApp)
		r.Get("/apps/{appID}", s.handleGetApp)
		r.Put("/apps/{appID}/config", s.handleSaveAppConfig)
		
		// Rendering
		r.Post("/render", s.handleRenderApp)
		r.Post("/notify", s.handlePushNotify)
		r.Post("/show", s.handleShowApp)
	})

	// Frame endpoint for LED matrix clients
	s.router.Get("/frame", s.handleFrame)
	s.router.Get("/frame/preview", s.handleFramePreview)
}

// handleDashboard serves the web UI
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := map[string]interface{}{
		"status":  "ok",
		"version": "0.2.0",
	}

	status["display_count"] = len(s.displays)
	
	// Show first display info if any exist
	for _, disp := range s.displays {
		frame := disp.GetFrame()
		status["current_app"] = frame.AppName
		status["brightness"] = frame.Brightness
		status["power"] = disp.IsPowerOn()
		status["rotation_enabled"] = disp.IsRotationEnabled()
		status["display"] = map[string]interface{}{
			"id":     disp.ID,
			"name":   disp.Name,
			"width":  disp.Width,
			"height": disp.Height,
		}
		break // Just show first one
	}

	json.NewEncoder(w).Encode(status)
}

// Multi-display handlers

func (s *Server) handleListDisplays(w http.ResponseWriter, r *http.Request) {
	displays := make([]map[string]interface{}, 0, len(s.displays))
	for id, disp := range s.displays {
		frame := disp.GetFrame()
		displays = append(displays, map[string]interface{}{
			"id":               id,
			"name":             disp.Name,
			"width":            disp.Width,
			"height":           disp.Height,
			"brightness":       frame.Brightness,
			"power":            disp.IsPowerOn(),
			"rotation_enabled": disp.IsRotationEnabled(),
			"current_app":      frame.AppName,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(displays)
}

func (s *Server) handleRegisterDisplay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "Display ID required", http.StatusBadRequest)
		return
	}

	// Check if display already exists
	if _, exists := s.displays[req.ID]; exists {
		// Just return success - display already registered
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "id": req.ID})
		return
	}

	// Create new display
	width := req.Width
	if width == 0 {
		width = DefaultWidth
	}
	height := req.Height
	if height == 0 {
		height = DefaultHeight
	}
	name := req.Name
	if name == "" {
		name = req.ID
	}

	disp := display.NewDisplay(req.ID, name, width, height, s.config, s.apps)
	s.displays[req.ID] = disp
	disp.Start()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "id": req.ID})
}

func (s *Server) getDisplay(displayID string) *display.Display {
	if disp, ok := s.displays[displayID]; ok {
		return disp
	}
	return nil
}

// getFirstDisplay returns the first available display (for legacy single-display API)
func (s *Server) getFirstDisplay() *display.Display {
	for _, d := range s.displays {
		return d
	}
	return nil
}

func (s *Server) handleGetDisplayByID(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	frame := disp.GetFrame()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":               disp.ID,
		"name":             disp.Name,
		"width":            disp.Width,
		"height":           disp.Height,
		"brightness":       frame.Brightness,
		"power":            disp.IsPowerOn(),
		"rotation_enabled": disp.IsRotationEnabled(),
		"current_app":      frame.AppName,
	})
}

func (s *Server) handleUpdateDisplay(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if brightness, ok := req["brightness"].(float64); ok {
		disp.SetBrightness(int(brightness))
	}
	if power, ok := req["power"].(bool); ok {
		disp.SetPower(power)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleSetDisplayBrightness(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	var req struct {
		Brightness int `json:"brightness"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	disp.SetBrightness(req.Brightness)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"brightness": req.Brightness})
}

func (s *Server) handleSetDisplayPower(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	var req struct {
		Power bool `json:"power"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	disp.SetPower(req.Power)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"power": req.Power})
}

func (s *Server) handleDisplaySkip(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	disp.Skip()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleGetDisplayRotation(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled": disp.IsRotationEnabled(),
		"apps":    disp.GetRotationApps(),
	})
}

func (s *Server) handleSetDisplayRotation(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	var req struct {
		Enabled *bool `json:"enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Enabled != nil {
		disp.SetRotationEnabled(*req.Enabled)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleAddToDisplayRotation(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	var req struct {
		AppID string `json:"app_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := disp.AddToRotation(req.AppID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleRemoveFromDisplayRotation(w http.ResponseWriter, r *http.Request) {
	displayID := chi.URLParam(r, "displayID")
	disp := s.getDisplay(displayID)
	if disp == nil {
		http.Error(w, "Display not found", http.StatusNotFound)
		return
	}

	appID := chi.URLParam(r, "appID")
	if err := disp.RemoveFromRotation(appID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

// Legacy single-display handlers (use default display)

func (s *Server) handleGetDisplay(w http.ResponseWriter, r *http.Request) {
	// Get first available display
	var disp *display.Display
	for _, d := range s.displays {
		disp = d
		break
	}
	if disp == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "", "name": "No displays", "width": 64, "height": 32,
			"brightness": 80, "power": false, "rotation_enabled": false, "current_app": "",
		})
		return
	}

	frame := s.getFirstDisplay().GetFrame()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":               s.getFirstDisplay().ID,
		"name":             s.getFirstDisplay().Name,
		"width":            s.getFirstDisplay().Width,
		"height":           s.getFirstDisplay().Height,
		"brightness":       frame.Brightness,
		"power":            s.getFirstDisplay().IsPowerOn(),
		"rotation_enabled": s.getFirstDisplay().IsRotationEnabled(),
		"current_app":      frame.AppName,
	})
}

func (s *Server) handleSetBrightness(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		Brightness int `json:"brightness"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.getFirstDisplay().SetBrightness(req.Brightness); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"brightness": req.Brightness})
}

func (s *Server) handleSetPower(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		Power bool `json:"power"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.getFirstDisplay().SetPower(req.Power); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"power": req.Power})
}

func (s *Server) handleSkip(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	s.getFirstDisplay().Skip()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleGetRotation(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled": s.getFirstDisplay().IsRotationEnabled(),
		"apps":    s.getFirstDisplay().GetRotationApps(),
	})
}

func (s *Server) handleSetRotation(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		Apps []rotation.AppEntry `json:"apps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.getFirstDisplay().SetRotationApps(req.Apps); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "apps": req.Apps})
}

func (s *Server) handleSetRotationEnabled(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.getFirstDisplay().SetRotationEnabled(req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"enabled": req.Enabled})
}

func (s *Server) handleAddToRotation(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		AppID string `json:"app_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.getFirstDisplay().AddToRotation(req.AppID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleRemoveFromRotation(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	appID := chi.URLParam(r, "appID")
	if err := s.getFirstDisplay().RemoveFromRotation(appID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	if s.apps == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.apps.List())
}

func (s *Server) handleListCommunity(w http.ResponseWriter, r *http.Request) {
	if s.apps == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.apps.ListCommunity())
}

func (s *Server) handleSearchCommunity(w http.ResponseWriter, r *http.Request) {
	if s.apps == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	query := r.URL.Query().Get("q")
	results := s.apps.SearchCommunity(query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	if s.apps == nil {
		http.Error(w, "App repository not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		AppID string `json:"app_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	app, err := s.apps.Install(req.AppID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (s *Server) handleUploadApp(w http.ResponseWriter, r *http.Request) {
	if s.apps == nil {
		http.Error(w, "App repository not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Source == "" {
		http.Error(w, "ID and source are required", http.StatusBadRequest)
		return
	}

	app, err := s.apps.InstallFromSource(req.ID, req.Name, []byte(req.Source))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (s *Server) handleUninstallApp(w http.ResponseWriter, r *http.Request) {
	if s.apps == nil {
		http.Error(w, "App repository not initialized", http.StatusInternalServerError)
		return
	}

	appID := chi.URLParam(r, "appID")
	if err := s.apps.Uninstall(appID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleGetApp(w http.ResponseWriter, r *http.Request) {
	if s.apps == nil {
		http.Error(w, "App repository not initialized", http.StatusInternalServerError)
		return
	}

	appID := chi.URLParam(r, "appID")
	app := s.apps.Get(appID)
	if app == nil {
		http.Error(w, "App not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

func (s *Server) handleSaveAppConfig(w http.ResponseWriter, r *http.Request) {
	if s.apps == nil {
		http.Error(w, "App repository not initialized", http.StatusInternalServerError)
		return
	}

	appID := chi.URLParam(r, "appID")
	
	var config map[string]string
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.apps.SaveConfig(appID, config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleRenderApp(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		AppPath string            `json:"app_path"`
		Source  string            `json:"source"`
		AppID   string            `json:"app_id"`
		Config  map[string]string `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	appID := req.AppID
	if appID == "" {
		appID = "inline"
	}

	if req.Source != "" {
		if err := s.getFirstDisplay().RenderSource(appID, []byte(req.Source), req.Config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if req.AppPath != "" {
		// Use ShowApp with duration 0 (permanent until next rotation)
		if err := s.getFirstDisplay().ShowApp(req.AppPath, 0); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Must provide app_path or source", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handlePushNotify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text      string `json:"text"`
		Color     string `json:"color"`
		Duration  int    `json:"duration"`
		Priority  string `json:"priority"`
		DisplayID string `json:"display_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get display ID from request or query parameter, default to first display
	displayID := req.DisplayID
	if displayID == "" {
		displayID = r.URL.Query().Get("display")
	}

	var disp *display.Display
	if displayID != "" {
		disp = s.getDisplay(displayID)
	} else {
		disp = s.getFirstDisplay()
	}

	if disp == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	priority := rotation.PriorityNormal
	switch req.Priority {
	case "low":
		priority = rotation.PriorityLow
	case "high":
		priority = rotation.PriorityHigh
	case "sticky":
		priority = rotation.PrioritySticky
	}

	disp.PushText(req.Text, req.Color, req.Duration, priority)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (s *Server) handleShowApp(w http.ResponseWriter, r *http.Request) {
	if s.getFirstDisplay() == nil {
		http.Error(w, "Display not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		AppID    string `json:"app_id"`
		Duration int    `json:"duration"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.getFirstDisplay().ShowApp(req.AppID, req.Duration); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

// handleFrame serves raw RGB pixels to LED matrix clients
func (s *Server) handleFrame(w http.ResponseWriter, r *http.Request) {
	var frame *display.FrameData

	// Get display ID from query param, default to "default"
	displayID := r.URL.Query().Get("display")
	if displayID == "" {
		displayID = "default"
	}

	if disp := s.displays[displayID]; disp != nil {
		frame = disp.GetFrame()
	} else {
		// No display registered yet, return fallback
		frame = s.generateFallbackFrame()
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Frame-Width", strconv.Itoa(frame.Width))
	w.Header().Set("X-Frame-Height", strconv.Itoa(frame.Height))
	w.Header().Set("X-Frame-Count", strconv.Itoa(frame.FrameCount))
	w.Header().Set("X-Frame-Delay-Ms", strconv.Itoa(frame.DelayMs))
	w.Header().Set("X-Dwell-Secs", strconv.Itoa(frame.DwellSecs))
	w.Header().Set("X-Brightness", strconv.Itoa(frame.Brightness))
	w.Header().Set("X-App-Name", frame.AppName)

	w.Write(frame.Pixels)
}

// handleFramePreview serves a PNG preview for browser debugging
func (s *Server) handleFramePreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("PNG preview not yet implemented - use /frame for raw pixels"))
}

func (s *Server) generateFallbackFrame() *display.FrameData {
	width := DefaultWidth
	height := DefaultHeight
	pixels := make([]byte, width*height*3)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			r := byte((x * 255) / width)
			g := byte((y * 255) / height)
			b := byte(128)

			pixels[idx] = r
			pixels[idx+1] = g
			pixels[idx+2] = b
		}
	}

	return &display.FrameData{
		Pixels:     pixels,
		Width:      width,
		Height:     height,
		FrameCount: 1,
		DelayMs:    50,
		DwellSecs:  10,
		Brightness: 80,
		AppName:    "test-pattern",
	}
}
