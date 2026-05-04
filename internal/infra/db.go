package infra

import (
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"github.com/cschleiden/go-workflows/backend"
	wfpostgres "github.com/cschleiden/go-workflows/backend/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"zxc/internal/models"
)

func NewWorkflowBackend(dsn string, migrate bool) backend.Backend {
	u, err := url.Parse(dsn)
	if err != nil {
		panic(fmt.Sprintf("parse dsn: %v", err))
	}
	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "5432"
	}
	port, _ := strconv.Atoi(portStr)
	password, _ := u.User.Password()
	dbname := u.Path
	if len(dbname) > 0 && dbname[0] == '/' {
		dbname = dbname[1:]
	}
	return wfpostgres.NewPostgresBackend(host, port, u.User.Username(), password, dbname,
		wfpostgres.WithApplyMigrations(migrate),
	)
}

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
	if err := db.AutoMigrate(&models.Tenant{}, &models.User{}); err != nil {
		return fmt.Errorf("failed to run root migrations: %w", err)
	}
	slog.Info("Root migrations completed")
	return nil
}

func RunUsersMigrations(db *gorm.DB) error {
	slog.Info("Running users database migrations")
	if err := db.AutoMigrate(&models.User{}); err != nil {
		return fmt.Errorf("failed to run users migrations: %w", err)
	}
	slog.Info("Users migrations completed")
	return nil
}

func RunDeployMigrations(db *gorm.DB) error {
	slog.Info("Running deploy database migrations")
	if err := db.AutoMigrate(&models.Target{}, &models.Payload{}, &models.Release{}, &models.Request{}); err != nil {
		return fmt.Errorf("failed to run deploy migrations: %w", err)
	}
	slog.Info("Deploy migrations completed")
	return nil
}

func RunAccountMigrations(db *gorm.DB) error {
	slog.Info("Running account database migrations")
	if err := db.AutoMigrate(&models.Profile{}, &models.Session{}, &models.Talk{}, &models.File{}, &models.Message{}); err != nil {
		return fmt.Errorf("failed to run account migrations: %w", err)
	}
	slog.Info("Account migrations completed")
	return nil
}
