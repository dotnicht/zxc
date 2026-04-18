package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
	"zxc/internal/queue"
)

type TenantCheckArgs struct {
	TenantID uuid.UUID `json:"tenant_id"`
}

func (TenantCheckArgs) Kind() string { return "tenant_check" }

type TenantCheckWorker struct {
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
	q         *queue.Queue
}

func NewTenantCheckWorker(rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error), q *queue.Queue) *TenantCheckWorker {
	return &TenantCheckWorker{rootDB: rootDB, newTenant: newTenant, q: q}
}

func (w *TenantCheckWorker) Work(ctx context.Context, job *queue.Job[TenantCheckArgs]) error {
	var tenant models.Tenant
	if err := w.rootDB.WithContext(ctx).First(&tenant, "id = ?", job.Args.TenantID).Error; err != nil {
		return err
	}

	db, err := w.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var releases []models.Release
	if err := db.WithContext(ctx).Where("status = ? AND deleted_at IS NULL", models.ReleaseDeployed).Find(&releases).Error; err != nil {
		return err
	}

	now := time.Now()
	for _, r := range releases {
		if err := w.q.Insert(ctx, CheckArgs{ReleaseID: r.ID, EnqueuedAt: now}); err != nil {
			return err
		}
	}
	return nil
}
