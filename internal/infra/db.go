package infra

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/cschleiden/go-workflows/backend"
	wfpostgres "github.com/cschleiden/go-workflows/backend/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"zxc/internal/models"
)

var (
	connCache   = map[string]*gorm.DB{}
	connCacheMu sync.Mutex

	wfBackendCache   = map[string]backend.Backend{}
	wfBackendCacheMu sync.Mutex
)

// WorkflowBackend returns a cached workflow backend for the given connection string.
// On first call it runs migrations; subsequent calls reuse the same instance.
// Returns an error if the connection string is invalid or the backend cannot be created.
func WorkflowBackend(conn string) (b backend.Backend, err error) {
	wfBackendCacheMu.Lock()
	defer wfBackendCacheMu.Unlock()
	if cached, ok := wfBackendCache[conn]; ok {
		return cached, nil
	}
	u, parseErr := url.Parse(conn)
	if parseErr != nil {
		return nil, fmt.Errorf("parse conn: %w", parseErr)
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
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("create backend: %v", r)
		}
	}()
	b = wfpostgres.NewPostgresBackend(host, port, u.User.Username(), password, dbname,
		wfpostgres.WithApplyMigrations(true),
		wfpostgres.WithPostgresOptions(func(db *sql.DB) {
			db.SetMaxIdleConns(1)
			db.SetMaxOpenConns(1)
			db.SetConnMaxLifetime(time.Hour)
		}),
	)
	wfBackendCache[conn] = b
	return b, nil
}

func NewConnection(conn string) (*gorm.DB, error) {
	connCacheMu.Lock()
	defer connCacheMu.Unlock()

	if db, ok := connCache[conn]; ok {
		return db, nil
	}

	db, err := gorm.Open(postgres.Open(conn), &gorm.Config{
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

	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	connCache[conn] = db
	return db, nil
}

func EnsureSchema(db *gorm.DB, schema string) error {
	return db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %q", schema)).Error
}

func RunRootMigrations(db *gorm.DB) error {
	slog.Info("Running root database migrations")
	if err := db.AutoMigrate(&models.Tenant{}, &models.User{}); err != nil {
		return fmt.Errorf("failed to run root migrations: %w", err)
	}
	slog.Info("Root migrations completed")
	return nil
}

func RunMainMigrations(db *gorm.DB) error {
	slog.Info("Running main database migrations")
	if err := db.AutoMigrate(&models.User{}, &models.System{}); err != nil {
		return fmt.Errorf("failed to run main migrations: %w", err)
	}
	slog.Info("Main migrations completed")
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
	if err := db.AutoMigrate(&models.Profile{}, &models.Session{}, &models.Talk{}, &models.File{}, &models.Contact{}, &models.Post{}); err != nil {
		return fmt.Errorf("failed to run account migrations: %w", err)
	}
	slog.Info("Account migrations completed")
	return nil
}
