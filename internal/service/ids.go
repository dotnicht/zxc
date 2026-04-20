package service

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/internal/db"
	"zxc/internal/models"
)

func parseUUID(raw string, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid %s: must be a valid UUID", field)
	}
	return id, nil
}

func requireAuthenticatedUser(raw string, authUserID uuid.UUID, field string) error {
	if raw == "" {
		return nil
	}

	id, err := parseUUID(raw, field)
	if err != nil {
		return err
	}
	if id != authUserID {
		return status.Errorf(codes.PermissionDenied, "%s must match authenticated user", field)
	}
	return nil
}

func resolveTenantDB(ctx context.Context, cache *db.Cache, tenantID uuid.UUID) (*models.Tenant, *gorm.DB, error) {
	tenant, _, err := authenticatedTenant(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}

	tenantDB, err := cache.Get(tenant.Database)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to connect to tenant database: %v", err)
	}
	return tenant, tenantDB, nil
}
