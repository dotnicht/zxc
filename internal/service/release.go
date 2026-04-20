package service

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/release"
	"zxc/internal/db"
	"zxc/internal/jobs"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

var validReleaseStatuses = map[string]bool{
	models.ReleaseUnknown:  true,
	models.ReleaseWait:     true,
	models.ReleaseDeployed: true,
	models.ReleaseDead:     true,
	models.ReleaseAlive:    true,
}

type Release struct {
	release.UnimplementedReleaseServiceServer
	db    *gorm.DB
	cache *db.Cache
	store *workflow.Store
}

func NewRelease(db *gorm.DB, cache *db.Cache, store *workflow.Store) *Release {
	return &Release{db: db, cache: cache, store: store}
}

func (s *Release) Create(ctx context.Context, req *release.CreateRequest) (*release.CreateResponse, error) {
	authUserID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireAuthenticatedUser(req.OwnerId, authUserID, "owner_id"); err != nil {
		return nil, err
	}

	targetID, err := parseUUID(req.TargetId, "target_id")
	if err != nil {
		return nil, err
	}

	payloadID, err := parseUUID(req.PayloadId, "payload_id")
	if err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	releaseStatus := req.Status
	if releaseStatus == "" {
		releaseStatus = models.ReleaseUnknown
	} else if !validReleaseStatuses[releaseStatus] {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status %q: must be one of unknown, deployed, dead, alive", releaseStatus)
	}

	tenant, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}

	var t models.Target
	if err := tenantDB.First(&t, "id = ?", targetID).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "target not found")
	}

	var p models.Payload
	if err := tenantDB.First(&p, "id = ?", payloadID).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "payload not found")
	}

	rel := &models.Release{
		Status:      releaseStatus,
		OwnerID:     authUserID,
		TargetID:    &targetID,
		PayloadID:   &payloadID,
		ChangedByID: authUserID,
	}
	if err := tenantDB.Create(rel).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create release: %v", err)
	}

	if err := s.store.RootTransaction(ctx, func(tx *gorm.DB) error {
		route := &models.Route{ID: rel.ID, TenantID: tenant.ID}
		if err := tx.Create(route).Error; err != nil {
			return err
		}
		return s.store.RecordEvent(ctx, tx, workflow.EventInput{
			Kind:          "release_created",
			AggregateType: "release",
			AggregateID:   rel.ID.String(),
			TenantID:      &tenant.ID,
			Payload: map[string]any{
				"release_id":    rel.ID.String(),
				"owner_id":      rel.OwnerID.String(),
				"target_id":     targetID.String(),
				"payload_id":    payloadID.String(),
				"changed_by_id": rel.ChangedByID.String(),
				"status":        rel.Status,
			},
		})
	}); err != nil {
		cleanupErr := tenantDB.Unscoped().Delete(&models.Release{}, "id = ?", rel.ID).Error
		return nil, status.Errorf(codes.Internal, "failed to persist release root state: %v", errors.Join(err, cleanupErr))
	}

	return &release.CreateResponse{Release: releaseToProto(rel)}, nil
}

func (s *Release) Get(ctx context.Context, req *release.GetRequest) (*release.GetResponse, error) {
	releaseID, err := parseUUID(req.Id, "id")
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

	var rel models.Release
	if err := tenantDB.First(&rel, "id = ?", releaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "release not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get release: %v", err)
	}

	return &release.GetResponse{Release: releaseToProto(&rel)}, nil
}

func (s *Release) Deploy(ctx context.Context, req *release.DeployRequest) (*release.DeployResponse, error) {
	releaseID, err := parseUUID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	authUserID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireAuthenticatedUser(req.UserId, authUserID, "user_id"); err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}

	result := tenantDB.Model(&models.Release{}).
		Where("id = ? AND status = ?", releaseID, models.ReleaseUnknown).
		Updates(map[string]any{"status": models.ReleaseWait, "changed_by_id": authUserID})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update release: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "release not found")
	}
	if err := s.store.RootTransaction(ctx, func(tx *gorm.DB) error {
		if err := s.store.RecordEvent(ctx, tx, workflow.EventInput{
			Kind:          "release_deploy_requested",
			AggregateType: "release",
			AggregateID:   releaseID.String(),
			TenantID:      &tenant.ID,
			Payload: map[string]any{
				"release_id": releaseID.String(),
				"user_id":    authUserID.String(),
			},
		}); err != nil {
			return err
		}
		return s.store.EnqueueCommand(ctx, tx, workflow.CommandInput{
			Kind:          "deploy_release",
			AggregateType: "release",
			AggregateID:   releaseID.String(),
			TenantID:      &tenant.ID,
			Payload: jobs.DeployReleaseArgs{
				TenantID:    tenant.ID,
				ReleaseID:   releaseID,
				ChangedByID: authUserID,
			},
			DedupeKey: "release-deploy:" + releaseID.String(),
		})
	}); err != nil {
		revertErr := tenantDB.Model(&models.Release{}).
			Where("id = ? AND status = ?", releaseID, models.ReleaseWait).
			Updates(map[string]any{"status": models.ReleaseUnknown, "changed_by_id": authUserID}).Error
		return nil, status.Errorf(codes.Internal, "failed to persist deploy request: %v", errors.Join(err, revertErr))
	}

	var rel models.Release
	if err := tenantDB.First(&rel, "id = ?", releaseID).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get release: %v", err)
	}

	return &release.DeployResponse{Release: releaseToProto(&rel)}, nil
}

func (s *Release) List(ctx context.Context, req *release.ListRequest) (*release.ListResponse, error) {
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
	if err := tenantDB.Model(&models.Release{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count releases: %v", err)
	}

	var releases []*models.Release
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&releases).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list releases: %v", err)
	}

	protoReleases := make([]*release.Release, len(releases))
	for i, r := range releases {
		protoReleases[i] = releaseToProto(r)
	}

	return &release.ListResponse{Releases: protoReleases, Total: int32(total)}, nil
}

func (s *Release) Search(ctx context.Context, req *release.SearchRequest) (*release.SearchResponse, error) {
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

	var total int64
	if err := tenantDB.Model(&models.Release{}).Where("status = ?", req.Query).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count releases: %v", err)
	}

	var releases []*models.Release
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.Where("status = ?", req.Query).Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&releases).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search releases: %v", err)
	}

	protoReleases := make([]*release.Release, len(releases))
	for i, r := range releases {
		protoReleases[i] = releaseToProto(r)
	}

	return &release.SearchResponse{Releases: protoReleases, Total: int32(total)}, nil
}

func releaseToProto(r *models.Release) *release.Release {
	p := &release.Release{
		Id:          r.ID.String(),
		Status:      r.Status,
		OwnerId:     r.OwnerID.String(),
		ChangedById: r.ChangedByID.String(),
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.TargetID != nil {
		p.TargetId = r.TargetID.String()
	}
	if r.PayloadID != nil {
		p.PayloadId = r.PayloadID.String()
	}
	return p
}
