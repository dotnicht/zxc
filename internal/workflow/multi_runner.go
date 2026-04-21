package workflow

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"zxc/internal/models"
)

type MultiRunner struct {
	rootDB         *gorm.DB
	newTenant      func(string) (*gorm.DB, error)
	lease          time.Duration
	maxConcurrent  int
	syncInterval   time.Duration
	allowTenantIDs map[string]struct{}
	denyTenantIDs  map[string]struct{}
	configure      func(*Runner)
}

type MultiRunnerOptions struct {
	Lease          time.Duration
	MaxConcurrent  int
	SyncInterval   time.Duration
	AllowTenantIDs []string
	DenyTenantIDs  []string
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
		rootDB:         rootDB,
		newTenant:      newTenant,
		lease:          lease,
		maxConcurrent:  maxConcurrent,
		syncInterval:   syncInterval,
		allowTenantIDs: listToSet(options.AllowTenantIDs),
		denyTenantIDs:  listToSet(options.DenyTenantIDs),
		configure:      configure,
	}
}

func listToSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func (m *MultiRunner) matchesTenant(tenant models.Tenant) bool {
	tenantID := tenant.ID.String()

	if len(m.allowTenantIDs) > 0 {
		if _, ok := m.allowTenantIDs[tenantID]; !ok {
			return false
		}
	}

	if _, ok := m.denyTenantIDs[tenantID]; ok {
		return false
	}

	return true
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
		var tenants []models.Tenant
		if err := m.rootDB.WithContext(ctx).Find(&tenants).Error; err != nil {
			slog.Error("failed to load tenants for worker runners", "error", err)
			return
		}

		active := make(map[string]struct{}, len(tenants))
		for _, tenant := range tenants {
			tenantKey := tenant.ID.String()
			if !m.matchesTenant(tenant) {
				if current, ok := runners[tenantKey]; ok {
					current.cancel()
					delete(runners, tenantKey)
				}
				continue
			}
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
				slog.Error("failed to connect tenant runner database", "tenant_id", tenant.ID, "error", err)
				continue
			}

			runner, err := NewRunner(tenantDB, m.lease, m.maxConcurrent)
			if err != nil {
				slog.Error("failed to create tenant runner", "tenant_id", tenant.ID, "error", err)
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
			go runner.Run(tenantCtx)
		}

		for tenantKey, runner := range runners {
			if _, ok := active[tenantKey]; ok {
				continue
			}
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
