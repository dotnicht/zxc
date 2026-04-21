package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/events"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

type AccountFromRequestArgs struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	RequestID uuid.UUID `json:"request_id"`
	ReleaseID uuid.UUID `json:"release_id"`
}

type AccountFromRequestWorker struct {
	store     *workflow.Store
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
}

func NewAccountFromRequestWorker(store *workflow.Store, rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error)) *AccountFromRequestWorker {
	return &AccountFromRequestWorker{store: store, rootDB: rootDB, newTenant: newTenant}
}

func (w *AccountFromRequestWorker) Work(ctx context.Context, job *workflow.Job[AccountFromRequestArgs]) error {
	var tenant models.Tenant
	if err := w.rootDB.WithContext(ctx).First(&tenant, "id = ?", job.Args.TenantID).Error; err != nil {
		return err
	}

	db, err := w.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var request models.Request
	if err := db.WithContext(ctx).First(&request, "id = ?", job.Args.RequestID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	nodeName, ok := extractNodeName(request.Data)
	if !ok {
		return nil
	}

	account := models.Account{Name: nodeName, Status: models.AccountUnknown}
	if err := db.WithContext(ctx).Create(&account).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			slog.Info("account already exists, skipping", "name", nodeName, "request_id", job.Args.RequestID)
			return nil
		}
		slog.Error("failed to create account", "name", nodeName, "request_id", job.Args.RequestID, "error", err)
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := w.store.RecordEvent(ctx, tx, events.AccountCreated{
			AccountID: account.ID,
			Name:      account.Name,
			RequestID: request.ID,
		}); err != nil {
			return err
		}

		return w.store.RecordEvent(ctx, tx, events.RequestAccountLinked{
			RequestID: request.ID,
			ReleaseID: job.Args.ReleaseID,
			AccountID: account.ID,
			Name:      account.Name,
		})
	})
}

func extractNodeName(data json.RawMessage) (string, bool) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false
	}

	candidates := []any{
		payload["node_name"],
		payload["nodeName"],
		payload["node"],
	}
	if nested, ok := payload["node"].(map[string]any); ok {
		candidates = append(candidates, nested["name"])
	}

	for _, candidate := range candidates {
		value, ok := candidate.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value, true
		}
	}
	return "", false
}
