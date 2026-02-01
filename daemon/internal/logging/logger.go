package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	mu        sync.Mutex
	level     Level
	output    io.Writer
	file      *os.File
	timestamp bool
}

type Config struct {
	Level     string
	Output    string
	Timestamp bool
}

func New(cfg Config) (*Logger, error) {
	l := &Logger{
		level:     ParseLevel(cfg.Level),
		timestamp: cfg.Timestamp,
	}

	if cfg.Output == "" || cfg.Output == "stdout" {
		l.output = os.Stdout
	} else {
		dir := filepath.Dir(cfg.Output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}

		f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		l.file = f
		l.output = f
	}

	return l, nil
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) log(level Level, format string, args ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)

	var line string
	if l.timestamp {
		line = fmt.Sprintf("%s [%s] %s\n", time.Now().Format(time.RFC3339), level, msg)
	} else {
		line = fmt.Sprintf("[%s] %s\n", level, msg)
	}

	l.output.Write([]byte(line))
}

func (l *Logger) Debug(format string, args ...any) {
	l.log(LevelDebug, format, args...)
}

func (l *Logger) Info(format string, args ...any) {
	l.log(LevelInfo, format, args...)
}

func (l *Logger) Warn(format string, args ...any) {
	l.log(LevelWarn, format, args...)
}

func (l *Logger) Error(format string, args ...any) {
	l.log(LevelError, format, args...)
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
}

// Default logger for package-level functions
var defaultLogger = &Logger{
	level:     LevelInfo,
	output:    os.Stdout,
	timestamp: true,
}

func SetDefault(l *Logger) {
	defaultLogger = l
}

func Debug(format string, args ...any) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...any) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...any) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...any) {
	defaultLogger.Error(format, args...)
}

// Fatal logs and exits
func Fatal(format string, args ...any) {
	defaultLogger.Error(format, args...)
	log.Fatalf(format, args...)
}
