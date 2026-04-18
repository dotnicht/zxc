package jobs

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
	"zxc/internal/queue"
)

type TenantDeployArgs struct {
	TenantID uuid.UUID `json:"tenant_id"`
}

func (TenantDeployArgs) Kind() string { return "tenant_deploy" }

type TenantDeployWorker struct {
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
	q         *queue.Queue
}

func NewTenantDeployWorker(rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error), q *queue.Queue) *TenantDeployWorker {
	return &TenantDeployWorker{rootDB: rootDB, newTenant: newTenant, q: q}
}

func (w *TenantDeployWorker) Work(ctx context.Context, job *queue.Job[TenantDeployArgs]) error {
	var tenant models.Tenant
	if err := w.rootDB.WithContext(ctx).First(&tenant, "id = ?", job.Args.TenantID).Error; err != nil {
		return err
	}

	db, err := w.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var releases []models.Release
	if err := db.WithContext(ctx).Where("status = ? AND deleted_at IS NULL", models.ReleaseWait).Find(&releases).Error; err != nil {
		return err
	}

	for _, r := range releases {
		if err := w.q.Insert(ctx, DeployArgs{ReleaseID: r.ID, ChangedByID: r.ChangedByID}); err != nil {
			return err
		}
	}
	return nil
}
