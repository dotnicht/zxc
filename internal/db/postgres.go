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
		Logger: logger.Default.LogMode(logger.Info),
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
	if err := db.AutoMigrate(&models.Tenant{}, &models.User{}, &models.Route{}); err != nil {
		return fmt.Errorf("failed to run root migrations: %w", err)
	}
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id           BIGSERIAL PRIMARY KEY,
			kind         TEXT NOT NULL,
			args         JSONB NOT NULL,
			status       TEXT NOT NULL DEFAULT 'pending',
			attempt      INT NOT NULL DEFAULT 0,
			max_attempts INT NOT NULL DEFAULT 5,
			scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			locked_until TIMESTAMPTZ,
			finished_at  TIMESTAMPTZ,
			error        TEXT,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS jobs_pending_idx ON jobs(scheduled_at) WHERE status = 'pending';
	`).Error; err != nil {
		return fmt.Errorf("failed to create jobs table: %w", err)
	}
	slog.Info("Root migrations completed")
	return nil
}

func RunTenantMigrations(db *gorm.DB) error {
	slog.Info("Running tenant database migrations")
	if err := db.AutoMigrate(&models.User{}, &models.Request{}, &models.Target{}, &models.Payload{}, &models.Release{}); err != nil {
		return fmt.Errorf("failed to run tenant migrations: %w", err)
	}
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS releases_history (
			history_id    BIGSERIAL PRIMARY KEY,
			operation     CHAR(1) NOT NULL,
			changed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			id            UUID,
			status        VARCHAR(20),
			owner_id      UUID,
			target_id     UUID,
			payload_id    UUID,
			changed_by_id UUID,
			created_at    TIMESTAMPTZ,
			updated_at    TIMESTAMPTZ,
			deleted_at    TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS releases_history_id_idx ON releases_history(id);

		CREATE TABLE IF NOT EXISTS payloads_history (
			history_id BIGSERIAL PRIMARY KEY,
			operation  CHAR(1) NOT NULL,
			changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			id         UUID,
			path       TEXT,
			owner_id   UUID,
			config     TEXT,
			start      TEXT,
			stop       TEXT,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ,
			deleted_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS payloads_history_id_idx ON payloads_history(id);

		CREATE TABLE IF NOT EXISTS servers_history (
			history_id   BIGSERIAL PRIMARY KEY,
			operation    CHAR(1) NOT NULL,
			changed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			id           UUID,
			address      TEXT,
			"user"       TEXT,
			key          TEXT,
			status       VARCHAR(20),
			deploying    BOOLEAN,
			deploying_at TIMESTAMPTZ,
			owner_id     UUID,
			created_at   TIMESTAMPTZ,
			updated_at   TIMESTAMPTZ,
			deleted_at   TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS servers_history_id_idx ON servers_history(id);

		CREATE OR REPLACE FUNCTION audit_releases() RETURNS TRIGGER AS $$
		BEGIN
			IF TG_OP = 'DELETE' THEN
				INSERT INTO releases_history(operation,id,status,owner_id,target_id,payload_id,changed_by_id,created_at,updated_at,deleted_at)
				VALUES('D',OLD.id,OLD.status,OLD.owner_id,OLD.target_id,OLD.payload_id,OLD.changed_by_id,OLD.created_at,OLD.updated_at,OLD.deleted_at);
			ELSE
				INSERT INTO releases_history(operation,id,status,owner_id,target_id,payload_id,changed_by_id,created_at,updated_at,deleted_at)
				VALUES(CASE WHEN TG_OP='INSERT' THEN 'I' ELSE 'U' END,NEW.id,NEW.status,NEW.owner_id,NEW.target_id,NEW.payload_id,NEW.changed_by_id,NEW.created_at,NEW.updated_at,NEW.deleted_at);
			END IF;
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS releases_audit ON releases;
		CREATE TRIGGER releases_audit
		AFTER INSERT OR UPDATE OR DELETE ON releases
		FOR EACH ROW EXECUTE FUNCTION audit_releases();

		CREATE OR REPLACE FUNCTION audit_payloads() RETURNS TRIGGER AS $$
		BEGIN
			IF TG_OP = 'DELETE' THEN
				INSERT INTO payloads_history(operation,id,path,owner_id,config,start,stop,created_at,updated_at,deleted_at)
				VALUES('D',OLD.id,OLD.path,OLD.owner_id,OLD.config,OLD.start,OLD.stop,OLD.created_at,OLD.updated_at,OLD.deleted_at);
			ELSE
				INSERT INTO payloads_history(operation,id,path,owner_id,config,start,stop,created_at,updated_at,deleted_at)
				VALUES(CASE WHEN TG_OP='INSERT' THEN 'I' ELSE 'U' END,NEW.id,NEW.path,NEW.owner_id,NEW.config,NEW.start,NEW.stop,NEW.created_at,NEW.updated_at,NEW.deleted_at);
			END IF;
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS payloads_audit ON payloads;
		CREATE TRIGGER payloads_audit
		AFTER INSERT OR UPDATE OR DELETE ON payloads
		FOR EACH ROW EXECUTE FUNCTION audit_payloads();

		CREATE OR REPLACE FUNCTION audit_servers() RETURNS TRIGGER AS $$
		BEGIN
			IF TG_OP = 'DELETE' THEN
				INSERT INTO servers_history(operation,id,address,"user",key,status,deploying,deploying_at,owner_id,created_at,updated_at,deleted_at)
				VALUES('D',OLD.id,OLD.address,OLD.user,OLD.key,OLD.status,OLD.deploying,OLD.deploying_at,OLD.owner_id,OLD.created_at,OLD.updated_at,OLD.deleted_at);
			ELSE
				INSERT INTO servers_history(operation,id,address,"user",key,status,deploying,deploying_at,owner_id,created_at,updated_at,deleted_at)
				VALUES(CASE WHEN TG_OP='INSERT' THEN 'I' ELSE 'U' END,NEW.id,NEW.address,NEW.user,NEW.key,NEW.status,NEW.deploying,NEW.deploying_at,NEW.owner_id,NEW.created_at,NEW.updated_at,NEW.deleted_at);
			END IF;
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS servers_audit ON servers;
		CREATE TRIGGER servers_audit
		AFTER INSERT OR UPDATE OR DELETE ON servers
		FOR EACH ROW EXECUTE FUNCTION audit_servers();
	`).Error; err != nil {
		return fmt.Errorf("failed to run audit migrations: %w", err)
	}
	slog.Info("Tenant migrations completed")
	return nil
}
