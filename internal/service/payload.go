package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/payload"
	"zxc/internal/authz"
	"zxc/internal/consts"
	"zxc/internal/db"
	"zxc/internal/models"
	"zxc/internal/storage"
)

type Payload struct {
	payload.UnimplementedPayloadServiceServer
	db    *gorm.DB
	cache *db.Cache
}

func NewPayload(db *gorm.DB, cache *db.Cache) *Payload {
	return &Payload{db: db, cache: cache}
}

const maxPayloadSize = 50 * 1024 * 1024

func validatePayloadZip(content []byte, config string) error {
	r, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return fmt.Errorf("not a valid zip archive: %w", err)
	}
	for _, f := range r.File {
		if f.Name != config {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open config file: %w", err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read config file: %w", err)
		}
		s := string(b)
		if !strings.Contains(s, consts.AUTH) {
			return fmt.Errorf("config file must contain %s", consts.AUTH)
		}
		if !strings.Contains(s, consts.URL) {
			return fmt.Errorf("config file must contain %s", consts.URL)
		}
		return nil
	}
	return fmt.Errorf("config file %q not found in zip", config)
}

func (s *Payload) Create(ctx context.Context, req *payload.CreateRequest) (*payload.CreateResponse, error) {
	if len(req.Content) == 0 {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}
	if len(req.Content) > maxPayloadSize {
		return nil, status.Errorf(codes.InvalidArgument, "content exceeds maximum allowed size of 50 MB")
	}
	if req.Config == "" {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}
	if err := validatePayloadZip(req.Content, req.Config); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payload zip: %v", err)
	}

	authUserID, err := userID(ctx)
	if err != nil {
		return nil, err
	}
	if err := assertOwner(req.OwnerId, authUserID, "owner_id"); err != nil {
		return nil, err
	}

	tenantID, err := parseID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	ten, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "payload.create", ten, authz.Resource{Type: "payload"}, authz.Related{}); err != nil {
		return nil, err
	}

	payloadID := uuid.New()

	mc, bucket, err := storage.ClientFromConnectionString(ten.Storage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to connect to storage: %v", err)
	}

	name := req.Name
	if name == "" {
		name = "payload"
	}
	scriptPath := fmt.Sprintf("payloads/%s/%s", payloadID, name)
	if err := mc.Upload(ctx, bucket, scriptPath, bytes.NewReader(req.Content), int64(len(req.Content)), "application/zip"); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload payload: %v", err)
	}

	p := &models.Payload{
		ID:      payloadID,
		Path:    scriptPath,
		OwnerID: authUserID,
		Config:  req.Config,
		Start:   req.Start,
		Stop:    req.Stop,
	}
	if err := tenantDB.Create(p).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create payload: %v", err)
	}

	return &payload.CreateResponse{Payload: payloadToProto(p)}, nil
}

func (s *Payload) Get(ctx context.Context, req *payload.GetRequest) (*payload.GetResponse, error) {
	id, err := parseID(req.Id, "id")
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

	var p models.Payload
	if err := tenantDB.First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "payload not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get payload: %v", err)
	}
	if _, err := authorize(ctx, "payload.get", tenant, authz.Resource{
		Type:    "payload",
		OwnerID: p.OwnerID,
	}, authz.Related{}); err != nil {
		return nil, err
	}

	return &payload.GetResponse{Payload: payloadToProto(&p)}, nil
}

func (s *Payload) Update(ctx context.Context, req *payload.UpdateRequest) (*payload.UpdateResponse, error) {
	id, err := parseID(req.Id, "id")
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

	var current models.Payload
	if err := tenantDB.First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "payload not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load payload: %v", err)
	}
	if _, err := authorize(ctx, "payload.update", tenant, authz.Resource{
		Type:    "payload",
		OwnerID: current.OwnerID,
	}, authz.Related{}); err != nil {
		return nil, err
	}

	result := tenantDB.Model(&models.Payload{}).Where("id = ?", id).Updates(&models.Payload{Path: req.Path, Config: req.Config, Start: req.Start, Stop: req.Stop})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update payload: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "payload not found")
	}

	var updated models.Payload
	if err := tenantDB.First(&updated, "id = ?", id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated payload: %v", err)
	}

	return &payload.UpdateResponse{Payload: payloadToProto(&updated)}, nil
}

func (s *Payload) Delete(ctx context.Context, req *payload.DeleteRequest) (*payload.DeleteResponse, error) {
	id, err := parseID(req.Id, "id")
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

	var current models.Payload
	if err := tenantDB.First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "payload not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load payload: %v", err)
	}
	if _, err := authorize(ctx, "payload.delete", tenant, authz.Resource{
		Type:    "payload",
		OwnerID: current.OwnerID,
	}, authz.Related{}); err != nil {
		return nil, err
	}

	result := tenantDB.Where("id = ?", id).Delete(&models.Payload{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete payload: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "payload not found")
	}

	return &payload.DeleteResponse{Success: true}, nil
}

func (s *Payload) List(ctx context.Context, req *payload.ListRequest) (*payload.ListResponse, error) {
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
	if _, err := authorize(ctx, "payload.list", tenant, authz.Resource{Type: "payload"}, authz.Related{}); err != nil {
		return nil, err
	}

	var total int64
	if err := tenantDB.Model(&models.Payload{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count payloads: %v", err)
	}

	var payloads []*models.Payload
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&payloads).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list payloads: %v", err)
	}

	out := make([]*payload.Payload, len(payloads))
	for i, p := range payloads {
		out[i] = payloadToProto(p)
	}

	return &payload.ListResponse{Payloads: out, Total: int32(total)}, nil
}

func payloadToProto(p *models.Payload) *payload.Payload {
	return &payload.Payload{
		Id:        p.ID.String(),
		Path:      p.Path,
		Name:      path.Base(p.Path),
		OwnerId:   p.OwnerID.String(),
		Config:    p.Config,
		Start:     p.Start,
		Stop:      p.Stop,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
