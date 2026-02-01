package fswatch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/user/bender/internal/logging"
)

// EventType represents the type of file system event
type EventType int

const (
	EventCreate EventType = iota
	EventModify
	EventDelete
)

func (e EventType) String() string {
	switch e {
	case EventCreate:
		return "create"
	case EventModify:
		return "modify"
	case EventDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Event represents a file system event
type Event struct {
	Type EventType
	Path string
	Info os.FileInfo
}

// Handler processes file system events
type Handler func(event Event)

// Watcher monitors directories for file changes
type Watcher struct {
	dirs            []string
	excludePatterns []string
	ignoreHidden    bool
	pollInterval    time.Duration
	handler         Handler
	known           map[string]time.Time
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// Config for the file watcher
type Config struct {
	Dirs            []string
	ExcludePatterns []string
	IgnoreHidden    bool
	PollInterval    time.Duration
	Handler         Handler
}

// NewWatcher creates a new file system watcher
func NewWatcher(cfg Config) *Watcher {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 2 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Watcher{
		dirs:            cfg.Dirs,
		excludePatterns: cfg.ExcludePatterns,
		ignoreHidden:    cfg.IgnoreHidden,
		pollInterval:    cfg.PollInterval,
		handler:         cfg.Handler,
		known:           make(map[string]time.Time),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start begins watching directories
func (w *Watcher) Start() error {
	// Initial scan to populate known files
	for _, dir := range w.dirs {
		w.scanDir(dir, true)
	}

	go w.pollLoop()
	logging.Info("file watcher started for %d directories", len(w.dirs))
	return nil
}

// Stop halts file watching
func (w *Watcher) Stop() error {
	w.cancel()
	logging.Info("file watcher stopped")
	return nil
}

func (w *Watcher) pollLoop() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			for _, dir := range w.dirs {
				w.scanDir(dir, false)
			}
		}
	}
}

func (w *Watcher) scanDir(dir string, initial bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		logging.Debug("failed to read directory %s: %v", dir, err)
		return
	}

	current := make(map[string]bool)

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)

		// Skip hidden files if configured
		if w.ignoreHidden && strings.HasPrefix(name, ".") {
			continue
		}

		// Skip excluded patterns
		if w.matchesExclude(name) {
			continue
		}

		// Skip directories
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		current[path] = true
		modTime := info.ModTime()

		w.mu.Lock()
		knownTime, exists := w.known[path]
		if !exists {
			w.known[path] = modTime
			w.mu.Unlock()
			if !initial && w.handler != nil {
				w.handler(Event{
					Type: EventCreate,
					Path: path,
					Info: info,
				})
			}
		} else if modTime.After(knownTime) {
			w.known[path] = modTime
			w.mu.Unlock()
			if w.handler != nil {
				w.handler(Event{
					Type: EventModify,
					Path: path,
					Info: info,
				})
			}
		} else {
			w.mu.Unlock()
		}
	}

	// Check for deleted files
	w.mu.Lock()
	for path := range w.known {
		if filepath.Dir(path) != dir {
			continue
		}
		if !current[path] {
			delete(w.known, path)
			if w.handler != nil {
				go w.handler(Event{
					Type: EventDelete,
					Path: path,
				})
			}
		}
	}
	w.mu.Unlock()
}

func (w *Watcher) matchesExclude(name string) bool {
	for _, pattern := range w.excludePatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// AddDir adds a directory to watch
func (w *Watcher) AddDir(dir string) {
	w.mu.Lock()
	w.dirs = append(w.dirs, dir)
	w.mu.Unlock()
	w.scanDir(dir, true)
}

// RemoveDir removes a directory from watching
func (w *Watcher) RemoveDir(dir string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, d := range w.dirs {
		if d == dir {
			w.dirs = append(w.dirs[:i], w.dirs[i+1:]...)
			break
		}
	}

	// Clean up known files from this directory
	for path := range w.known {
		if filepath.Dir(path) == dir {
			delete(w.known, path)
		}
	}
}
