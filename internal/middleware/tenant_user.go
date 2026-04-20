package middleware

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/internal/db"
	"zxc/internal/models"
)

func User(cache *db.Cache, rootDB *gorm.DB, rootUserID uuid.UUID) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok || len(md) == 0 {
			return nil, status.Error(codes.Unauthenticated, "x-user-id metadata is required")
		}

		tenantIDs := md.Get("x-tenant-id")
		userIDs := md.Get("x-user-id")
		if len(userIDs) == 0 {
			return nil, status.Error(codes.Unauthenticated, "x-user-id metadata is required")
		}

		userID, err := metadataUUID(md, "x-user-id")
		if err != nil {
			return nil, err
		}

		if len(tenantIDs) == 0 {
			if userID != rootUserID {
				return nil, status.Error(codes.PermissionDenied, "x-tenant-id metadata is required for non-root requests")
			}

			var rootUser models.User
			if err := rootDB.WithContext(ctx).First(&rootUser, "id = ?", userID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, status.Error(codes.NotFound, "user not found")
				}
				return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
			}
			return handler(contextWithUser(ctx, &rootUser), req)
		}

		tenantID, err := metadataUUID(md, "x-tenant-id")
		if err != nil {
			return nil, err
		}

		var tenant models.Tenant
		if err := rootDB.WithContext(ctx).First(&tenant, "id = ?", tenantID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, status.Error(codes.NotFound, "tenant not found")
			}
			return nil, status.Errorf(codes.Internal, "failed to get tenant: %v", err)
		}

		tenantDB, err := cache.Get(tenant.Database)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to connect to tenant database: %v", err)
		}

		var user models.User
		if err := tenantDB.First(&user, "id = ?", userID).Error; err != nil {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		ctx = contextWithTenant(ctx, &tenant)
		return handler(contextWithUser(ctx, &user), req)
	}
}
