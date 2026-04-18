package jobs

import (
	"context"

	"gorm.io/gorm"
	"zxc/internal/models"
	"zxc/internal/queue"
)

type TargetScanArgs struct{}

func (TargetScanArgs) Kind() string { return "target_scan" }

type TargetScanWorker struct {
	rootDB *gorm.DB
	q      *queue.Queue
}

func NewTargetScanWorker(rootDB *gorm.DB, q *queue.Queue) *TargetScanWorker {
	return &TargetScanWorker{rootDB: rootDB, q: q}
}

func (w *TargetScanWorker) Work(ctx context.Context, job *queue.Job[TargetScanArgs]) error {
	var tenants []*models.Tenant
	if err := w.rootDB.WithContext(ctx).Limit(1000).Find(&tenants).Error; err != nil {
		return err
	}
	for _, t := range tenants {
		if err := w.q.Insert(ctx, TenantTargetCheckArgs{TenantID: t.ID}); err != nil {
			return err
		}
	}
	return nil
}
