package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cschleiden/go-workflows/worker"
	"zxc/internal/config"
	"zxc/internal/infra"
	"zxc/internal/jobs"
	"zxc/internal/models"
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

	jobs.RegisterDeployDeps(root, infra.NewConnection, cfg)
	jobs.RegisterAccountDeps(root, infra.NewConnection, infra.NewConnection)
	jobs.RegisterProbeDeps(root, infra.NewConnection)
	jobs.RegisterGenerateDeps(root, infra.NewConnection)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	var (
		mu      sync.Mutex
		running = map[string]*worker.Worker{} // keyed by jobs DSN
	)

	startWorker := func(jobsDSN string) {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := running[jobsDSN]; ok {
			return
		}
		backend := infra.WorkflowBackend(jobsDSN)
		w := worker.New(backend, nil)
		if err := w.RegisterWorkflow(jobs.Deploy); err != nil {
			slog.Error("register deploy workflow", "error", err)
			return
		}
		if err := w.RegisterWorkflow(jobs.Account); err != nil {
			slog.Error("register account workflow", "error", err)
			return
		}
		if err := w.RegisterWorkflow(jobs.Probe); err != nil {
			slog.Error("register probe workflow", "error", err)
			return
		}
		if err := w.RegisterWorkflow(jobs.Generate); err != nil {
			slog.Error("register generate workflow", "error", err)
			return
		}
		if err := w.RegisterActivity(jobs.DeployActivity); err != nil {
			slog.Error("register deploy activity", "error", err)
			return
		}
		if err := w.RegisterActivity(jobs.AccountActivity); err != nil {
			slog.Error("register account activity", "error", err)
			return
		}
		if err := w.RegisterActivity(jobs.ProbeActivity); err != nil {
			slog.Error("register probe activity", "error", err)
			return
		}
		if err := w.RegisterActivity(jobs.GenerateActivity); err != nil {
			slog.Error("register generate activity", "error", err)
			return
		}
		if err := w.Start(ctx); err != nil {
			slog.Error("start worker", "dsn", jobsDSN, "error", err)
			return
		}
		running[jobsDSN] = w
		slog.Info("started worker for tenant jobs DB", "dsn", jobsDSN)
	}

	discover := func() {
		var tenants []models.Tenant
		if err := root.WithContext(ctx).Where("deleted_at IS NULL").Find(&tenants).Error; err != nil {
			slog.Error("discover tenants", "error", err)
			return
		}
		for _, t := range tenants {
			if t.Jobs != "" {
				startWorker(t.Jobs)
			}
		}
	}

	// Initial discovery
	discover()

	// Poll for new tenants
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for dsn, w := range running {
				w.WaitForCompletion()
				slog.Info("worker stopped", "dsn", dsn)
			}
			mu.Unlock()
			return
		case <-ticker.C:
			discover()
		}
	}
}
