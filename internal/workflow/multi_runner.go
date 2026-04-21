package workflow

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
)

type MultiRunner struct {
	rootDB        *gorm.DB
	newTenant     func(string) (*gorm.DB, error)
	workerID      uuid.UUID
	lease         time.Duration
	maxConcurrent int
	syncInterval  time.Duration
	configure     func(*Runner)
}

type MultiRunnerOptions struct {
	WorkerID      uuid.UUID
	Lease         time.Duration
	MaxConcurrent int
	SyncInterval  time.Duration
}

func NewMultiRunner(rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error), options MultiRunnerOptions, configure func(*Runner)) *MultiRunner {
	lease := options.Lease
	if lease <= 0 {
		lease = 10 * time.Minute
	}
	maxConcurrent := options.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 8
	}
	syncInterval := options.SyncInterval
	if syncInterval <= 0 {
		syncInterval = 5 * time.Second
	}
	return &MultiRunner{
		rootDB:        rootDB,
		newTenant:     newTenant,
		workerID:      options.WorkerID,
		lease:         lease,
		maxConcurrent: maxConcurrent,
		syncInterval:  syncInterval,
		configure:     configure,
	}
}

func (m *MultiRunner) assignedTenants(ctx context.Context) ([]models.Tenant, error) {
	var tenants []models.Tenant
	err := m.rootDB.WithContext(ctx).
		Model(&models.Tenant{}).
		Distinct("tenants.*").
		Joins("JOIN worker_tenant_assignments ON worker_tenant_assignments.tenant_id = tenants.id").
		Joins("JOIN workers ON workers.id = worker_tenant_assignments.worker_id").
		Where("worker_tenant_assignments.worker_id = ?", m.workerID).
		Where("workers.deleted_at IS NULL").
		Find(&tenants).Error
	return tenants, err
}

func (m *MultiRunner) Run(ctx context.Context) {
	type tenantRunner struct {
		database string
		cancel   context.CancelFunc
	}

	runners := make(map[string]tenantRunner)
	ticker := time.NewTicker(m.syncInterval)
	defer ticker.Stop()

	syncRunners := func() {
		tenants, err := m.assignedTenants(ctx)
		if err != nil {
			slog.Error("failed to load assigned tenants for worker runners", "worker_id", m.workerID, "error", err)
			return
		}

		active := make(map[string]struct{}, len(tenants))
		for _, tenant := range tenants {
			tenantKey := tenant.ID.String()
			active[tenantKey] = struct{}{}

			current, ok := runners[tenantKey]
			if ok && current.database == tenant.Database {
				continue
			}
			if ok {
				current.cancel()
				delete(runners, tenantKey)
			}

			tenantDB, err := m.newTenant(tenant.Database)
			if err != nil {
				slog.Error("failed to connect tenant runner database", "worker_id", m.workerID, "tenant_id", tenant.ID, "error", err)
				continue
			}

			runner, err := NewRunner(tenantDB, m.lease, m.maxConcurrent)
			if err != nil {
				slog.Error("failed to create tenant runner", "worker_id", m.workerID, "tenant_id", tenant.ID, "error", err)
				continue
			}
			if m.configure != nil {
				m.configure(runner)
			}

			tenantCtx, tenantCancel := context.WithCancel(ctx)
			runners[tenantKey] = tenantRunner{
				database: tenant.Database,
				cancel:   tenantCancel,
			}
			slog.Info("started tenant runner", "worker_id", m.workerID, "tenant_id", tenant.ID, "tenant_name", tenant.Name)
			go runner.Run(tenantCtx)
		}

		for tenantKey, runner := range runners {
			if _, ok := active[tenantKey]; ok {
				continue
			}
			slog.Info("stopped tenant runner", "worker_id", m.workerID, "tenant_id", tenantKey)
			runner.cancel()
			delete(runners, tenantKey)
		}
	}

	syncRunners()
	for {
		select {
		case <-ctx.Done():
			for _, runner := range runners {
				runner.cancel()
			}
			return
		case <-ticker.C:
			syncRunners()
		}
	}
}
