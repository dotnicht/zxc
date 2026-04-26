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
	"zxc/internal/authz"
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

func assertOwner(raw string, authUserID uuid.UUID, field string) error {
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid %s: must be a valid UUID", field)
	}
	if id != authUserID {
		return status.Errorf(codes.PermissionDenied, "%s must match authenticated user", field)
	}
	return nil
}

func resolve(ctx context.Context, cache *infra.Cache, tenantID uuid.UUID) (*models.Tenant, *gorm.DB, error) {
	tenant, _, err := ctxTenant(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}
	tenantDB, err := cache.Get(tenant.Database)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to connect to tenant database: %v", err)
	}
	return tenant, tenantDB, nil
}

type userKey struct{}
type tenantKey struct{}

func UserInterceptor(cache *infra.Cache, rootDB *gorm.DB, rootUserID uuid.UUID) grpc.UnaryServerInterceptor {
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

		tenantDB, err := cache.Get(tenant.Database)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to connect to tenant database: %v", err)
		}

		var user models.User
		if err := tenantDB.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, status.Error(codes.NotFound, "user not found")
			}
			return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
		}

		ctx = context.WithValue(ctx, tenantKey{}, &tenant)
		ctx = context.WithValue(ctx, userKey{}, &user)
		return handler(ctx, req)
	}
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

func ctxTenant(ctx context.Context, tenantID uuid.UUID) (*models.Tenant, *models.User, error) {
	user, ok := ctx.Value(userKey{}).(*models.User)
	if !ok || user == nil {
		return nil, nil, status.Error(codes.Unauthenticated, "authenticated tenant user is required")
	}
	tenant, ok := ctx.Value(tenantKey{}).(*models.Tenant)
	if !ok || tenant == nil || tenant.ID != tenantID {
		return nil, nil, status.Error(codes.PermissionDenied, "requested tenant does not match authenticated tenant")
	}
	return tenant, user, nil
}

func ctxUserID(ctx context.Context) (uuid.UUID, error) {
	user, ok := ctx.Value(userKey{}).(*models.User)
	if !ok || user == nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authenticated user is required")
	}
	return user.ID, nil
}

func authorize(ctx context.Context, action string, tenant *models.Tenant, resource authz.Resource, related authz.Related) (authz.Decision, error) {
	engine, err := authz.Default()
	if err != nil {
		return authz.Decision{}, status.Errorf(codes.Internal, "failed to load authorization policy: %v", err)
	}

	user, ok := ctx.Value(userKey{}).(*models.User)
	if !ok || user == nil {
		return authz.Decision{}, status.Error(codes.Unauthenticated, "authenticated user is required")
	}

	input := authz.Input{
		Subject:  authz.Subject{ID: user.ID, IsRoot: tenant == nil},
		Action:   action,
		Resource: resource,
		Related:  related,
	}
	if tenant != nil {
		input.Tenant = authz.Tenant{ID: tenant.ID, OwnerID: tenant.OwnerID}
	}

	decision, err := engine.Evaluate(ctx, input)
	if err != nil {
		return authz.Decision{}, status.Errorf(codes.Internal, "authorization policy evaluation failed: %v", err)
	}
	if !decision.Allow {
		if decision.Reason == "" {
			decision.Reason = "policy denied"
		}
		return decision, status.Error(codes.PermissionDenied, decision.Reason)
	}
	return decision, nil
}
