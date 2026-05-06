package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/cschleiden/go-workflows/client"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"zxc/api/release"
	"zxc/internal/infra"
	"zxc/internal/jobs"
	"zxc/internal/models"
)

type Release struct {
	release.UnimplementedReleaseServiceServer
}

func NewRelease() *Release {
	return &Release{}
}

func (s *Release) Create(ctx context.Context, req *release.CreateRequest) (*release.CreateResponse, error) {
	authUserID, err := ctxUserID(ctx)
	if err != nil {
		return nil, err
	}

	targetID := uuid.UUID(req.TargetId)
	payloadID := uuid.UUID(req.PayloadId)

	_, db, err := ctxDeployDB(ctx)
	if err != nil {
		return nil, err
	}

	var t models.Target
	if err := db.First(&t, "id = ?", targetID).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "target not found")
	}

	var p models.Payload
	if err := db.First(&p, "id = ?", payloadID).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "payload not found")
	}

	rel := &models.Release{
		Status:      models.ReleaseUnknown,
		OwnerID:     authUserID,
		TargetID:    &targetID,
		PayloadID:   &payloadID,
		ChangedByID: authUserID,
	}
	if err := db.Create(rel).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create release: %v", err)
	}

	return &release.CreateResponse{Release: releaseToProto(rel)}, nil
}

func (s *Release) Get(ctx context.Context, req *release.GetRequest) (*release.GetResponse, error) {
	id := uuid.UUID(req.Id)

	_, db, err := ctxDeployDB(ctx)
	if err != nil {
		return nil, err
	}

	var rel models.Release
	if err := db.First(&rel, "id = ?", id).Error; err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "release not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get release: %v", err)
	}

	return &release.GetResponse{Release: releaseToProto(&rel)}, nil
}

func (s *Release) Deploy(ctx context.Context, req *release.DeployRequest) (*release.DeployResponse, error) {
	id := uuid.UUID(req.Id)

	authUserID, err := ctxUserID(ctx)
	if err != nil {
		return nil, err
	}

	tenant, db, err := ctxDeployDB(ctx)
	if err != nil {
		return nil, err
	}

	var current models.Release
	if err := db.First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "release not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load release: %v", err)
	}

	res := db.Model(&models.Release{}).
		Where("id = ? AND status = ? AND deleted_at IS NULL", id, models.ReleaseUnknown).
		Updates(map[string]any{
			"status":        models.ReleaseWait,
			"changed_by_id": authUserID,
			"updated_at":    time.Now().UTC(),
		})
	if res.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update release: %v", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "release not found")
	}

	wfb, err := infra.WorkflowBackend(tenant.Jobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initialise jobs backend: %v", err)
	}
	wfc := client.New(wfb)
	if _, err := wfc.CreateWorkflowInstance(ctx, client.WorkflowInstanceOptions{
		InstanceID: "deploy:" + id.String(),
	}, jobs.Deploy, jobs.DeployArgs{
		TenantID:    tenant.ID,
		ReleaseID:   id,
		ChangedByID: authUserID,
	}); err != nil {
		db.Model(&models.Release{}).
			Where("id = ? AND status = ? AND deleted_at IS NULL", id, models.ReleaseWait).
			Updates(map[string]any{
				"status":        models.ReleaseUnknown,
				"changed_by_id": authUserID,
				"updated_at":    time.Now().UTC(),
			})
		return nil, status.Errorf(codes.Internal, "failed to start deploy workflow: %v", err)
	}

	var rel models.Release
	if err := db.First(&rel, "id = ?", id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get release: %v", err)
	}

	return &release.DeployResponse{Release: releaseToProto(&rel)}, nil
}

func (s *Release) List(ctx context.Context, req *release.ListRequest) (*release.ListResponse, error) {
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
	if err := db.Model(&models.Release{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count releases: %v", err)
	}

	var releases []*models.Release
	offset := (int(page) - 1) * int(size)
	if err := db.Order("created_at DESC").Limit(int(size)).Offset(offset).Find(&releases).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list releases: %v", err)
	}

	out := make([]*release.Release, len(releases))
	for i, r := range releases {
		out[i] = releaseToProto(r)
	}

	return &release.ListResponse{Releases: out, Total: int32(total)}, nil
}

func releaseToProto(r *models.Release) *release.Release {
	p := &release.Release{
		Id:          r.ID[:],
		Status:      r.Status,
		OwnerId:     r.OwnerID[:],
		ChangedById: r.ChangedByID[:],
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.TargetID != nil {
		p.TargetId = r.TargetID[:]
	}
	if r.PayloadID != nil {
		p.PayloadId = r.PayloadID[:]
	}
	return p
}
