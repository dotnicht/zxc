package service

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"zxc/internal/middleware"
	"zxc/internal/models"
)

func ctxTenant(ctx context.Context, tenantID uuid.UUID) (*models.Tenant, *models.User, error) {
	user, ok := middleware.UserFromContext(ctx)
	if !ok || user == nil {
		return nil, nil, status.Error(codes.Unauthenticated, "authenticated tenant user is required")
	}

	tenant, ok := middleware.TenantFromContext(ctx, tenantID)
	if !ok || tenant == nil {
		return nil, nil, status.Error(codes.PermissionDenied, "requested tenant does not match authenticated tenant")
	}

	return tenant, user, nil
}

func userID(ctx context.Context) (uuid.UUID, error) {
	user, ok := middleware.UserFromContext(ctx)
	if !ok || user == nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authenticated tenant user is required")
	}
	return user.ID, nil
}
