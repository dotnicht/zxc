package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"zxc/internal/config"
	"zxc/internal/db"
	"zxc/internal/jobs"
	"zxc/internal/logger"
	"zxc/internal/queue"
)

func main() {
	logger.Init()

	configPath := flag.String("config", "config.toml", "path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	sqlDB, err := sql.Open("postgres", cfg.Database)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	rootDB, err := db.NewConnection(cfg.Database)
	if err != nil {
		slog.Error("failed to connect to root database", "error", err)
		os.Exit(1)
	}

	cache := db.NewCache()

	q := queue.New(sqlDB)

	deployW := jobs.NewDeployWorker(cache.Get, rootDB, cfg)
	checkW := jobs.NewCheckWorker(cache.Get, rootDB)
	scanW := jobs.NewScanWorker(rootDB, q)
	aliveScanW := jobs.NewAliveCheckScanWorker(rootDB, q)
	tenantDepW := jobs.NewTenantDeployWorker(rootDB, cache.Get, q)
	tenantChkW := jobs.NewTenantCheckWorker(rootDB, cache.Get, q)
	targetScanW := jobs.NewTargetScanWorker(rootDB, q)
	tenantTargetChkW := jobs.NewTenantTargetCheckWorker(rootDB, cache.Get, q)
	targetChkW := jobs.NewTargetCheckWorker(rootDB, cache.Get)

	queue.Register(q, "deploy", deployW.Work)
	queue.Register(q, "check", checkW.Work)
	queue.Register(q, "scan", scanW.Work)
	queue.Register(q, "alive_check_scan", aliveScanW.Work)
	queue.Register(q, "tenant_deploy", tenantDepW.Work)
	queue.Register(q, "tenant_check", tenantChkW.Work)
	queue.Register(q, "target_scan", targetScanW.Work)
	queue.Register(q, "tenant_target_check", tenantTargetChkW.Work)
	queue.Register(q, "target_check", targetChkW.Work)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	go runPeriodic(ctx, 30*time.Second, func() {
		if err := q.Insert(ctx, jobs.ScanArgs{}); err != nil {
			slog.Error("insert scan job", "error", err)
		}
	})
	go runPeriodic(ctx, 30*time.Second, func() {
		if err := q.Insert(ctx, jobs.AliveCheckScanArgs{}); err != nil {
			slog.Error("insert alive_check_scan job", "error", err)
		}
	})
	go runPeriodic(ctx, 30*time.Second, func() {
		if err := q.Insert(ctx, jobs.TargetScanArgs{}); err != nil {
			slog.Error("insert target_scan job", "error", err)
		}
	})

	q.Run(ctx)
}

func runPeriodic(ctx context.Context, interval time.Duration, fn func()) {
	fn()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}
