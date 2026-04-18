package jobs

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
	"zxc/internal/queue"
)

type TenantTargetCheckArgs struct {
	TenantID uuid.UUID `json:"tenant_id"`
}

func (TenantTargetCheckArgs) Kind() string { return "tenant_target_check" }

type TenantTargetCheckWorker struct {
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
	q         *queue.Queue
}

func NewTenantTargetCheckWorker(rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error), q *queue.Queue) *TenantTargetCheckWorker {
	return &TenantTargetCheckWorker{rootDB: rootDB, newTenant: newTenant, q: q}
}

func (w *TenantTargetCheckWorker) Work(ctx context.Context, job *queue.Job[TenantTargetCheckArgs]) error {
	var tenant models.Tenant
	if err := w.rootDB.WithContext(ctx).First(&tenant, "id = ?", job.Args.TenantID).Error; err != nil {
		return err
	}

	db, err := w.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var targets []*models.Target
	if err := db.WithContext(ctx).Limit(1000).Find(&targets).Error; err != nil {
		return err
	}

	for _, t := range targets {
		if err := w.q.Insert(ctx, TargetCheckArgs{TargetID: t.ID, TenantID: job.Args.TenantID}); err != nil {
			return err
		}
	}
	return nil
}
