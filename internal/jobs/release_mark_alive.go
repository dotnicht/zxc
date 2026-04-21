package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

type ReleaseMarkAliveArgs struct {
	TenantID  uuid.UUID       `json:"tenant_id"`
	ReleaseID uuid.UUID       `json:"release_id"`
	Body      json.RawMessage `json:"body"`
}

type ReleaseMarkAliveWorker struct {
	store     *workflow.Store
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
}

func NewReleaseMarkAliveWorker(store *workflow.Store, rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error)) *ReleaseMarkAliveWorker {
	return &ReleaseMarkAliveWorker{store: store, rootDB: rootDB, newTenant: newTenant}
}

func (w *ReleaseMarkAliveWorker) Work(ctx context.Context, job *workflow.Job[ReleaseMarkAliveArgs]) error {
	var tenant models.Tenant
	if err := w.rootDB.WithContext(ctx).First(&tenant, "id = ?", job.Args.TenantID).Error; err != nil {
		return err
	}

	db, err := w.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var release models.Release
	if err := db.WithContext(ctx).First(&release, "id = ?", job.Args.ReleaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	switch release.Status {
	case models.ReleaseAlive, models.ReleaseDead:
		return nil
	case models.ReleaseWait:
		return workflow.Snooze(time.Second)
	case models.ReleaseDeployed:
		if sinceDeploy := time.Since(release.UpdatedAt); sinceDeploy < 3*time.Second {
			return workflow.Snooze(3*time.Second - sinceDeploy)
		}
		if err := authorizeReleaseTransition(ctx, &tenant, models.ReleaseDeployed, models.ReleaseAlive); err != nil {
			return err
		}
		result := db.WithContext(ctx).Model(&models.Release{}).
			Where("id = ? AND status = ?", release.ID, models.ReleaseDeployed).
			Update("status", models.ReleaseAlive)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := w.store.RecordEvent(ctx, db, workflow.EventInput{
			Kind:          "release_alive",
			AggregateType: "release",
			AggregateID:   release.ID,
			Payload: map[string]any{
				"release_id": release.ID.String(),
				"body":       job.Args.Body,
			},
		}); err != nil {
			revertErr := db.WithContext(ctx).Model(&models.Release{}).
				Where("id = ? AND status = ?", release.ID, models.ReleaseAlive).
				Update("status", models.ReleaseDeployed).Error
			return errors.Join(err, revertErr)
		}
		return nil
	default:
		return nil
	}
}
