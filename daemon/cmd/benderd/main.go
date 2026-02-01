package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	version   = "dev"
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
				log.Fatalf("failed to get home directory: %v", err)
			}
			configPath = homeDir + "/.config/bender/config.yaml"
		}
	}

	fmt.Printf("benderd version %s starting...\n", version)
	fmt.Printf("config: %s\n", configPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("\nreceived signal %v, shutting down...\n", sig)
		cancel()
	}()

	if err := run(ctx); err != nil {
		log.Fatalf("daemon error: %v", err)
	}

	fmt.Println("daemon stopped")
}

func run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
