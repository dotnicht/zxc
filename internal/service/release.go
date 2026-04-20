package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/release"
	"zxc/internal/db"
	"zxc/internal/models"
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
}

func NewRelease(db *gorm.DB, cache *db.Cache) *Release {
	return &Release{db: db, cache: cache}
}

func (s *Release) resolveTenant(ctx context.Context, tenantIDStr string) (*models.Tenant, *gorm.DB, error) {
	_, tenant, _, err := authenticatedTenant(ctx, tenantIDStr)
	if err != nil {
		return nil, nil, err
	}
	tenantDB, err := s.cache.Get(tenant.Database)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to connect to tenant database: %v", err)
	}
	return tenant, tenantDB, nil
}

func (s *Release) Create(ctx context.Context, req *release.CreateRequest) (*release.CreateResponse, error) {
	authUserID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.OwnerId != "" && req.OwnerId != authUserID.String() {
		return nil, status.Error(codes.PermissionDenied, "owner_id must match authenticated user")
	}

	targetID, err := uuid.Parse(req.TargetId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target_id: must be a valid UUID")
	}

	payloadID, err := uuid.Parse(req.PayloadId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payload_id: must be a valid UUID")
	}

	releaseStatus := req.Status
	if releaseStatus == "" {
		releaseStatus = models.ReleaseUnknown
	} else if !validReleaseStatuses[releaseStatus] {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status %q: must be one of unknown, deployed, dead, alive", releaseStatus)
	}

	tenant, tenantDB, err := s.resolveTenant(ctx, req.TenantId)
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

	route := &models.Route{ID: rel.ID, TenantID: tenant.ID}
	if err := s.db.Create(route).Error; err != nil {
		tenantDB.Delete(&models.Release{}, "id = ?", rel.ID)
		return nil, status.Errorf(codes.Internal, "failed to create route: %v", err)
	}

	return &release.CreateResponse{Release: releaseToProto(rel)}, nil
}

func (s *Release) Get(ctx context.Context, req *release.GetRequest) (*release.GetResponse, error) {
	releaseID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id: must be a valid UUID")
	}

	_, tenantDB, err := s.resolveTenant(ctx, req.TenantId)
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
	releaseID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id: must be a valid UUID")
	}

	authUserID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.UserId != "" && req.UserId != authUserID.String() {
		return nil, status.Error(codes.PermissionDenied, "user_id must match authenticated user")
	}

	_, tenantDB, err := s.resolveTenant(ctx, req.TenantId)
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

	_, tenantDB, err := s.resolveTenant(ctx, req.TenantId)
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

	_, tenantDB, err := s.resolveTenant(ctx, req.TenantId)
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
