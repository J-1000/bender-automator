package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/user/bender/internal/api"
	"github.com/user/bender/internal/clipboard"
	"github.com/user/bender/internal/config"
	"github.com/user/bender/internal/fileops"
	"github.com/user/bender/internal/fswatch"
	"github.com/user/bender/internal/llm"
	"github.com/user/bender/internal/logging"
	"github.com/user/bender/internal/notify"
	"github.com/user/bender/internal/task"
)

var (
	version    = "dev"
	configPath string
)

func main() {
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	if configPath == "" {
		configPath = os.Getenv("BENDER_CONFIG")
		if configPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get home directory: %v\n", err)
				os.Exit(1)
			}
			configPath = homeDir + "/.config/bender/config.yaml"
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := logging.New(logging.Config{
		Level:     cfg.Logging.Level,
		Timestamp: cfg.Logging.IncludeTimestamps,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()
	logging.SetDefault(logger)

	logging.Info("benderd version %s starting", version)
	logging.Info("config loaded from %s", configPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logging.Info("received signal %v, shutting down...", sig)
		cancel()
	}()

	if err := run(ctx, cfg); err != nil {
		logging.Fatal("daemon error: %v", err)
	}

	logging.Info("daemon stopped")
}

func run(ctx context.Context, cfg *config.Config) error {
	// Initialize LLM router
	router, err := llm.NewRouter(&cfg.LLM)
	if err != nil {
		return fmt.Errorf("init llm router: %w", err)
	}
	logging.Info("LLM router initialized with provider: %s", router.DefaultProviderName())

	// Initialize task queue
	homeDir, _ := os.UserHomeDir()
	dbPath := filepath.Join(homeDir, ".local", "share", "bender", "bender.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	queue, err := task.NewQueue(task.Config{
		DBPath:       dbPath,
		MaxWorkers:   cfg.Queue.MaxConcurrent,
		MaxRetries:   cfg.Queue.MaxRetries,
		RetryDelay:   time.Duration(cfg.Queue.RetryDelaySeconds) * time.Second,
		TaskTimeout:  time.Duration(cfg.Queue.DefaultTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("init task queue: %w", err)
	}

	// Initialize undo manager
	undoMgr, err := fileops.NewUndoManager(filepath.Join(homeDir, ".local", "share", "bender", "bender.db"))
	if err != nil {
		return fmt.Errorf("init undo manager: %w", err)
	}
	defer undoMgr.Close()

	// Initialize notifier
	notifier := notify.New(notify.Config{
		Enabled:      cfg.Notifications.Enabled,
		Sound:        cfg.Notifications.Sound,
		ShowPreviews: cfg.Notifications.ShowPreviews,
	})

	// Register task handlers
	registerTaskHandlers(queue, router, cfg)

	if err := queue.Start(); err != nil {
		return fmt.Errorf("start task queue: %w", err)
	}
	defer queue.Stop()

	// Initialize API server
	server := api.NewServer("")
	api.RegisterStatusHandlers(server, version)
	registerAPIHandlers(server, queue, router, cfg, undoMgr)

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("start api server: %w", err)
	}
	defer server.Stop()

	// Initialize clipboard monitor
	var clipMonitor *clipboard.Monitor
	if cfg.Clipboard.Enabled {
		clipMonitor = clipboard.NewMonitor(clipboard.Config{
			MinLength:  cfg.Clipboard.MinLength,
			DebounceMs: cfg.Clipboard.DebounceMs,
			OnChange: func(content string) {
				if cfg.Clipboard.AutoSummarize {
					queue.Enqueue(task.TaskClipboardSummarize, []byte(`{"content":"`+escapeJSON(content)+`"}`), 0)
					if cfg.Clipboard.Notification {
						notifier.Send("Bender", "Summarizing clipboard content...")
					}
				}
			},
		})
		if err := clipMonitor.Start(); err != nil {
			logging.Warn("failed to start clipboard monitor: %v", err)
		} else {
			defer clipMonitor.Stop()
		}
	}

	// Initialize file watcher
	var fileWatcher *fswatch.Watcher
	if cfg.AutoFile.Enabled && len(cfg.AutoFile.WatchDirs) > 0 {
		fileWatcher = fswatch.NewWatcher(fswatch.Config{
			Dirs:            cfg.AutoFile.WatchDirs,
			ExcludePatterns: cfg.AutoFile.ExcludePatterns,
			IgnoreHidden:    cfg.AutoFile.IgnoreHidden,
			Handler: func(event fswatch.Event) {
				if event.Type == fswatch.EventCreate {
					queue.Enqueue(task.TaskFileClassify, []byte(`{"path":"`+escapeJSON(event.Path)+`"}`), 0)
				}
			},
		})
		if err := fileWatcher.Start(); err != nil {
			logging.Warn("failed to start file watcher: %v", err)
		} else {
			defer fileWatcher.Stop()
		}
	}

	logging.Info("daemon ready")
	<-ctx.Done()
	return nil
}

func registerTaskHandlers(queue *task.Queue, router *llm.Router, cfg *config.Config) {
	queue.RegisterHandler(task.TaskClipboardSummarize, func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
		return handleClipboardSummarize(ctx, payload, router)
	})

	queue.RegisterHandler(task.TaskFileClassify, func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
		return handleFileClassify(ctx, payload, router, cfg)
	})

	queue.RegisterHandler(task.TaskFileRename, func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
		return handleFileRename(ctx, payload, router, cfg)
	})

	queue.RegisterHandler(task.TaskGitCommit, func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
		return handleGitCommit(ctx, payload, router, cfg)
	})

	queue.RegisterHandler(task.TaskScreenshotTag, func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
		return handleScreenshotTag(ctx, payload, router, cfg)
	})
}

