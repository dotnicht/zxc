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
	"zxc/internal/authz"
	"zxc/internal/jobs"
	"zxc/internal/models"
)

type Target struct {
	target.UnimplementedTargetServiceServer
	wfclient *client.Client
}

func NewTarget(wfclient *client.Client) *Target {
	return &Target{wfclient: wfclient}
}

func (s *Target) Create(ctx context.Context, req *target.CreateRequest) (*target.CreateResponse, error) {
	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	authUserID, err := ctxUserID(ctx)
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := ctxTenantAndDB(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "target.create", tenant, authz.Resource{Type: "target"}, authz.Related{}); err != nil {
		return nil, err
	}

	t := &models.Target{Address: req.Address, User: req.User, Key: req.Key, OwnerID: authUserID}
	if err := tenantDB.Create(t).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create target: %v", err)
	}
	if err := s.enqueueProbe(ctx, tenant.ID, t.ID); err != nil {
		cleanupErr := tenantDB.Unscoped().Delete(&models.Target{}, "id = ?", t.ID).Error
		return nil, status.Errorf(codes.Internal, "failed to persist target creation: %v", errors.Join(err, cleanupErr))
	}

	decision, err := authorize(ctx, "target.get", tenant, authz.Resource{
		Type:    "target",
		OwnerID: t.OwnerID,
	}, authz.Related{})
	if err != nil {
		return nil, err
	}
	return &target.CreateResponse{Target: s.targetToProto(t, decision.RevealSecret)}, nil
}

func (s *Target) Get(ctx context.Context, req *target.GetRequest) (*target.GetResponse, error) {
	id := uuid.MustParse(req.Id)

	tenant, tenantDB, err := ctxTenantAndDB(ctx)
	if err != nil {
		return nil, err
	}

	var t models.Target
	if err := tenantDB.First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "target not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get target: %v", err)
	}
	decision, err := authorize(ctx, "target.get", tenant, authz.Resource{
		Type:    "target",
		OwnerID: t.OwnerID,
	}, authz.Related{})
	if err != nil {
		return nil, err
	}
	return &target.GetResponse{Target: s.targetToProto(&t, decision.RevealSecret)}, nil
}

func (s *Target) Update(ctx context.Context, req *target.UpdateRequest) (*target.UpdateResponse, error) {
	id := uuid.MustParse(req.Id)

	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	tenant, tenantDB, err := ctxTenantAndDB(ctx)
	if err != nil {
		return nil, err
	}

	var previous models.Target
	if err := tenantDB.First(&previous, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "target not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load previous target state: %v", err)
	}
	if _, err := authorize(ctx, "target.update", tenant, authz.Resource{
		Type:    "target",
		OwnerID: previous.OwnerID,
	}, authz.Related{}); err != nil {
		return nil, err
	}

	result := tenantDB.Model(&models.Target{}).Where("id = ?", id).Updates(&models.Target{Address: req.Address, User: req.User, Key: req.Key})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update target: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "target not found")
	}

	var updated models.Target
	if err := tenantDB.First(&updated, "id = ?", id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated target: %v", err)
	}
	if err := s.enqueueProbe(ctx, tenant.ID, updated.ID); err != nil {
		revertErr := tenantDB.Model(&models.Target{}).Where("id = ?", id).Updates(map[string]any{
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

	decision, err := authorize(ctx, "target.get", tenant, authz.Resource{
		Type:    "target",
		OwnerID: updated.OwnerID,
	}, authz.Related{})
	if err != nil {
		return nil, err
	}
	return &target.UpdateResponse{Target: s.targetToProto(&updated, decision.RevealSecret)}, nil
}

func (s *Target) Delete(ctx context.Context, req *target.DeleteRequest) (*target.DeleteResponse, error) {
	id := uuid.MustParse(req.Id)

	tenant, tenantDB, err := ctxTenantAndDB(ctx)
	if err != nil {
		return nil, err
	}

	var existing models.Target
	if err := tenantDB.First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "target not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load target: %v", err)
	}
	if _, err := authorize(ctx, "target.delete", tenant, authz.Resource{
		Type:    "target",
		OwnerID: existing.OwnerID,
	}, authz.Related{}); err != nil {
		return nil, err
	}

	result := tenantDB.Where("id = ?", id).Delete(&models.Target{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete target: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "target not found")
	}

	return &target.DeleteResponse{Success: true}, nil
}

func (s *Target) List(ctx context.Context, req *target.ListRequest) (*target.ListResponse, error) {
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	tenant, tenantDB, err := ctxTenantAndDB(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "target.list", tenant, authz.Resource{Type: "target"}, authz.Related{}); err != nil {
		return nil, err
	}

	var total int64
	if err := tenantDB.Model(&models.Target{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count targets: %v", err)
	}

	var targets []*models.Target
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&targets).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list targets: %v", err)
	}

	out := make([]*target.Target, len(targets))
	for i, t := range targets {
		d, err := authorize(ctx, "target.get", tenant, authz.Resource{
			Type:    "target",
			OwnerID: t.OwnerID,
		}, authz.Related{})
		if err != nil {
			return nil, err
		}
		out[i] = s.targetToProto(t, d.RevealSecret)
	}

	return &target.ListResponse{Targets: out, Total: int32(total)}, nil
}

func (s *Target) targetToProto(t *models.Target, reveal bool) *target.Target {
	p := &target.Target{
		Id:        t.ID.String(),
		Address:   t.Address,
		User:      t.User,
		Key:       t.Key,
		Status:    t.Status,
		OwnerId:   t.OwnerID.String(),
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if !reveal {
		p.Key = ""
	}
	return p
}

func (s *Target) enqueueProbe(ctx context.Context, tenantID uuid.UUID, targetID uuid.UUID) error {
	_, err := s.wfclient.CreateWorkflowInstance(ctx, client.WorkflowInstanceOptions{
		InstanceID: "probe:" + targetID.String(),
	}, jobs.Probe, jobs.ProbeArgs{TenantID: tenantID, TargetID: targetID})
	return err
}
