package clipboard

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/user/bender/internal/logging"
)

// Monitor watches clipboard for changes
type Monitor struct {
	minLength    int
	debounce     time.Duration
	onChange     func(content string)
	lastContent  string
	lastChange   int64
	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// Config for clipboard monitor
type Config struct {
	MinLength   int
	DebounceMs  int
	OnChange    func(content string)
}

// NewMonitor creates a new clipboard monitor
func NewMonitor(cfg Config) *Monitor {
	if cfg.MinLength == 0 {
		cfg.MinLength = 500
	}
	debounce := time.Duration(cfg.DebounceMs) * time.Millisecond
	if debounce == 0 {
		debounce = time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		minLength: cfg.MinLength,
		debounce:  debounce,
		onChange:  cfg.OnChange,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins monitoring the clipboard
func (m *Monitor) Start() error {
	go m.pollLoop()
	logging.Info("clipboard monitor started (min_length=%d)", m.minLength)
	return nil
}

// Stop halts clipboard monitoring
func (m *Monitor) Stop() error {
	m.cancel()
	logging.Info("clipboard monitor stopped")
	return nil
}

func (m *Monitor) pollLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkClipboard()
		}
	}
}

func (m *Monitor) checkClipboard() {
	content, err := m.read()
	if err != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Skip if content hasn't changed
	if content == m.lastContent {
		return
	}

	// Skip if content is too short
	if len(content) < m.minLength {
		m.lastContent = content
		return
	}

	// Debounce rapid changes
	now := time.Now().UnixMilli()
	if now-m.lastChange < m.debounce.Milliseconds() {
		return
	}

	m.lastContent = content
	m.lastChange = now

	if m.onChange != nil {
		go m.onChange(content)
	}
}

// read gets the current clipboard content using pbpaste
func (m *Monitor) read() (string, error) {
	cmd := exec.CommandContext(m.ctx, "pbpaste")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Read returns the current clipboard content
func Read() (string, error) {
	cmd := exec.Command("pbpaste")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Write sets the clipboard content using pbcopy
func Write(content string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}
