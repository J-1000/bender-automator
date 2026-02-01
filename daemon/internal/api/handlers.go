package api

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"time"
)

type DaemonStatus struct {
	Running   bool      `json:"running"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
	StartedAt time.Time `json:"started_at"`
	PID       int       `json:"pid"`
	GoVersion string    `json:"go_version"`
}

type HealthCheck struct {
	Status    string            `json:"status"`
	Checks    map[string]string `json:"checks"`
	Timestamp time.Time         `json:"timestamp"`
}

type StatusHandler struct {
	version   string
	startedAt time.Time
	pid       int
}

func NewStatusHandler(version string) *StatusHandler {
	return &StatusHandler{
		version:   version,
		startedAt: time.Now(),
		pid:       getpid(),
	}
}

func getpid() int {
	return os.Getpid()
}

func (h *StatusHandler) HandleStatus(ctx context.Context, params json.RawMessage) (any, error) {
	return DaemonStatus{
		Running:   true,
		Version:   h.version,
		Uptime:    time.Since(h.startedAt).Round(time.Second).String(),
		StartedAt: h.startedAt,
		PID:       h.pid,
		GoVersion: runtime.Version(),
	}, nil
}

func (h *StatusHandler) HandleHealth(ctx context.Context, params json.RawMessage) (any, error) {
	checks := map[string]string{
		"daemon": "ok",
	}

	return HealthCheck{
		Status:    "healthy",
		Checks:    checks,
		Timestamp: time.Now(),
	}, nil
}

// RegisterStatusHandlers registers status-related handlers on the server
func RegisterStatusHandlers(s *Server, version string) {
	h := NewStatusHandler(version)
	s.Handle("status.get", h.HandleStatus)
	s.Handle("status.health", h.HandleHealth)
}
