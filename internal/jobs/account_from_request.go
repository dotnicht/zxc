package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	var (
		request        models.Request
		account        models.Account
		accountCreated bool
	)

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&request, "id = ?", job.Args.RequestID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if request.AccountID != nil {
			return nil
		}

		nodeName, ok := extractNodeName(request.Data)
		if !ok {
			return nil
		}

		if err := tx.Where("name = ?", nodeName).First(&account).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			account = models.Account{Name: nodeName}
			if err := tx.Create(&account).Error; err != nil {
				if err := tx.Where("name = ?", nodeName).First(&account).Error; err != nil {
					return err
				}
			} else {
				accountCreated = true
			}
		}

		result := tx.Model(&models.Request{}).Where("id = ? AND account_id IS NULL", request.ID).Update("account_id", account.ID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		request.AccountID = &account.ID
		return nil
	})
	if err != nil {
		return fmt.Errorf("sync request account: %w", err)
	}
	if request.ID == uuid.Nil || request.AccountID == nil {
		return nil
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if accountCreated {
			if err := w.store.RecordEvent(ctx, tx, workflow.EventInput{
				Kind:          "account_created",
				AggregateType: "account",
				AggregateID:   account.ID,
				Payload: map[string]any{
					"account_id": account.ID.String(),
					"name":       account.Name,
					"request_id": request.ID.String(),
				},
			}); err != nil {
				return err
			}
		}

		return w.store.RecordEvent(ctx, tx, workflow.EventInput{
			Kind:          "request_account_linked",
			AggregateType: "request",
			AggregateID:   request.ID,
			Payload: map[string]any{
				"request_id": request.ID.String(),
				"release_id": job.Args.ReleaseID.String(),
				"account_id": account.ID.String(),
				"name":       account.Name,
			},
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
