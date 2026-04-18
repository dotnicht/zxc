package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
	"zxc/internal/queue"
)

type CheckArgs struct {
	ReleaseID  uuid.UUID `json:"release_id"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

func (CheckArgs) Kind() string { return "check" }

type CheckWorker struct {
	newTenant func(string) (*gorm.DB, error)
	rootDB    *gorm.DB
}

func NewCheckWorker(newTenant func(string) (*gorm.DB, error), rootDB *gorm.DB) *CheckWorker {
	return &CheckWorker{newTenant: newTenant, rootDB: rootDB}
}

func (w *CheckWorker) Work(ctx context.Context, job *queue.Job[CheckArgs]) error {
	id := job.Args.ReleaseID

	var route models.Route
	if err := w.rootDB.WithContext(ctx).Preload("Tenant").First(&route, "id = ?", id).Error; err != nil {
		return fmt.Errorf("load route for release %s: %w", id, err)
	}

	db, err := w.newTenant(route.Tenant.Database)
	if err != nil {
		return err
	}

	result := db.WithContext(ctx).
		Model(&models.Release{}).
		Where("id = ? AND status = ?", id, models.ReleaseDeployed).
		Update("status", models.ReleaseAlive)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var r models.Release
		if err := db.WithContext(ctx).Select("status").First(&r, "id = ?", id).Error; err != nil {
			return err
		}
		if r.Status == models.ReleaseWait {
			if !job.Args.EnqueuedAt.IsZero() && time.Since(job.Args.EnqueuedAt) > 2*time.Minute {
				return fmt.Errorf("release %s stuck in wait after %v, deploy likely failed",
					id, time.Since(job.Args.EnqueuedAt).Round(time.Second))
			}
			return queue.Snooze(3 * time.Second)
		}
	}
	return nil
}
