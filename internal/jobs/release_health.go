package jobs

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/events"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

type ReleaseHealthTimeoutArgs struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	ReleaseID uuid.UUID `json:"release_id"`
}

type ReleaseHealthWorker struct {
	store     *workflow.Store
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
}

func NewReleaseHealthWorker(store *workflow.Store, rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error)) *ReleaseHealthWorker {
	return &ReleaseHealthWorker{store: store, rootDB: rootDB, newTenant: newTenant}
}

func (w *ReleaseHealthWorker) Work(ctx context.Context, job *workflow.Job[ReleaseHealthTimeoutArgs]) error {
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
	case models.ReleaseWait, models.ReleaseDeployed:
		previousStatus := release.Status
		if err := authorizeReleaseTransition(ctx, &tenant, previousStatus, models.ReleaseDead); err != nil {
			return err
		}
		result := db.WithContext(ctx).Model(&models.Release{}).
			Where("id = ? AND status IN ?", release.ID, []string{models.ReleaseWait, models.ReleaseDeployed}).
			Update("status", models.ReleaseDead)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		if err := w.store.RecordEvent(ctx, db, events.ReleaseHealthTimeout{
			ReleaseID: release.ID,
		}); err != nil {
			revertErr := db.WithContext(ctx).Model(&models.Release{}).
				Where("id = ? AND status = ?", release.ID, models.ReleaseDead).
				Update("status", previousStatus).Error
			return errors.Join(err, revertErr)
		}
		return nil
	default:
		return nil
	}
}
