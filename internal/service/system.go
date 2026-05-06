package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/system"
	"zxc/internal/models"
)

type System struct {
	system.UnimplementedSystemServiceServer
}

func NewSystem() *System {
	return &System{}
}

func (s *System) Create(ctx context.Context, req *system.CreateRequest) (*system.CreateResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	_, db, err := ctxUsersDB(ctx)
	if err != nil {
		return nil, err
	}

	m := &models.System{Name: req.Name}
	if err := db.Create(m).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create system: %v", err))
	}

	return &system.CreateResponse{System: systemToProto(m)}, nil
}

func (s *System) Get(ctx context.Context, req *system.GetRequest) (*system.GetResponse, error) {
	id := uuid.UUID(req.Id)

	_, db, err := ctxUsersDB(ctx)
	if err != nil {
		return nil, err
	}

	var m models.System
	if err := db.First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "system not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get system: %v", err))
	}

	return &system.GetResponse{System: systemToProto(&m)}, nil
}

func (s *System) Update(ctx context.Context, req *system.UpdateRequest) (*system.UpdateResponse, error) {
	id := uuid.UUID(req.Id)

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	_, db, err := ctxUsersDB(ctx)
	if err != nil {
		return nil, err
	}

	var current models.System
	if err := db.First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "system not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to load system: %v", err))
	}

	result := db.Model(&models.System{}).Where("id = ?", id).Updates(&models.System{Name: req.Name})
	if result.Error != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to update system: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "system not found")
	}

	var updated models.System
	if err := db.First(&updated, "id = ?", id).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to fetch updated system: %v", err))
	}

	return &system.UpdateResponse{System: systemToProto(&updated)}, nil
}

func (s *System) Delete(ctx context.Context, req *system.DeleteRequest) (*system.DeleteResponse, error) {
	id := uuid.UUID(req.Id)

	_, db, err := ctxUsersDB(ctx)
	if err != nil {
		return nil, err
	}

	result := db.Where("id = ?", id).Delete(&models.System{})
	if result.Error != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to delete system: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "system not found")
	}

	return &system.DeleteResponse{Success: true}, nil
}

func (s *System) List(ctx context.Context, req *system.ListRequest) (*system.ListResponse, error) {
	page, size := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	_, db, err := ctxUsersDB(ctx)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := db.Model(&models.System{}).Count(&total).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to count systems: %v", err))
	}

	var systems []*models.System
	offset := (int(page) - 1) * int(size)
	if err := db.Order("created_at DESC").Limit(int(size)).Offset(offset).Find(&systems).Error; err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list systems: %v", err))
	}

	out := make([]*system.System, len(systems))
	for i, m := range systems {
		out[i] = systemToProto(m)
	}

	return &system.ListResponse{Systems: out, Total: int32(total)}, nil
}

func systemToProto(m *models.System) *system.System {
	return &system.System{
		Id:        m.ID[:],
		Name:      m.Name,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
