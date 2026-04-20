package service

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"zxc/internal/middleware"
	"zxc/internal/models"
)

func authenticatedTenant(ctx context.Context, tenantIDStr string) (uuid.UUID, *models.Tenant, *models.User, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, nil, nil, status.Error(codes.InvalidArgument, "invalid tenant_id: must be a valid UUID")
	}

	user, ok := middleware.UserFromContext(ctx)
	if !ok || user == nil {
		return uuid.Nil, nil, nil, status.Error(codes.Unauthenticated, "authenticated tenant user is required")
	}

	tenant, ok := middleware.TenantFromContext(ctx, tenantID)
	if !ok || tenant == nil {
		return uuid.Nil, nil, nil, status.Error(codes.PermissionDenied, "requested tenant does not match authenticated tenant")
	}

	return tenantID, tenant, user, nil
}

func authenticatedUserID(ctx context.Context) (uuid.UUID, error) {
	user, ok := middleware.UserFromContext(ctx)
	if !ok || user == nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authenticated tenant user is required")
	}
	return user.ID, nil
}
