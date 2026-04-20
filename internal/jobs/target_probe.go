package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/deployer"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

type TargetProbeArgs struct {
	TenantID uuid.UUID `json:"tenant_id"`
	TargetID uuid.UUID `json:"target_id"`
}

type TargetProbeWorker struct {
	store     *workflow.Store
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
}

func NewTargetProbeWorker(store *workflow.Store, rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error)) *TargetProbeWorker {
	return &TargetProbeWorker{store: store, rootDB: rootDB, newTenant: newTenant}
}

func (w *TargetProbeWorker) Work(ctx context.Context, job *workflow.Job[TargetProbeArgs]) error {
	var tenant models.Tenant
	if err := w.rootDB.WithContext(ctx).First(&tenant, "id = ?", job.Args.TenantID).Error; err != nil {
		return err
	}

	db, err := w.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var target models.Target
	if err := db.WithContext(ctx).First(&target, "id = ?", job.Args.TargetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	newStatus := models.TargetOnline
	eventKind := "target_probe_succeeded"
	if err := deployer.Ping(ctx, target.Address, target.User, []byte(target.Key)); err != nil {
		newStatus = models.TargetOffline
		eventKind = "target_probe_failed"
	}

	if err := db.WithContext(ctx).Model(&models.Target{}).
		Where("id = ? AND status <> ?", target.ID, newStatus).
		Update("status", newStatus).Error; err != nil {
		return err
	}

	if err := w.store.RecordEvent(ctx, nil, workflow.EventInput{
		Kind:          eventKind,
		AggregateType: "target",
		AggregateID:   target.ID.String(),
		TenantID:      &job.Args.TenantID,
		Payload: map[string]any{
			"target_id": target.ID.String(),
			"status":    newStatus,
		},
	}); err != nil {
		return err
	}

	return workflow.Snooze(30 * time.Second)
}
