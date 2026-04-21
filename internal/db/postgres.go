package db

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"zxc/internal/models"
)

func NewConnection(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func RunRootMigrations(db *gorm.DB) error {
	slog.Info("Running root database migrations")
	if err := db.AutoMigrate(&models.Tenant{}, &models.User{}, &models.Route{}, &models.Event{}, &models.Command{}); err != nil {
		return fmt.Errorf("failed to run root migrations: %w", err)
	}
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS events_aggregate_idx ON events(aggregate_type, aggregate_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS commands_due_idx ON commands(run_at, id)
			WHERE status IN ('pending', 'running');
		CREATE UNIQUE INDEX IF NOT EXISTS commands_active_dedupe_idx ON commands(dedupe_key)
			WHERE is_active = TRUE;
	`).Error; err != nil {
		return fmt.Errorf("failed to create workflow tables: %w", err)
	}
	slog.Info("Root migrations completed")
	return nil
}

func RunTenantMigrations(db *gorm.DB) error {
	slog.Info("Running tenant database migrations")
	if err := db.AutoMigrate(&models.User{}, &models.Account{}, &models.Request{}, &models.Target{}, &models.Payload{}, &models.Release{}); err != nil {
		return fmt.Errorf("failed to run tenant migrations: %w", err)
	}
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS accounts_name_active_idx ON accounts(name)
			WHERE deleted_at IS NULL;
	`).Error; err != nil {
		return fmt.Errorf("failed to create account indexes: %w", err)
	}
	if err := db.Exec(`
		DROP TRIGGER IF EXISTS releases_audit ON releases;
		DROP TRIGGER IF EXISTS payloads_audit ON payloads;
		DROP TRIGGER IF EXISTS servers_audit ON targets;

		DROP FUNCTION IF EXISTS audit_releases();
		DROP FUNCTION IF EXISTS audit_payloads();
		DROP FUNCTION IF EXISTS audit_servers();

		DROP TABLE IF EXISTS releases_history;
		DROP TABLE IF EXISTS payloads_history;
		DROP TABLE IF EXISTS servers_history;
	`).Error; err != nil {
		return fmt.Errorf("failed to remove temporal audit tables: %w", err)
	}
	slog.Info("Tenant migrations completed")
	return nil
}
