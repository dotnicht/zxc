package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
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
	if cfg.Worker.ID == uuid.Nil {
		slog.Error("worker.id must be set in config")
		os.Exit(1)
	}

	rootDB, err := db.NewConnection(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to root database", "error", err)
		os.Exit(1)
	}

	cache := db.NewCache()
	store := workflow.NewStore()

	deploy := jobs.NewDeployWorker(store, cache.Get, rootDB, cfg)
	health := jobs.NewReleaseHealthWorker(store, rootDB, cache.Get)
	alive := jobs.NewReleaseMarkAliveWorker(store, rootDB, cache.Get)
	account := jobs.NewAccountFromRequestWorker(store, rootDB, cache.Get)
	probe := jobs.NewTargetProbeWorker(store, rootDB, cache.Get)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	multiRunner := workflow.NewMultiRunner(rootDB, cache.Get, workflow.MultiRunnerOptions{
		WorkerID:      cfg.Worker.ID,
		Lease:         10 * time.Minute,
		MaxConcurrent: 8,
		SyncInterval:  5 * time.Second,
	}, func(runner *workflow.Runner) {
		workflow.Register(runner, "deploy_release", deploy.Work)
		workflow.Register(runner, "release_health_timeout", health.Work)
		workflow.Register(runner, "release_mark_alive", alive.Work)
		workflow.Register(runner, "account_from_request", account.Work)
		workflow.Register(runner, "probe_target", probe.Work)
	})
	multiRunner.Run(ctx)
}
