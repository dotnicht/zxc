package service

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/user"
	"zxc/internal/authz"
	"zxc/internal/db"
	"zxc/internal/models"
)

type User struct {
	user.UnimplementedUserServiceServer
	db    *gorm.DB
	cache *db.Cache
}

func NewUser(db *gorm.DB, cache *db.Cache) *User {
	return &User{db: db, cache: cache}
}

func (s *User) Create(ctx context.Context, req *user.CreateRequest) (*user.CreateResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	tenantID, err := parseID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "user.create", tenant, authz.Resource{Type: "user"}, authz.Related{}); err != nil {
		return nil, err
	}

	u := &models.User{Name: req.Name}
	if err := tenantDB.Create(u).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create user: %v", err))
	}

	return &user.CreateResponse{User: userToProto(u)}, nil
}

func (s *User) Get(ctx context.Context, req *user.GetRequest) (*user.GetResponse, error) {
	uid, err := parseID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	tenantID, err := parseID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}

	var u models.User
	if err := tenantDB.First(&u, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get user: %v", err))
	}
	if _, err := authorize(ctx, "user.get", tenant, authz.Resource{
		Type:    "user",
		OwnerID: u.ID,
	}, authz.Related{}); err != nil {
		return nil, err
	}

	return &user.GetResponse{User: userToProto(&u)}, nil
}

func (s *User) Update(ctx context.Context, req *user.UpdateRequest) (*user.UpdateResponse, error) {
	uid, err := parseID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	tenantID, err := parseID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}

	var current models.User
	if err := tenantDB.First(&current, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to load user: %v", err))
	}
	if _, err := authorize(ctx, "user.update", tenant, authz.Resource{
		Type:    "user",
		OwnerID: current.ID,
	}, authz.Related{}); err != nil {
		return nil, err
	}

	result := tenantDB.Model(&models.User{}).Where("id = ?", uid).Updates(&models.User{Name: req.Name})
	if result.Error != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update user: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	var updated models.User
	if err := tenantDB.First(&updated, "id = ?", uid).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to fetch updated user: %v", err))
	}

	return &user.UpdateResponse{User: userToProto(&updated)}, nil
}

func (s *User) Delete(ctx context.Context, req *user.DeleteRequest) (*user.DeleteResponse, error) {
	uid, err := parseID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	tenantID, err := parseID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "user.delete", tenant, authz.Resource{
		Type:    "user",
		OwnerID: uid,
	}, authz.Related{}); err != nil {
		return nil, err
	}

	result := tenantDB.Where("id = ?", uid).Delete(&models.User{})
	if result.Error != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete user: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &user.DeleteResponse{Success: true}, nil
}

func (s *User) List(ctx context.Context, req *user.ListRequest) (*user.ListResponse, error) {
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	tenantID, err := parseID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "user.list", tenant, authz.Resource{Type: "user"}, authz.Related{}); err != nil {
		return nil, err
	}

	var total int64
	if err := tenantDB.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to count users: %v", err))
	}

	var users []*models.User
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&users).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list users: %v", err))
	}

	out := make([]*user.User, len(users))
	for i, u := range users {
		out[i] = userToProto(u)
	}

	return &user.ListResponse{Users: out, Total: int32(total)}, nil
}

func userToProto(m *models.User) *user.User {
	return &user.User{
		Id:        m.ID.String(),
		Name:      m.Name,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
