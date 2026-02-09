package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		level:    LevelInfo,
		output:   &buf,
		ring:     make([]LogEntry, 100),
		ringSize: 100,
	}

	l.Debug("should not appear")
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("debug message should be filtered out at info level")
	}
	if !strings.Contains(output, "[INFO] info message") {
		t.Error("info message should appear")
	}
	if !strings.Contains(output, "[WARN] warn message") {
		t.Error("warn message should appear")
	}
	if !strings.Contains(output, "[ERROR] error message") {
		t.Error("error message should appear")
	}
}

func TestLogFormat(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		level:    LevelDebug,
		output:   &buf,
		ring:     make([]LogEntry, 100),
		ringSize: 100,
	}

	l.Info("hello %s %d", "world", 42)

	output := buf.String()
	if !strings.Contains(output, "hello world 42") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestLogTimestamps(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		level:     LevelDebug,
		output:    &buf,
		timestamp: true,
		ring:      make([]LogEntry, 100),
		ringSize:  100,
	}

	l.Info("test")
	output := buf.String()
	// RFC3339 timestamp has "T" between date and time
	if !strings.Contains(output, "T") {
		t.Errorf("expected timestamp in output: %s", output)
	}
}

func TestRecent(t *testing.T) {
	l := &Logger{
		level:    LevelDebug,
		output:   &bytes.Buffer{},
		ring:     make([]LogEntry, 100),
		ringSize: 100,
	}

	l.Info("first")
	l.Warn("second")
	l.Error("third")

	entries := l.Recent(10, "")
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Most recent first
	if entries[0].Message != "third" {
		t.Errorf("expected 'third', got %q", entries[0].Message)
	}
}

func TestRecentWithFilter(t *testing.T) {
	l := &Logger{
		level:    LevelDebug,
		output:   &bytes.Buffer{},
		ring:     make([]LogEntry, 100),
		ringSize: 100,
	}

	l.Info("info1")
	l.Warn("warn1")
	l.Info("info2")
	l.Error("error1")

	entries := l.Recent(10, "WARN")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Message != "warn1" {
		t.Errorf("expected 'warn1', got %q", entries[0].Message)
	}
}

func TestRecentLimit(t *testing.T) {
	l := &Logger{
		level:    LevelDebug,
		output:   &bytes.Buffer{},
		ring:     make([]LogEntry, 100),
		ringSize: 100,
	}

	for i := 0; i < 20; i++ {
		l.Info("msg %d", i)
	}

	entries := l.Recent(5, "")
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"error", LevelError},
		{"unknown", LevelInfo},
	}

	for _, tt := range tests {
		if got := ParseLevel(tt.input); got != tt.expected {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
