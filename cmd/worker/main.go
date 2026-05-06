package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
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

	root, err := infra.Connect(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to root database", "error", err)
		os.Exit(1)
	}

	pluginsDir := os.Getenv("PLUGINS_DIR")
	if pluginsDir == "" {
		pluginsDir = filepath.Join(filepath.Dir(os.Args[0]), "plugins")
	}

	jobs.RegisterDeploy(root, infra.Connect, cfg)
	jobs.RegisterAccount(root, infra.Connect, infra.Connect)
	jobs.RegisterProbe(root, infra.Connect)
	jobs.RegisterSync(root, infra.Connect, pluginsDir)

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
		running = map[string]*worker.Worker{}
	)

	startWorker := func(jobs_ string) {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := running[jobs_]; ok {
			return
		}
		backend, err := infra.Backend(jobs_)
		if err != nil {
			slog.Error("workflow backend", "jobs", jobs_, "error", err)
			return
		}
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
		if err := w.RegisterWorkflow(jobs.Sync); err != nil {
			slog.Error("register sync workflow", "error", err)
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
		if err := w.RegisterActivity(jobs.SyncActivity); err != nil {
			slog.Error("register sync activity", "error", err)
			return
		}
		if err := w.Start(ctx); err != nil {
			slog.Error("start worker", "jobs", jobs_, "error", err)
			return
		}
		running[jobs_] = w
		slog.Info("started worker for tenant jobs DB", "jobs", jobs_)
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

	discover()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			done := make(chan struct{})
			go func() {
				mu.Lock()
				defer mu.Unlock()
				for jobs_, w := range running {
					w.WaitForCompletion()
					slog.Info("worker stopped", "jobs", jobs_)
				}
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				slog.Info("shutdown timeout reached, exiting")
			}
			return
		case <-ticker.C:
			discover()
		}
	}
}
