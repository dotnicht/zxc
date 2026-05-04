package service

import (
	"context"
	"errors"
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"zxc/internal/infra"
	"zxc/internal/models"
)

var validator protovalidate.Validator

func init() {
	var err error
	validator, err = protovalidate.New()
	if err != nil {
		panic("failed to initialize protovalidate: " + err.Error())
	}
}

func ValidateInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if msg, ok := req.(proto.Message); ok {
			if err := validator.Validate(msg); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "%v", err)
			}
		}
		return handler(ctx, req)
	}
}

type userKey struct{}
type tenantKey struct{}
type usersDBKey struct{}
type deployDBKey struct{}
type accountDBKey struct{}

func UserInterceptor(rootDB *gorm.DB, rootUserID uuid.UUID) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "x-user-id metadata is required")
		}

		userID, err := metaUUID(md, "x-user-id")
		if err != nil {
			return nil, err
		}

		tenantIDs := md.Get("x-tenant-id")
		if len(tenantIDs) == 0 {
			if userID != rootUserID {
				return nil, status.Error(codes.PermissionDenied, "x-tenant-id metadata is required for non-root requests")
			}
			var root models.User
			if err := rootDB.WithContext(ctx).First(&root, "id = ?", userID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, status.Error(codes.NotFound, "user not found")
				}
				return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
			}
			return handler(context.WithValue(ctx, userKey{}, &root), req)
		}

		tenantID, err := metaUUID(md, "x-tenant-id")
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

		usersDB, err := infra.NewConnection(tenant.UsersDatabase)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to connect to users database: %v", err)
		}

		deployDB, err := infra.NewConnection(tenant.DeployDatabase)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to connect to deploy database: %v", err)
		}

		accountDB, err := infra.NewConnection(tenant.AccountDatabase)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to connect to account database: %v", err)
		}

		var user models.User
		if err := usersDB.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, status.Error(codes.NotFound, "user not found")
			}
			return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
		}

		ctx = context.WithValue(ctx, tenantKey{}, &tenant)
		ctx = context.WithValue(ctx, userKey{}, &user)
		ctx = context.WithValue(ctx, usersDBKey{}, usersDB)
		ctx = context.WithValue(ctx, deployDBKey{}, deployDB)
		ctx = context.WithValue(ctx, accountDBKey{}, accountDB)
		return handler(ctx, req)
	}
}

func ctxTenant(ctx context.Context) (*models.Tenant, error) {
	tenant, ok := ctx.Value(tenantKey{}).(*models.Tenant)
	if !ok || tenant == nil {
		return nil, status.Error(codes.Unauthenticated, "authenticated tenant user is required")
	}
	return tenant, nil
}

func ctxUsersDB(ctx context.Context) (*models.Tenant, *gorm.DB, error) {
	tenant, err := ctxTenant(ctx)
	if err != nil {
		return nil, nil, err
	}
	db, ok := ctx.Value(usersDBKey{}).(*gorm.DB)
	if !ok || db == nil {
		return nil, nil, status.Error(codes.Internal, "users database connection unavailable")
	}
	return tenant, db, nil
}

func ctxDeployDB(ctx context.Context) (*models.Tenant, *gorm.DB, error) {
	tenant, err := ctxTenant(ctx)
	if err != nil {
		return nil, nil, err
	}
	db, ok := ctx.Value(deployDBKey{}).(*gorm.DB)
	if !ok || db == nil {
		return nil, nil, status.Error(codes.Internal, "deploy database connection unavailable")
	}
	return tenant, db, nil
}

func ctxAccountDB(ctx context.Context) (*models.Tenant, *gorm.DB, error) {
	tenant, err := ctxTenant(ctx)
	if err != nil {
		return nil, nil, err
	}
	db, ok := ctx.Value(accountDBKey{}).(*gorm.DB)
	if !ok || db == nil {
		return nil, nil, status.Error(codes.Internal, "account database connection unavailable")
	}
	return tenant, db, nil
}

func metaUUID(md metadata.MD, key string) (uuid.UUID, error) {
	vals := md.Get(key)
	if len(vals) == 0 {
		return uuid.Nil, status.Errorf(codes.Unauthenticated, "%s metadata is required", key)
	}
	id, err := uuid.Parse(vals[0])
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, fmt.Sprintf("%s must be a valid UUID", key))
	}
	return id, nil
}

func ctxUserID(ctx context.Context) (uuid.UUID, error) {
	user, ok := ctx.Value(userKey{}).(*models.User)
	if !ok || user == nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authenticated user is required")
	}
	return user.ID, nil
}
