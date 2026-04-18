package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const defaultMaxAttempts = 5

type SnoozeErr struct{ Until time.Time }

func (e SnoozeErr) Error() string { return fmt.Sprintf("snooze until %v", e.Until) }

func Snooze(d time.Duration) error { return SnoozeErr{Until: time.Now().Add(d)} }

type Job[T any] struct {
	ID          int64
	Args        T
	Attempt     int
	MaxAttempts int
}

type Args interface {
	Kind() string
}

type Queue struct {
	db       *sql.DB
	handlers map[string]func(ctx context.Context, raw json.RawMessage, attempt, maxAttempts int) error
}

func New(db *sql.DB) *Queue {
	return &Queue{
		db:       db,
		handlers: make(map[string]func(context.Context, json.RawMessage, int, int) error),
	}
}

func Register[T any](q *Queue, kind string, fn func(ctx context.Context, job *Job[T]) error) {
	q.handlers[kind] = func(ctx context.Context, raw json.RawMessage, attempt, maxAttempts int) error {
		var args T
		if err := json.Unmarshal(raw, &args); err != nil {
			return fmt.Errorf("unmarshal %s args: %w", kind, err)
		}
		return fn(ctx, &Job[T]{Args: args, Attempt: attempt, MaxAttempts: maxAttempts})
	}
}

func (q *Queue) Insert(ctx context.Context, args Args) error {
	b, err := json.Marshal(args)
	if err != nil {
		return err
	}
	_, err = q.db.ExecContext(ctx,
		`INSERT INTO jobs(kind, args, status, attempt, max_attempts, scheduled_at)
		 VALUES($1, $2, 'pending', 0, $3, NOW())`,
		args.Kind(), b, defaultMaxAttempts,
	)
	return err
}

func (q *Queue) Run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.poll(ctx)
		}
	}
}

func (q *Queue) poll(ctx context.Context) {
	row := q.db.QueryRowContext(ctx, `
		UPDATE jobs
		SET status = 'running', locked_until = NOW() + interval '10 minutes', attempt = attempt + 1
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = 'pending'
			  AND scheduled_at <= NOW()
			  AND attempt < max_attempts
			ORDER BY scheduled_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, kind, args, attempt, max_attempts
	`)
	var id int64
	var kind string
	var raw json.RawMessage
	var attempt, maxAttempts int
	if err := row.Scan(&id, &kind, &raw, &attempt, &maxAttempts); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Error("queue poll", "error", err)
		}
		return
	}

	handler, ok := q.handlers[kind]
	if !ok {
		slog.Warn("no handler", "kind", kind)
		q.markFailed(id, "no handler registered")
		return
	}

	go func() {
		if err := handler(ctx, raw, attempt, maxAttempts); err != nil {
			var snooze SnoozeErr
			if errors.As(err, &snooze) {
				q.reschedule(id, snooze.Until)
				return
			}
			slog.Error("job failed", "kind", kind, "id", id, "attempt", attempt, "error", err)
			if attempt >= maxAttempts {
				q.markFailed(id, err.Error())
			} else {
				q.markPending(id)
			}
			return
		}
		q.markDone(id)
	}()
}

func (q *Queue) markDone(id int64) {
	if _, err := q.db.Exec(`UPDATE jobs SET status = 'done', finished_at = NOW() WHERE id = $1`, id); err != nil {
		slog.Error("markDone", "id", id, "error", err)
	}
}

func (q *Queue) markFailed(id int64, msg string) {
	if _, err := q.db.Exec(`UPDATE jobs SET status = 'failed', finished_at = NOW(), error = $2 WHERE id = $1`, id, msg); err != nil {
		slog.Error("markFailed", "id", id, "error", err)
	}
}

func (q *Queue) markPending(id int64) {
	if _, err := q.db.Exec(`UPDATE jobs SET status = 'pending' WHERE id = $1`, id); err != nil {
		slog.Error("markPending", "id", id, "error", err)
	}
}

func (q *Queue) reschedule(id int64, until time.Time) {
	if _, err := q.db.Exec(`UPDATE jobs SET status = 'pending', scheduled_at = $2 WHERE id = $1`, id, until); err != nil {
		slog.Error("reschedule", "id", id, "error", err)
	}
}
