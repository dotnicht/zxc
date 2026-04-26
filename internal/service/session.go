package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/session"
	"zxc/internal/authz"
	"zxc/internal/infra"
	"zxc/internal/models"
)

var validSessionStatuses = map[string]bool{
	models.SessionOnline:  true,
	models.SessionOffline: true,
	models.SessionSync:    true,
}

type Session struct {
	session.UnimplementedSessionServiceServer
	cache *infra.Cache
}

func NewSession(cache *infra.Cache) *Session {
	return &Session{cache: cache}
}

func validateSessionStatus(raw string) error {
	if !validSessionStatuses[raw] {
		return status.Errorf(codes.InvalidArgument, "invalid status: must be one of %q, %q, %q", models.SessionOnline, models.SessionOffline, models.SessionSync)
	}
	return nil
}

func (s *Session) Create(ctx context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	accountID := uuid.MustParse(req.AccountId)
	if err := validateSessionStatus(req.Status); err != nil {
		return nil, err
	}

	tenantID := uuid.MustParse(req.TenantId)

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "session.create", tenant, authz.Resource{Type: "session"}, authz.Related{}); err != nil {
		return nil, err
	}

	account, err := loadAccount(ctx, tenantDB, accountID)
	if err != nil {
		return nil, err
	}

	record := &models.Session{
		AccountID: accountID,
		Status:    req.Status,
	}
	if err := tenantDB.WithContext(ctx).Create(record).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}
	if account.Status == models.AccountUnknown {
		if err := tenantDB.WithContext(ctx).Model(&models.Account{}).
			Where("id = ? AND status = ?", accountID, models.AccountUnknown).
			Update("status", models.AccountActive).Error; err != nil {
			cleanupErr := tenantDB.WithContext(ctx).Unscoped().Delete(&models.Session{}, "id = ?", record.ID).Error
			return nil, status.Errorf(codes.Internal, "failed to activate account: %v", errors.Join(err, cleanupErr))
		}
	}

	return &session.CreateResponse{Session: sessionToProto(record)}, nil
}

func (s *Session) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	id := uuid.MustParse(req.Id)
	tenantID := uuid.MustParse(req.TenantId)

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "session.get", tenant, authz.Resource{Type: "session"}, authz.Related{}); err != nil {
		return nil, err
	}

	var record models.Session
	if err := tenantDB.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	return &session.GetResponse{Session: sessionToProto(&record)}, nil
}

func (s *Session) Update(ctx context.Context, req *session.UpdateRequest) (*session.UpdateResponse, error) {
	id := uuid.MustParse(req.Id)
	accountID := uuid.MustParse(req.AccountId)
	if err := validateSessionStatus(req.Status); err != nil {
		return nil, err
	}

	tenantID := uuid.MustParse(req.TenantId)

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "session.update", tenant, authz.Resource{Type: "session"}, authz.Related{}); err != nil {
		return nil, err
	}

	if _, err := loadAccount(ctx, tenantDB, accountID); err != nil {
		return nil, err
	}

	var previous models.Session
	if err := tenantDB.WithContext(ctx).First(&previous, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load session: %v", err)
	}

	result := tenantDB.WithContext(ctx).Model(&models.Session{}).Where("id = ?", id).Updates(map[string]any{
		"account_id": accountID,
		"status":     req.Status,
	})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update session: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "session not found")
	}

	var updated models.Session
	if err := tenantDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated session: %v", err)
	}

	return &session.UpdateResponse{Session: sessionToProto(&updated)}, nil
}

func (s *Session) Delete(ctx context.Context, req *session.DeleteRequest) (*session.DeleteResponse, error) {
	id := uuid.MustParse(req.Id)
	tenantID := uuid.MustParse(req.TenantId)

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "session.delete", tenant, authz.Resource{Type: "session"}, authz.Related{}); err != nil {
		return nil, err
	}

	var current models.Session
	if err := tenantDB.WithContext(ctx).First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load session: %v", err)
	}

	result := tenantDB.WithContext(ctx).Where("id = ?", id).Delete(&models.Session{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete session: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "session not found")
	}

	return &session.DeleteResponse{Success: true}, nil
}

func (s *Session) List(ctx context.Context, req *session.ListRequest) (*session.ListResponse, error) {
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	tenantID := uuid.MustParse(req.TenantId)

	tenant, tenantDB, err := resolve(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorize(ctx, "session.list", tenant, authz.Resource{Type: "session"}, authz.Related{}); err != nil {
		return nil, err
	}

	var total int64
	if err := tenantDB.WithContext(ctx).Model(&models.Session{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count sessions: %v", err)
	}

	var records []*models.Session
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.WithContext(ctx).Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&records).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sessions: %v", err)
	}

	out := make([]*session.Session, len(records))
	for i, record := range records {
		out[i] = sessionToProto(record)
	}

	return &session.ListResponse{Sessions: out, Total: int32(total)}, nil
}

func loadAccount(ctx context.Context, tenantDB *gorm.DB, accountID uuid.UUID) (*models.Account, error) {
	var account models.Account
	if err := tenantDB.WithContext(ctx).First(&account, "id = ?", accountID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load account: %v", err)
	}
	return &account, nil
}

func sessionToProto(record *models.Session) *session.Session {
	return &session.Session{
		Id:        record.ID.String(),
		AccountId: record.AccountID.String(),
		Status:    record.Status,
		CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
