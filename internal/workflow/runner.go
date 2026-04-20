package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"zxc/internal/models"
)

type SnoozeErr struct {
	Until time.Time
}

func (e SnoozeErr) Error() string {
	return fmt.Sprintf("snooze until %s", e.Until.Format(time.RFC3339))
}

func Snooze(d time.Duration) error {
	return SnoozeErr{Until: time.Now().UTC().Add(d)}
}

type Job[T any] struct {
	ID          int64
	Args        T
	Attempt     int
	MaxAttempts int
}

type rawJob struct {
	ID          int64
	Kind        string
	Payload     json.RawMessage
	Attempt     int
	MaxAttempts int
}

type Runner struct {
	db            *gorm.DB
	sqlDB         *sql.DB
	lease         time.Duration
	maxConcurrent int
	handlers      map[string]func(ctx context.Context, raw json.RawMessage, attempt, maxAttempts int) error
}

func NewRunner(db *gorm.DB, lease time.Duration, maxConcurrent int) (*Runner, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if lease <= 0 {
		lease = 10 * time.Minute
	}
	if maxConcurrent <= 0 {
		maxConcurrent = 8
	}
	return &Runner{
		db:            db,
		sqlDB:         sqlDB,
		lease:         lease,
		maxConcurrent: maxConcurrent,
		handlers:      make(map[string]func(context.Context, json.RawMessage, int, int) error),
	}, nil
}

func Register[T any](r *Runner, kind string, fn func(ctx context.Context, job *Job[T]) error) {
	r.handlers[kind] = func(ctx context.Context, raw json.RawMessage, attempt, maxAttempts int) error {
		var args T
		if err := json.Unmarshal(raw, &args); err != nil {
			return fmt.Errorf("unmarshal %s payload: %w", kind, err)
		}
		return fn(ctx, &Job[T]{Args: args, Attempt: attempt, MaxAttempts: maxAttempts})
	}
}

func (r *Runner) Run(ctx context.Context) {
	sem := make(chan struct{}, r.maxConcurrent)
	idleDelay := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case sem <- struct{}{}:
		default:
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
				continue
			}
		}

		job, ok, err := r.claim(ctx)
		if err != nil {
			<-sem
			slog.Error("claim command", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(idleDelay):
			}
			continue
		}
		if !ok {
			<-sem
			select {
			case <-ctx.Done():
				return
			case <-time.After(idleDelay):
			}
			continue
		}

		go func(job rawJob) {
			defer func() { <-sem }()
			r.execute(ctx, job)
		}(job)
	}
}

func (r *Runner) claim(ctx context.Context) (rawJob, bool, error) {
	row := r.sqlDB.QueryRowContext(ctx, `
		WITH next AS (
			SELECT id
			FROM commands
			WHERE run_at <= NOW()
			  AND attempt < max_attempts
			  AND (
				status = 'pending'
				OR (status = 'running' AND lease_until IS NOT NULL AND lease_until < NOW())
			  )
			ORDER BY run_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE commands c
		SET status = 'running',
			lease_until = NOW() + ($1 * interval '1 second'),
			attempt = c.attempt + 1,
			updated_at = NOW(),
			last_error = NULL
		FROM next
		WHERE c.id = next.id
		RETURNING c.id, c.kind, c.payload, c.attempt, c.max_attempts
	`, int(r.lease.Seconds()))

	var job rawJob
	if err := row.Scan(&job.ID, &job.Kind, &job.Payload, &job.Attempt, &job.MaxAttempts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return rawJob{}, false, nil
		}
		return rawJob{}, false, err
	}
	return job, true, nil
}

func (r *Runner) execute(ctx context.Context, job rawJob) {
	handler, ok := r.handlers[job.Kind]
	if !ok {
		slog.Warn("no command handler", "kind", job.Kind, "command_id", job.ID)
		r.markFailed(job.ID, "no handler registered")
		return
	}

	if err := handler(ctx, job.Payload, job.Attempt, job.MaxAttempts); err != nil {
		var snooze SnoozeErr
		if errors.As(err, &snooze) {
			r.reschedule(job.ID, snooze.Until)
			return
		}
		slog.Error("command failed", "kind", job.Kind, "command_id", job.ID, "attempt", job.Attempt, "error", err)
		if job.Attempt >= job.MaxAttempts {
			r.markFailed(job.ID, err.Error())
			return
		}
		r.retry(job.ID, err.Error(), job.Attempt)
		return
	}

	r.markDone(job.ID)
}

func (r *Runner) markDone(id int64) {
	if err := r.db.Exec(`
		UPDATE commands
		SET status = ?, is_active = FALSE, lease_until = NULL, finished_at = NOW(), updated_at = NOW()
		WHERE id = ?
	`, models.CommandDone, id).Error; err != nil {
		slog.Error("mark command done", "command_id", id, "error", err)
	}
}

func (r *Runner) markFailed(id int64, msg string) {
	if err := r.db.Exec(`
		UPDATE commands
		SET status = ?, is_active = FALSE, lease_until = NULL, finished_at = NOW(), last_error = ?, updated_at = NOW()
		WHERE id = ?
	`, models.CommandFailed, msg, id).Error; err != nil {
		slog.Error("mark command failed", "command_id", id, "error", err)
	}
}

func (r *Runner) retry(id int64, msg string, attempt int) {
	delay := backoff(attempt)
	if err := r.db.Exec(`
		UPDATE commands
		SET status = ?, lease_until = NULL, run_at = ?, last_error = ?, updated_at = NOW()
		WHERE id = ?
	`, models.CommandPending, time.Now().UTC().Add(delay), msg, id).Error; err != nil {
		slog.Error("retry command", "command_id", id, "error", err)
	}
}

func (r *Runner) reschedule(id int64, until time.Time) {
	if err := r.db.Exec(`
		UPDATE commands
		SET status = ?, lease_until = NULL, run_at = ?, attempt = GREATEST(attempt - 1, 0), updated_at = NOW()
		WHERE id = ?
	`, models.CommandPending, until.UTC(), id).Error; err != nil {
		slog.Error("reschedule command", "command_id", id, "error", err)
	}
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		return time.Second
	}
	delay := time.Second << min(attempt-1, 5)
	if delay > time.Minute {
		return time.Minute
	}
	return delay
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
