package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/target"
	"zxc/internal/db"
	"zxc/internal/jobs"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

type Target struct {
	target.UnimplementedTargetServiceServer
	db    *gorm.DB
	cache *db.Cache
	store *workflow.Store
}

func NewTarget(db *gorm.DB, cache *db.Cache, store *workflow.Store) *Target {
	return &Target{db: db, cache: cache, store: store}
}

func (s *Target) Create(ctx context.Context, req *target.CreateRequest) (*target.CreateResponse, error) {
	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	authUserID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireAuthenticatedUser(req.OwnerId, authUserID, "owner_id"); err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	_, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}

	t := &models.Target{Address: req.Address, User: req.User, Key: req.Key, OwnerID: authUserID}
	if err := tenantDB.Create(t).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create target: %v", err)
	}
	if err := s.store.RecordEvent(ctx, nil, workflow.EventInput{
		Kind:          "target_created",
		AggregateType: "target",
		AggregateID:   t.ID.String(),
		TenantID:      &tenantID,
		Payload: map[string]any{
			"target_id": t.ID.String(),
			"address":   t.Address,
			"user":      t.User,
		},
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record target event: %v", err)
	}
	if err := s.enqueueProbe(ctx, tenantID, t.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to schedule target probe: %v", err)
	}

	return &target.CreateResponse{Target: targetToProto(t)}, nil
}

func (s *Target) Get(ctx context.Context, req *target.GetRequest) (*target.GetResponse, error) {
	id, err := parseUUID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	_, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
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

	return &target.GetResponse{Target: targetToProto(&t)}, nil
}

func (s *Target) Update(ctx context.Context, req *target.UpdateRequest) (*target.UpdateResponse, error) {
	id, err := parseUUID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	_, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
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
	if err := s.store.RecordEvent(ctx, nil, workflow.EventInput{
		Kind:          "target_updated",
		AggregateType: "target",
		AggregateID:   updated.ID.String(),
		TenantID:      &tenantID,
		Payload: map[string]any{
			"target_id": updated.ID.String(),
			"address":   updated.Address,
			"user":      updated.User,
		},
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record target event: %v", err)
	}
	if err := s.enqueueProbe(ctx, tenantID, updated.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to schedule target probe: %v", err)
	}

	return &target.UpdateResponse{Target: targetToProto(&updated)}, nil
}

func (s *Target) Delete(ctx context.Context, req *target.DeleteRequest) (*target.DeleteResponse, error) {
	id, err := parseUUID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	_, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}

	result := tenantDB.Where("id = ?", id).Delete(&models.Target{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete target: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "target not found")
	}
	if err := s.store.RecordEvent(ctx, nil, workflow.EventInput{
		Kind:          "target_deleted",
		AggregateType: "target",
		AggregateID:   id.String(),
		TenantID:      &tenantID,
		Payload: map[string]any{
			"target_id": id.String(),
		},
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record target event: %v", err)
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

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	_, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
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

	protoTargets := make([]*target.Target, len(targets))
	for i, t := range targets {
		protoTargets[i] = targetToProto(t)
	}

	return &target.ListResponse{Targets: protoTargets, Total: int32(total)}, nil
}

func (s *Target) Search(ctx context.Context, req *target.SearchRequest) (*target.SearchResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "search query is required")
	}

	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	_, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}

	pattern := "%" + req.Query + "%"
	var total int64
	if err := tenantDB.Model(&models.Target{}).Where("address ILIKE ?", pattern).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count targets: %v", err)
	}

	var targets []*models.Target
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.Where("address ILIKE ?", pattern).Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&targets).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search targets: %v", err)
	}

	protoTargets := make([]*target.Target, len(targets))
	for i, t := range targets {
		protoTargets[i] = targetToProto(t)
	}

	return &target.SearchResponse{Targets: protoTargets, Total: int32(total)}, nil
}

func targetToProto(t *models.Target) *target.Target {
	return &target.Target{
		Id:        t.ID.String(),
		Address:   t.Address,
		User:      t.User,
		Key:       t.Key,
		Status:    t.Status,
		OwnerId:   t.OwnerID.String(),
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *Target) enqueueProbe(ctx context.Context, tenantID uuid.UUID, targetID uuid.UUID) error {
	return s.store.EnqueueCommand(ctx, nil, workflow.CommandInput{
		Kind:          "probe_target",
		AggregateType: "target",
		AggregateID:   targetID.String(),
		TenantID:      &tenantID,
		Payload: jobs.TargetProbeArgs{
			TenantID: tenantID,
			TargetID: targetID,
		},
		DedupeKey: "target-probe:" + targetID.String(),
	})
}
