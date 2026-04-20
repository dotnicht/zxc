package workflow

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"zxc/internal/models"
)

func mockWorkflowDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db, mock
}

func TestRootTransaction(t *testing.T) {
	db, mock := mockWorkflowDB(t)
	store := NewStore(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT 1")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := store.RootTransaction(context.Background(), func(tx *gorm.DB) error {
		return tx.Exec("SELECT 1").Error
	}); err != nil {
		t.Fatalf("RootTransaction returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestRootTransactionRollback(t *testing.T) {
	db, mock := mockWorkflowDB(t)
	store := NewStore(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT 1")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	err := store.RootTransaction(context.Background(), func(tx *gorm.DB) error {
		if execErr := tx.Exec("SELECT 1").Error; execErr != nil {
			return execErr
		}
		return context.Canceled
	})
	if err == nil {
		t.Fatalf("expected callback error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestEnqueueCommandWithoutDedupeUsesDefaults(t *testing.T) {
	db, mock := mockWorkflowDB(t)
	store := NewStore(db)
	tenantID := uuid.New()

	mock.ExpectExec(`INSERT INTO commands\(kind, aggregate_type, aggregate_id, tenant_id, payload, status, run_at, attempt, max_attempts, is_active, created_at, updated_at\)\s+VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, 0, \$8, TRUE, NOW\(\), NOW\(\)\)`).
		WithArgs("release_mark_alive", "release", "rel-1", &tenantID, []byte(`{"ok":true}`), models.CommandPending, sqlmock.AnyArg(), 5).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.EnqueueCommand(context.Background(), nil, CommandInput{
		Kind:          "release_mark_alive",
		AggregateType: "release",
		AggregateID:   "rel-1",
		TenantID:      &tenantID,
		Payload:       map[string]bool{"ok": true},
	})
	if err != nil {
		t.Fatalf("EnqueueCommand returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestEnqueueCommandWithDedupe(t *testing.T) {
	db, mock := mockWorkflowDB(t)
	store := NewStore(db)
	tenantID := uuid.New()
	runAt := time.Unix(1_700_000_000, 0).UTC()

	mock.ExpectExec(`INSERT INTO commands\(kind, aggregate_type, aggregate_id, tenant_id, payload, status, run_at, attempt, max_attempts, dedupe_key, is_active, created_at, updated_at\)\s+VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, 0, \$8, \$9, TRUE, NOW\(\), NOW\(\)\)\s+ON CONFLICT \(dedupe_key\) WHERE is_active = TRUE\s+DO UPDATE SET\s+run_at = LEAST\(commands.run_at, EXCLUDED.run_at\),\s+payload = EXCLUDED.payload,\s+tenant_id = EXCLUDED.tenant_id,\s+aggregate_type = EXCLUDED.aggregate_type,\s+aggregate_id = EXCLUDED.aggregate_id,\s+updated_at = NOW\(\),\s+last_error = NULL`).
		WithArgs("release_health_timeout", "release", "rel-1", &tenantID, []byte(`{"retry":1}`), models.CommandPending, runAt, 1, "release-health:rel-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.EnqueueCommand(context.Background(), nil, CommandInput{
		Kind:          "release_health_timeout",
		AggregateType: "release",
		AggregateID:   "rel-1",
		TenantID:      &tenantID,
		Payload:       map[string]int{"retry": 1},
		RunAt:         runAt,
		MaxAttempts:   1,
		DedupeKey:     "release-health:rel-1",
	})
	if err != nil {
		t.Fatalf("EnqueueCommand returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
