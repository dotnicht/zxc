package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"zxc/internal/config"
	"zxc/internal/infra"
	"zxc/internal/models"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	configPath := flag.String("config", "config.toml", "path to configuration file")
	flag.Parse()

	slog.Info("=== Migration Runner ===")
	slog.Info("Config", "path", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Ensuring root database exists...")
	dbCreated, err := ensureRoot(cfg.Database)
	if err != nil {
		slog.Error("Failed to ensure root database", "error", err)
		os.Exit(1)
	}

	slog.Info("Connecting to root database...")
	rootDB, err := infra.Connect(cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to root database", "error", err)
		os.Exit(1)
	}

	sqlDB, err := rootDB.DB()
	if err != nil {
		slog.Error("Failed to get database instance", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	slog.Info("Running root migrations...")
	if err := (infra.Migrator{DB: rootDB}).Root(); err != nil {
		slog.Error("Failed to run root migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("Root migrations completed")

	if dbCreated {
		user := &models.User{
			ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Name: "adolf",
		}
		if err := rootDB.Create(user).Error; err != nil {
			slog.Error("Failed to create default user", "error", err)
			os.Exit(1)
		}
		slog.Info("Created default user", "name", user.Name, "id", user.ID)
	}

	slog.Info("Fetching tenants...")
	var tenants []*models.Tenant
	if err := rootDB.Find(&tenants).Error; err != nil {
		slog.Error("Failed to fetch tenants", "error", err)
		os.Exit(1)
	}

	if len(tenants) == 0 {
		slog.Info("No tenants found. Nothing to migrate.")
		return
	}

	slog.Info("Running tenant migrations", "count", len(tenants))

	var successCount, failCount int
	for i, tenant := range tenants {
		slog.Info("Migrating tenant", "index", i+1, "total", len(tenants), "name", tenant.Name)
		start := time.Now()
		if err := migrateTenant(tenant); err != nil {
			slog.Error("Tenant migration failed", "name", tenant.Name, "error", err)
			failCount++
		} else {
			slog.Info("Tenant migration completed", "name", tenant.Name, "duration", time.Since(start))
			successCount++
		}
	}

	slog.Info("Summary", "succeeded", successCount, "failed", failCount)
	if failCount > 0 {
		slog.Error("Migration completed with errors", "count", failCount)
		os.Exit(1)
	}
	slog.Info("All migrations completed successfully")
}

func ensureRoot(conn string) (created bool, err error) {
	u, err := url.Parse(conn)
	if err != nil {
		return false, fmt.Errorf("invalid connection string: %w", err)
	}

	dbName := strings.TrimPrefix(u.Path, "/")

	adminURL := *u
	adminURL.Path = "/postgres"

	sqlDB, err := sql.Open("postgres", adminURL.String())
	if err != nil {
		return false, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		return false, fmt.Errorf("postgres is unreachable: %w", err)
	}

	var exists bool
	if err := sqlDB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		slog.Info("Root database not found, creating...", "name", dbName)
		if _, err := sqlDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
			return false, fmt.Errorf("failed to create database: %w", err)
		}
		slog.Info("Created root database", "name", dbName)
		return true, nil
	}

	slog.Info("Root database already exists", "name", dbName)
	return false, nil
}

func migrateTenant(tenant *models.Tenant) error {
	for _, m := range []struct {
		connStr string
		fn      func(infra.Migrator) error
		label   string
	}{
		{tenant.Main, func(mg infra.Migrator) error { return mg.Main() }, "main"},
		{tenant.Deploy, func(mg infra.Migrator) error { return mg.Deploy() }, "deploy"},
		{tenant.Account, func(mg infra.Migrator) error { return mg.Account() }, "account"},
	} {
		if m.connStr == "" {
			continue
		}
		db, err := gorm.Open(postgres.Open(m.connStr), &gorm.Config{Logger: nil})
		if err != nil {
			return fmt.Errorf("connect to %s db: %w", m.label, err)
		}
		if err := m.fn(infra.Migrator{DB: db}); err != nil {
			return fmt.Errorf("migrate %s db: %w", m.label, err)
		}
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
	return nil
}