func registerAPIHandlers(server *api.Server, queue *task.Queue, router *llm.Router, cfg *config.Config, undoMgr *fileops.UndoManager) {
	// Config handlers
	server.Handle("config.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		return cfg, nil
	})

	server.Handle("config.set", func(ctx context.Context, params json.RawMessage) (any, error) {
		// TODO: merge partial config and persist
		return map[string]string{"status": "ok"}, nil
	})

	server.Handle("config.reload", func(ctx context.Context, params json.RawMessage) (any, error) {
		newCfg, err := config.Load(configPath)
		if err != nil {
			return nil, fmt.Errorf("reload config: %w", err)
		}
		*cfg = *newCfg
		logging.Info("configuration reloaded")
		return map[string]string{"status": "reloaded"}, nil
	})

	// Task handlers
	server.Handle("task.queue", func(ctx context.Context, params json.RawMessage) (any, error) {
		return queue.ListTasks(100)
	})

	server.Handle("task.cancel", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("parse params: %w", err)
		}
		if err := queue.CancelTask(p.ID); err != nil {
			return nil, err
		}
		return map[string]string{"status": "cancelled"}, nil
	})

	server.Handle("task.history", func(ctx context.Context, params json.RawMessage) (any, error) {
		limit := 50
		var p struct {
			Limit int `json:"limit"`
		}
		if err := json.Unmarshal(params, &p); err == nil && p.Limit > 0 {
			limit = p.Limit
		}
		return queue.ListTasks(limit)
	})

	// Ad-hoc feature handlers (synchronous - enqueue and wait)
	server.Handle("clipboard.summarize", func(ctx context.Context, params json.RawMessage) (any, error) {
		t, err := queue.EnqueueAndWait(ctx, task.TaskClipboardSummarize, params, 1)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(t.Result), nil
	})

	server.Handle("clipboard.get_summary", func(ctx context.Context, params json.RawMessage) (any, error) {
		// Return the most recent completed clipboard summarization
		tasks, err := queue.ListTasks(100)
		if err != nil {
			return nil, err
		}
		for _, t := range tasks {
			if t.Type == task.TaskClipboardSummarize && t.Status == task.StatusCompleted && t.Result != nil {
				return json.RawMessage(t.Result), nil
			}
		}
		return nil, nil
	})

	server.Handle("file.classify", func(ctx context.Context, params json.RawMessage) (any, error) {
		t, err := queue.EnqueueAndWait(ctx, task.TaskFileClassify, params, 1)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(t.Result), nil
	})

	server.Handle("file.rename", func(ctx context.Context, params json.RawMessage) (any, error) {
		t, err := queue.EnqueueAndWait(ctx, task.TaskFileRename, params, 1)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(t.Result), nil
	})

	server.Handle("git.generate_commit", func(ctx context.Context, params json.RawMessage) (any, error) {
		t, err := queue.EnqueueAndWait(ctx, task.TaskGitCommit, params, 1)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(t.Result), nil
	})

	// Logs handler
	server.Handle("logs.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		limit := 100
		levelFilter := ""
		var p struct {
			Limit int    `json:"limit"`
			Level string `json:"level"`
		}
		if err := json.Unmarshal(params, &p); err == nil {
			if p.Limit > 0 {
				limit = p.Limit
			}
			levelFilter = p.Level
		}
		return logging.Recent(limit, levelFilter), nil
	})

	// File operations
	server.Handle("file.move", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
			TaskID      string `json:"task_id"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("parse params: %w", err)
		}

		actualDst, err := fileops.MoveFile(p.Source, p.Destination)
		if err != nil {
			return nil, err
		}

		// Record for undo
		if p.TaskID != "" {
			undoMgr.Record(fileops.Operation{
				ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
				TaskID:       p.TaskID,
				Type:         fileops.OpMove,
				OriginalPath: p.Source,
				NewPath:      actualDst,
				CreatedAt:    time.Now(),
			})
		}

		logging.Info("moved %s -> %s", p.Source, actualDst)
		return map[string]string{"destination": actualDst}, nil
	})

	server.Handle("undo", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p struct {
			TaskID string `json:"task_id"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("parse params: %w", err)
		}

		count, err := undoMgr.Undo(p.TaskID)
		if err != nil {
			return nil, err
		}

		logging.Info("undid %d operations for task %s", count, p.TaskID)
		return map[string]any{"undone": count, "task_id": p.TaskID}, nil
	})
}

func escapeJSON(s string) string {
	// Basic JSON string escaping
	result := ""
	for _, c := range s {
		switch c {
		case '"':
			result += `\"`
		case '\\':
			result += `\\`
		case '\n':
			result += `\n`
		case '\r':
			result += `\r`
		case '\t':
			result += `\t`
		default:
			result += string(c)
		}
	}
	return result
}
