package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/cschleiden/go-workflows/workflow"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
)

type AccountArgs struct {
	TenantID  uuid.UUID
	RequestID uuid.UUID
	ReleaseID uuid.UUID
}

func Account(ctx workflow.Context, args AccountArgs) error {
	_, err := workflow.ExecuteActivity[any](ctx, workflow.DefaultActivityOptions, AccountActivity, args).Get(ctx)
	return err
}

type accountDeps struct {
	rootDB     *gorm.DB
	newDeploy  func(string) (*gorm.DB, error)
	newAccount func(string) (*gorm.DB, error)
}

var accountDep *accountDeps

func RegisterAccount(rootDB *gorm.DB, connect func(string) (*gorm.DB, error), connectAccount func(string) (*gorm.DB, error)) {
	accountDep = &accountDeps{rootDB: rootDB, newDeploy: connect, newAccount: connectAccount}
}

func AccountActivity(ctx context.Context, args AccountArgs) error {
	var tenant models.Tenant
	if err := accountDep.rootDB.WithContext(ctx).First(&tenant, "id = ?", args.TenantID).Error; err != nil {
		return err
	}

	deployDB, err := accountDep.newDeploy(tenant.Deploy)
	if err != nil {
		return err
	}

	accountDB, err := accountDep.newAccount(tenant.Account)
	if err != nil {
		return err
	}

	var request models.Request
	if err := deployDB.WithContext(ctx).First(&request, "id = ?", args.RequestID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	nodeName, ok := extract(request.Data)
	if !ok {
		return nil
	}

	var release models.Release
	if err := deployDB.WithContext(ctx).First(&release, "id = ?", args.ReleaseID).Error; err != nil {
		return err
	}
	var payload models.Payload
	if err := deployDB.WithContext(ctx).First(&payload, "id = ?", release.PayloadID).Error; err != nil {
		return err
	}

	profile := &models.Profile{Name: nodeName, Status: models.ProfileUnknown, SystemID: payload.SystemID}
	if err := accountDB.WithContext(ctx).Create(profile).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			slog.Info("profile already exists, skipping", "name", nodeName, "request_id", args.RequestID)
			return nil
		}
		return err
	}

	return nil
}

func extract(data json.RawMessage) (string, bool) {
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
