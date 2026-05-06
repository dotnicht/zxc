package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/user"
	"zxc/internal/models"
)

type User struct {
	user.UnimplementedUserServiceServer
}

func NewUser() *User {
	return &User{}
}

func (s *User) Create(ctx context.Context, req *user.CreateRequest) (*user.CreateResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	_, db, err := usersDB(ctx)
	if err != nil {
		return nil, err
	}

	u := &models.User{Name: req.Name}
	if err := db.Create(u).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create user: %v", err))
	}

	return &user.CreateResponse{User: s.proto(u)}, nil
}

func (s *User) Get(ctx context.Context, req *user.GetRequest) (*user.GetResponse, error) {
	uid := uuid.UUID(req.Id)

	_, db, err := usersDB(ctx)
	if err != nil {
		return nil, err
	}

	var u models.User
	if err := db.First(&u, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get user: %v", err))
	}

	return &user.GetResponse{User: s.proto(&u)}, nil
}

func (s *User) Update(ctx context.Context, req *user.UpdateRequest) (*user.UpdateResponse, error) {
	uid := uuid.UUID(req.Id)

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	_, db, err := usersDB(ctx)
	if err != nil {
		return nil, err
	}

	var current models.User
	if err := db.First(&current, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to load user: %v", err))
	}

	result := db.Model(&models.User{}).Where("id = ?", uid).Updates(&models.User{Name: req.Name})
	if result.Error != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update user: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	var updated models.User
	if err := db.First(&updated, "id = ?", uid).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to fetch updated user: %v", err))
	}

	return &user.UpdateResponse{User: s.proto(&updated)}, nil
}

func (s *User) Delete(ctx context.Context, req *user.DeleteRequest) (*user.DeleteResponse, error) {
	uid := uuid.UUID(req.Id)

	_, db, err := usersDB(ctx)
	if err != nil {
		return nil, err
	}

	result := db.Where("id = ?", uid).Delete(&models.User{})
	if result.Error != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete user: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &user.DeleteResponse{Success: true}, nil
}

func (s *User) List(ctx context.Context, req *user.ListRequest) (*user.ListResponse, error) {
	page, size := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	_, db, err := usersDB(ctx)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to count users: %v", err))
	}

	var users []*models.User
	offset := (int(page) - 1) * int(size)
	if err := db.Order("created_at DESC").Limit(int(size)).Offset(offset).Find(&users).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list users: %v", err))
	}

	out := make([]*user.User, len(users))
	for i, u := range users {
		out[i] = s.proto(u)
	}

	return &user.ListResponse{Users: out, Total: int32(total)}, nil
}

func (s *User) proto(m *models.User) *user.User {
	return &user.User{
		Id:        m.ID[:],
		Name:      m.Name,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
