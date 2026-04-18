package jobs

import (
	"context"

	"gorm.io/gorm"
	"zxc/internal/models"
	"zxc/internal/queue"
)

type ScanArgs struct{}

func (ScanArgs) Kind() string { return "scan" }

type ScanWorker struct {
	rootDB *gorm.DB
	q      *queue.Queue
}

func NewScanWorker(rootDB *gorm.DB, q *queue.Queue) *ScanWorker {
	return &ScanWorker{rootDB: rootDB, q: q}
}

func (w *ScanWorker) Work(ctx context.Context, job *queue.Job[ScanArgs]) error {
	var tenants []*models.Tenant
	if err := w.rootDB.WithContext(ctx).Limit(1000).Find(&tenants).Error; err != nil {
		return err
	}
	for _, t := range tenants {
		if err := w.q.Insert(ctx, TenantDeployArgs{TenantID: t.ID}); err != nil {
			return err
		}
	}
	return nil
}
