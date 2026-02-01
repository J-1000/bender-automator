package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/bender/internal/config"
	"github.com/user/bender/internal/logging"
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
	logging.Info("daemon running with provider: %s", cfg.LLM.DefaultProvider)

	<-ctx.Done()
	return nil
}
