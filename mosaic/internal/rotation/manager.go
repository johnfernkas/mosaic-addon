package rotation

import (
	"log"
	"sync"
	"time"
)

// AppEntry represents an app in the rotation
type AppEntry struct {
	ID       string            `json:"id" yaml:"id"`
	Name     string            `json:"name" yaml:"name"`
	Path     string            `json:"path" yaml:"path"`
	Config   map[string]string `json:"config" yaml:"config"`
	DwellMs  int               `json:"dwell_ms" yaml:"dwell_ms"`   // 0 = use default
	Enabled  bool              `json:"enabled" yaml:"enabled"`
}

// Manager handles app rotation for a display
type Manager struct {
	mu sync.RWMutex

	apps         []AppEntry
	currentIndex int
	enabled      bool
	defaultDwell time.Duration
	brightness   int

	// Notification queue (higher priority)
	notifyQueue []Notification

	// Channels for control
	skipCh   chan struct{}
	updateCh chan struct{}
	stopCh   chan struct{}

	// Callback when rotation advances
	onAdvance func(app AppEntry)
}

// Notification represents a temporary display override
type Notification struct {
	ID       string            `json:"id"`
	Text     string            `json:"text,omitempty"`
	AppID    string            `json:"app_id,omitempty"`
	Source   string            `json:"source,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
	Duration time.Duration     `json:"duration"`
	Priority Priority          `json:"priority"`
	Created  time.Time         `json:"created"`
}

// Priority levels for notifications
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PrioritySticky
)

// NewManager creates a new rotation manager
func NewManager(defaultDwell time.Duration) *Manager {
	if defaultDwell == 0 {
		defaultDwell = 10 * time.Second
	}

	return &Manager{
		apps:         make([]AppEntry, 0),
		enabled:      true,
		defaultDwell: defaultDwell,
		brightness:   80,
		skipCh:       make(chan struct{}, 1),
		updateCh:     make(chan struct{}, 1),
		stopCh:       make(chan struct{}),
	}
}

// SetApps sets the app rotation list
func (m *Manager) SetApps(apps []AppEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.apps = apps
	if m.currentIndex >= len(apps) {
		m.currentIndex = 0
	}

	m.notifyUpdate()
}

// GetApps returns the current app list
func (m *Manager) GetApps() []AppEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]AppEntry, len(m.apps))
	copy(result, m.apps)
	return result
}

// AddApp adds an app to the rotation
func (m *Manager) AddApp(app AppEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.apps = append(m.apps, app)
	m.notifyUpdate()
}

// RemoveApp removes an app from the rotation
func (m *Manager) RemoveApp(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, app := range m.apps {
		if app.ID == id {
			m.apps = append(m.apps[:i], m.apps[i+1:]...)
			if m.currentIndex >= len(m.apps) && len(m.apps) > 0 {
				m.currentIndex = 0
			}
			m.notifyUpdate()
			return true
		}
	}
	return false
}

// SetEnabled enables or disables rotation
func (m *Manager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// IsEnabled returns whether rotation is enabled
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// SetBrightness sets display brightness (0-100)
func (m *Manager) SetBrightness(brightness int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if brightness < 0 {
		brightness = 0
	} else if brightness > 100 {
		brightness = 100
	}
	m.brightness = brightness
}

// GetBrightness returns current brightness
func (m *Manager) GetBrightness() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.brightness
}

// Skip advances to the next app immediately
func (m *Manager) Skip() {
	select {
	case m.skipCh <- struct{}{}:
	default:
	}
}

// CurrentApp returns the currently displayed app
func (m *Manager) CurrentApp() *AppEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.apps) == 0 {
		return nil
	}

	// Find next enabled app
	for i := 0; i < len(m.apps); i++ {
		idx := (m.currentIndex + i) % len(m.apps)
		if m.apps[idx].Enabled {
			app := m.apps[idx]
			return &app
		}
	}

	return nil
}

// PushNotification adds a notification to the queue
func (m *Manager) PushNotification(n Notification) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n.Created = time.Now()
	
	// Insert by priority (higher priority first)
	inserted := false
	for i, existing := range m.notifyQueue {
		if n.Priority > existing.Priority {
			m.notifyQueue = append(m.notifyQueue[:i], append([]Notification{n}, m.notifyQueue[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		m.notifyQueue = append(m.notifyQueue, n)
	}

	m.notifyUpdate()
}

// ClearNotifications clears all non-sticky notifications
func (m *Manager) ClearNotifications() {
	m.mu.Lock()
	defer m.mu.Unlock()

	sticky := make([]Notification, 0)
	for _, n := range m.notifyQueue {
		if n.Priority == PrioritySticky {
			sticky = append(sticky, n)
		}
	}
	m.notifyQueue = sticky
}

// OnAdvance sets the callback for when rotation advances
func (m *Manager) OnAdvance(fn func(app AppEntry)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onAdvance = fn
}

// Run starts the rotation loop (blocking)
func (m *Manager) Run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastAdvance time.Time
	var currentDwell time.Duration

	for {
		select {
		case <-m.stopCh:
			return

		case <-m.skipCh:
			m.advance()
			lastAdvance = time.Now()
			currentDwell = m.getCurrentDwell()

		case <-m.updateCh:
			// Config changed, recalculate
			currentDwell = m.getCurrentDwell()

		case <-ticker.C:
			if !m.IsEnabled() {
				continue
			}

			// Check if we should advance
			if currentDwell == 0 {
				currentDwell = m.getCurrentDwell()
				lastAdvance = time.Now()
			}

			if time.Since(lastAdvance) >= currentDwell {
				m.advance()
				lastAdvance = time.Now()
				currentDwell = m.getCurrentDwell()
			}
		}
	}
}

// Stop stops the rotation loop
func (m *Manager) Stop() {
	close(m.stopCh)
}

func (m *Manager) advance() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.apps) == 0 {
		return
	}

	// Find next enabled app
	startIdx := m.currentIndex
	for i := 0; i < len(m.apps); i++ {
		m.currentIndex = (startIdx + i + 1) % len(m.apps)
		if m.apps[m.currentIndex].Enabled {
			break
		}
	}

	if m.onAdvance != nil {
		app := m.apps[m.currentIndex]
		go m.onAdvance(app)
	}

	log.Printf("Rotation advanced to: %s", m.apps[m.currentIndex].Name)
}

func (m *Manager) getCurrentDwell() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.apps) == 0 {
		return m.defaultDwell
	}

	app := m.apps[m.currentIndex]
	if app.DwellMs > 0 {
		return time.Duration(app.DwellMs) * time.Millisecond
	}
	return m.defaultDwell
}

func (m *Manager) notifyUpdate() {
	select {
	case m.updateCh <- struct{}{}:
	default:
	}
}
