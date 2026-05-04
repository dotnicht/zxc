package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/cschleiden/go-workflows/workflow"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/infra"
	"zxc/internal/models"
)

type ProbeArgs struct {
	TenantID uuid.UUID
	TargetID uuid.UUID
}

func Probe(ctx workflow.Context, args ProbeArgs) error {
	for {
		if _, err := workflow.ExecuteActivity[any](ctx, workflow.DefaultActivityOptions, ProbeActivity, args).Get(ctx); err != nil {
			return err
		}
		workflow.Sleep(ctx, 30*time.Second)
	}
}

type probeDeps struct {
	rootDB    *gorm.DB
	newDeploy func(string) (*gorm.DB, error)
}

var probeDep *probeDeps

func RegisterProbeDeps(rootDB *gorm.DB, newDeploy func(string) (*gorm.DB, error)) {
	probeDep = &probeDeps{rootDB: rootDB, newDeploy: newDeploy}
}

func ProbeActivity(ctx context.Context, args ProbeArgs) error {
	var tenant models.Tenant
	if err := probeDep.rootDB.WithContext(ctx).First(&tenant, "id = ?", args.TenantID).Error; err != nil {
		return err
	}

	db, err := probeDep.newDeploy(tenant.DeployDatabase)
	if err != nil {
		return err
	}

	var target models.Target
	if err := db.WithContext(ctx).First(&target, "id = ?", args.TargetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	newStatus := models.TargetOnline
	if err := infra.Ping(ctx, target.Address, target.User, []byte(target.Key)); err != nil {
		newStatus = models.TargetOffline
	}

	return db.WithContext(ctx).Model(&models.Target{}).
		Where("id = ? AND status <> ?", target.ID, newStatus).
		Update("status", newStatus).Error
}
