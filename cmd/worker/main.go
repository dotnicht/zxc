package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zxc/internal/config"
	"zxc/internal/db"
	"zxc/internal/jobs"
	"zxc/internal/workflow"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	configPath := flag.String("config", "config.toml", "path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	rootDB, err := db.NewConnection(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to root database", "error", err)
		os.Exit(1)
	}

	cache := db.NewCache()
	store := workflow.NewStore(rootDB)
	runner, err := workflow.NewRunner(rootDB, 10*time.Minute, 8)
	if err != nil {
		slog.Error("failed to initialize workflow runner", "error", err)
		os.Exit(1)
	}

	deploy := jobs.NewDeployWorker(store, cache.Get, rootDB, cfg)
	health := jobs.NewReleaseHealthWorker(store, rootDB, cache.Get)
	alive := jobs.NewReleaseMarkAliveWorker(store, rootDB, cache.Get)
	probe := jobs.NewTargetProbeWorker(store, rootDB, cache.Get)

	workflow.Register(runner, "deploy_release", deploy.Work)
	workflow.Register(runner, "release_health_timeout", health.Work)
	workflow.Register(runner, "release_mark_alive", alive.Work)
	workflow.Register(runner, "probe_target", probe.Work)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	runner.Run(ctx)
}
