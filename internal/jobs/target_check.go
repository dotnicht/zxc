package jobs

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/deployer"
	"zxc/internal/models"
	"zxc/internal/queue"
)

type TargetCheckArgs struct {
	TargetID uuid.UUID `json:"target_id"`
	TenantID uuid.UUID `json:"tenant_id"`
}

func (TargetCheckArgs) Kind() string { return "target_check" }

type TargetCheckWorker struct {
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
}

func NewTargetCheckWorker(rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error)) *TargetCheckWorker {
	return &TargetCheckWorker{rootDB: rootDB, newTenant: newTenant}
}

func (w *TargetCheckWorker) Work(ctx context.Context, job *queue.Job[TargetCheckArgs]) error {
	var tenant models.Tenant
	if err := w.rootDB.WithContext(ctx).First(&tenant, "id = ?", job.Args.TenantID).Error; err != nil {
		return err
	}

	db, err := w.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var t models.Target
	if err := db.First(&t, "id = ?", job.Args.TargetID).Error; err != nil {
		return err
	}

	newStatus := models.TargetOnline
	if err := deployer.Ping(ctx, t.Address, t.User, []byte(t.Key)); err != nil {
		newStatus = models.TargetOffline
	}

	return db.Model(&models.Target{}).Where("id = ?", job.Args.TargetID).Update("status", newStatus).Error
}
