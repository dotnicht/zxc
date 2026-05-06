package service

import (
	"context"
	"errors"

	"github.com/cschleiden/go-workflows/client"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/target"
	"zxc/internal/infra"
	"zxc/internal/jobs"
	"zxc/internal/models"
)

type Target struct {
	target.UnimplementedTargetServiceServer
}

func NewTarget() *Target {
	return &Target{}
}

func (s *Target) Create(ctx context.Context, req *target.CreateRequest) (*target.CreateResponse, error) {
	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	authUserID, err := ctxUserID(ctx)
	if err != nil {
		return nil, err
	}

	tenant, db, err := ctxDeployDB(ctx)
	if err != nil {
		return nil, err
	}

	t := &models.Target{Address: req.Address, User: req.User, Key: req.Key, OwnerID: authUserID}
	if err := db.Create(t).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create target: %v", err)
	}
	if err := s.enqueueProbe(ctx, tenant, t.ID); err != nil {
		cleanupErr := db.Unscoped().Delete(&models.Target{}, "id = ?", t.ID).Error
		return nil, status.Errorf(codes.Internal, "failed to persist target creation: %v", errors.Join(err, cleanupErr))
	}

	return &target.CreateResponse{Target: targetToProto(t)}, nil
}

func (s *Target) Get(ctx context.Context, req *target.GetRequest) (*target.GetResponse, error) {
	id := uuid.UUID(req.Id)

	_, db, err := ctxDeployDB(ctx)
	if err != nil {
		return nil, err
	}

	var t models.Target
	if err := db.First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "target not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get target: %v", err)
	}

	return &target.GetResponse{Target: targetToProto(&t)}, nil
}

func (s *Target) Update(ctx context.Context, req *target.UpdateRequest) (*target.UpdateResponse, error) {
	id := uuid.UUID(req.Id)

	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	tenant, db, err := ctxDeployDB(ctx)
	if err != nil {
		return nil, err
	}

	var previous models.Target
	if err := db.First(&previous, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "target not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load previous target state: %v", err)
	}

	result := db.Model(&models.Target{}).Where("id = ?", id).Updates(&models.Target{Address: req.Address, User: req.User, Key: req.Key})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update target: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "target not found")
	}

	var updated models.Target
	if err := db.First(&updated, "id = ?", id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated target: %v", err)
	}
	if err := s.enqueueProbe(ctx, tenant, updated.ID); err != nil {
		revertErr := db.Model(&models.Target{}).Where("id = ?", id).Updates(map[string]any{
			"address":      previous.Address,
			"user":         previous.User,
			"key":          previous.Key,
			"status":       previous.Status,
			"deploying":    previous.Deploying,
			"deploying_at": previous.DeployingAt,
			"owner_id":     previous.OwnerID,
		}).Error
		return nil, status.Errorf(codes.Internal, "failed to persist target update: %v", errors.Join(err, revertErr))
	}

	return &target.UpdateResponse{Target: targetToProto(&updated)}, nil
}

func (s *Target) Delete(ctx context.Context, req *target.DeleteRequest) (*target.DeleteResponse, error) {
	id := uuid.UUID(req.Id)

	_, db, err := ctxDeployDB(ctx)
	if err != nil {
		return nil, err
	}

	var existing models.Target
	if err := db.First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "target not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load target: %v", err)
	}

	result := db.Where("id = ?", id).Delete(&models.Target{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete target: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "target not found")
	}

	return &target.DeleteResponse{Success: true}, nil
}

func (s *Target) List(ctx context.Context, req *target.ListRequest) (*target.ListResponse, error) {
	page, size := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	_, db, err := ctxDeployDB(ctx)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := db.Model(&models.Target{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count targets: %v", err)
	}

	var targets []*models.Target
	offset := (int(page) - 1) * int(size)
	if err := db.Order("created_at DESC").Limit(int(size)).Offset(offset).Find(&targets).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list targets: %v", err)
	}

	out := make([]*target.Target, len(targets))
	for i, t := range targets {
		out[i] = targetToProto(t)
	}

	return &target.ListResponse{Targets: out, Total: int32(total)}, nil
}

func targetToProto(t *models.Target) *target.Target {
	return &target.Target{
		Id:        t.ID[:],
		Address:   t.Address,
		User:      t.User,
		Key:       t.Key,
		Status:    t.Status,
		OwnerId:   t.OwnerID[:],
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *Target) enqueueProbe(ctx context.Context, t *models.Tenant, targetID uuid.UUID) error {
	wfb, err := infra.WorkflowBackend(t.Jobs)
	if err != nil {
		return err
	}
	wfc := client.New(wfb)
	_, err = wfc.CreateWorkflowInstance(ctx, client.WorkflowInstanceOptions{
		InstanceID: "probe:" + targetID.String(),
	}, jobs.Probe, jobs.ProbeArgs{TenantID: t.ID, TargetID: targetID})
	return err
}
