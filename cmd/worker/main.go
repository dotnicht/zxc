package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cschleiden/go-workflows/worker"
	"zxc/internal/config"
	"zxc/internal/infra"
	"zxc/internal/jobs"
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

	root, err := infra.NewConnection(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to root database", "error", err)
		os.Exit(1)
	}

	backend := infra.NewWorkflowBackend(cfg.Database, true)

	jobs.RegisterDeployDeps(root, infra.NewConnection, cfg)
	jobs.RegisterAccountDeps(root, infra.NewConnection)
	jobs.RegisterProbeDeps(root, infra.NewConnection)

	w := worker.New(backend, nil)
	if err := w.RegisterWorkflow(jobs.Deploy); err != nil {
		slog.Error("failed to register deploy workflow", "error", err)
		os.Exit(1)
	}
	if err := w.RegisterWorkflow(jobs.Account); err != nil {
		slog.Error("failed to register account workflow", "error", err)
		os.Exit(1)
	}
	if err := w.RegisterWorkflow(jobs.Probe); err != nil {
		slog.Error("failed to register probe workflow", "error", err)
		os.Exit(1)
	}
	if err := w.RegisterActivity(jobs.DeployActivity); err != nil {
		slog.Error("failed to register deploy activity", "error", err)
		os.Exit(1)
	}
	if err := w.RegisterActivity(jobs.AccountActivity); err != nil {
		slog.Error("failed to register account activity", "error", err)
		os.Exit(1)
	}
	if err := w.RegisterActivity(jobs.ProbeActivity); err != nil {
		slog.Error("failed to register probe activity", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if err := w.Start(ctx); err != nil {
		slog.Error("failed to start worker", "error", err)
		os.Exit(1)
	}

	w.WaitForCompletion()
}
