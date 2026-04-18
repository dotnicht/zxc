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

func User(cache *db.Cache, rootDB *gorm.DB) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		tenantIDs := md.Get("x-tenant-id")
		userIDs := md.Get("x-user-id")
		if len(tenantIDs) == 0 && len(userIDs) == 0 {
			return handler(ctx, req)
		}
		if len(tenantIDs) == 0 || len(userIDs) == 0 {
			return nil, status.Error(codes.InvalidArgument, "x-tenant-id and x-user-id must both be provided")
		}

		tenantID, err := uuid.Parse(tenantIDs[0])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "x-tenant-id must be a valid UUID")
		}

		userID, err := uuid.Parse(userIDs[0])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "x-user-id must be a valid UUID")
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
