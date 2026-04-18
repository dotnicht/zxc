package jobs

import (
	"context"

	"gorm.io/gorm"
	"zxc/internal/models"
	"zxc/internal/queue"
)

type AliveCheckScanArgs struct{}

func (AliveCheckScanArgs) Kind() string { return "alive_check_scan" }

type AliveCheckScanWorker struct {
	rootDB *gorm.DB
	q      *queue.Queue
}

func NewAliveCheckScanWorker(rootDB *gorm.DB, q *queue.Queue) *AliveCheckScanWorker {
	return &AliveCheckScanWorker{rootDB: rootDB, q: q}
}

func (w *AliveCheckScanWorker) Work(ctx context.Context, job *queue.Job[AliveCheckScanArgs]) error {
	var tenants []*models.Tenant
	if err := w.rootDB.WithContext(ctx).Limit(1000).Find(&tenants).Error; err != nil {
		return err
	}
	for _, t := range tenants {
		if err := w.q.Insert(ctx, TenantCheckArgs{TenantID: t.ID}); err != nil {
			return err
		}
	}
	return nil
}
