package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
)

const defaultMaxAttempts = 5

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) RootTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx.WithContext(ctx))
	})
}

type EventInput struct {
	Kind          string
	AggregateType string
	AggregateID   string
	TenantID      *uuid.UUID
	Payload       any
}

type CommandInput struct {
	Kind          string
	AggregateType string
	AggregateID   string
	TenantID      *uuid.UUID
	Payload       any
	RunAt         time.Time
	MaxAttempts   int
	DedupeKey     string
}

func (s *Store) RecordEvent(ctx context.Context, tx *gorm.DB, in EventInput) error {
	body, err := json.Marshal(in.Payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	db := s.db.WithContext(ctx)
	if tx != nil {
		db = tx.WithContext(ctx)
	}
	return db.Create(&models.Event{
		Kind:          in.Kind,
		AggregateType: in.AggregateType,
		AggregateID:   in.AggregateID,
		TenantID:      in.TenantID,
		Payload:       body,
	}).Error
}

func (s *Store) EnqueueCommand(ctx context.Context, tx *gorm.DB, in CommandInput) error {
	body, err := json.Marshal(in.Payload)
	if err != nil {
		return fmt.Errorf("marshal command payload: %w", err)
	}

	runAt := in.RunAt.UTC()
	if runAt.IsZero() {
		runAt = time.Now().UTC()
	}

	maxAttempts := in.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}

	db := s.db.WithContext(ctx)
	if tx != nil {
		db = tx.WithContext(ctx)
	}
	if in.DedupeKey == "" {
		return db.Exec(`
			INSERT INTO commands(kind, aggregate_type, aggregate_id, tenant_id, payload, status, run_at, attempt, max_attempts, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, TRUE, NOW(), NOW())
		`, in.Kind, in.AggregateType, in.AggregateID, in.TenantID, body, models.CommandPending, runAt, maxAttempts).Error
	}

	return db.Exec(`
		INSERT INTO commands(kind, aggregate_type, aggregate_id, tenant_id, payload, status, run_at, attempt, max_attempts, dedupe_key, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, TRUE, NOW(), NOW())
		ON CONFLICT (dedupe_key) WHERE is_active = TRUE
		DO UPDATE SET
			run_at = LEAST(commands.run_at, EXCLUDED.run_at),
			payload = EXCLUDED.payload,
			tenant_id = EXCLUDED.tenant_id,
			aggregate_type = EXCLUDED.aggregate_type,
			aggregate_id = EXCLUDED.aggregate_id,
			updated_at = NOW(),
			last_error = NULL
	`, in.Kind, in.AggregateType, in.AggregateID, in.TenantID, body, models.CommandPending, runAt, maxAttempts, in.DedupeKey).Error
}
