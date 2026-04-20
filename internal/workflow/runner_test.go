package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"zxc/internal/models"
)

func mockRunnerDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestRegister(t *testing.T) {
	type args struct {
		Name string `json:"name"`
	}

	r := &Runner{handlers: make(map[string]func(context.Context, json.RawMessage, int, int) error)}
	var got Job[args]

	Register(r, "hello", func(_ context.Context, job *Job[args]) error {
		got = *job
		return nil
	})

	err := r.handlers["hello"](context.Background(), []byte(`{"name":"alice"}`), 2, 5)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if got.Args.Name != "alice" || got.Attempt != 2 || got.MaxAttempts != 5 {
		t.Fatalf("unexpected job: %+v", got)
	}
}

func TestRegisterRejectsInvalidJSON(t *testing.T) {
	type args struct {
		Value string `json:"value"`
	}

	r := &Runner{handlers: make(map[string]func(context.Context, json.RawMessage, int, int) error)}
	Register(r, "bad", func(_ context.Context, _ *Job[args]) error { return nil })

	err := r.handlers["bad"](context.Background(), []byte("{"), 1, 3)
	if err == nil || !strings.Contains(err.Error(), "unmarshal bad payload") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
}

func TestExecuteMarksMissingHandlerFailed(t *testing.T) {
	db, mock := mockRunnerDB(t)
	r := &Runner{db: db, handlers: map[string]func(context.Context, json.RawMessage, int, int) error{}}

	mock.ExpectExec(`UPDATE commands SET status = \$1, is_active = FALSE, lease_until = NULL, finished_at = NOW\(\), last_error = \$2, updated_at = NOW\(\) WHERE id = \$3`).
		WithArgs(models.CommandFailed, "no handler registered", int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r.execute(context.Background(), rawJob{ID: 7, Kind: "missing"})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestExecuteMarksDoneOnSuccess(t *testing.T) {
	db, mock := mockRunnerDB(t)
	r := &Runner{
		db: db,
		handlers: map[string]func(context.Context, json.RawMessage, int, int) error{
			"ok": func(context.Context, json.RawMessage, int, int) error { return nil },
		},
	}

	mock.ExpectExec(`UPDATE commands SET status = \$1, is_active = FALSE, lease_until = NULL, finished_at = NOW\(\), updated_at = NOW\(\) WHERE id = \$2`).
		WithArgs(models.CommandDone, int64(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r.execute(context.Background(), rawJob{ID: 8, Kind: "ok"})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestExecuteRetriesRegularError(t *testing.T) {
	db, mock := mockRunnerDB(t)
	r := &Runner{
		db: db,
		handlers: map[string]func(context.Context, json.RawMessage, int, int) error{
			"retry": func(context.Context, json.RawMessage, int, int) error { return errors.New("boom") },
		},
	}

	mock.ExpectExec(`UPDATE commands SET status = \$1, lease_until = NULL, run_at = \$2, last_error = \$3, updated_at = NOW\(\) WHERE id = \$4`).
		WithArgs(models.CommandPending, sqlmock.AnyArg(), "boom", int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r.execute(context.Background(), rawJob{ID: 9, Kind: "retry", Attempt: 1, MaxAttempts: 3})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestExecuteFailsWhenAttemptsExhausted(t *testing.T) {
	db, mock := mockRunnerDB(t)
	r := &Runner{
		db: db,
		handlers: map[string]func(context.Context, json.RawMessage, int, int) error{
			"fail": func(context.Context, json.RawMessage, int, int) error { return errors.New("boom") },
		},
	}

	mock.ExpectExec(`UPDATE commands SET status = \$1, is_active = FALSE, lease_until = NULL, finished_at = NOW\(\), last_error = \$2, updated_at = NOW\(\) WHERE id = \$3`).
		WithArgs(models.CommandFailed, "boom", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r.execute(context.Background(), rawJob{ID: 10, Kind: "fail", Attempt: 3, MaxAttempts: 3})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestExecuteReschedulesSnooze(t *testing.T) {
	db, mock := mockRunnerDB(t)
	until := time.Unix(1_700_000_000, 0).UTC()
	r := &Runner{
		db: db,
		handlers: map[string]func(context.Context, json.RawMessage, int, int) error{
			"snooze": func(context.Context, json.RawMessage, int, int) error { return SnoozeErr{Until: until} },
		},
	}

	mock.ExpectExec(`UPDATE commands SET status = \$1, lease_until = NULL, run_at = \$2, attempt = GREATEST\(attempt - 1, 0\), updated_at = NOW\(\) WHERE id = \$3`).
		WithArgs(models.CommandPending, until, int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r.execute(context.Background(), rawJob{ID: 11, Kind: "snooze"})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestBackoffAndHelpers(t *testing.T) {
	if got := backoff(0); got != time.Second {
		t.Fatalf("backoff(0)=%v want %v", got, time.Second)
	}
	if got := backoff(1); got != time.Second {
		t.Fatalf("backoff(1)=%v want %v", got, time.Second)
	}
	if got := backoff(3); got != 4*time.Second {
		t.Fatalf("backoff(3)=%v want %v", got, 4*time.Second)
	}
	if got := backoff(10); got != 32*time.Second {
		t.Fatalf("backoff(10)=%v want %v", got, 32*time.Second)
	}
	if got := min(2, 7); got != 2 {
		t.Fatalf("min(2,7)=%d want 2", got)
	}
	if got := min(9, 1); got != 1 {
		t.Fatalf("min(9,1)=%d want 1", got)
	}
}
