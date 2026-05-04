package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/session"
	"zxc/internal/models"
)

type Session struct {
	session.UnimplementedSessionServiceServer
}

func NewSession() *Session {
	return &Session{}
}

func validateSessionStatus(raw string) error {
	switch raw {
	case models.SessionOnline, models.SessionOffline, models.SessionSync:
		return nil
	}
	return status.Errorf(codes.InvalidArgument, "invalid status: must be one of %q, %q, %q", models.SessionOnline, models.SessionOffline, models.SessionSync)
}

func (s *Session) Create(ctx context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	profileID := uuid.MustParse(req.AccountId)
	if err := validateSessionStatus(req.Status); err != nil {
		return nil, err
	}

	_, db, err := ctxAccountDB(ctx)
	if err != nil {
		return nil, err
	}

	var profile models.Profile
	if err := db.First(&profile, "id = ?", profileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load account: %v", err)
	}

	record := &models.Session{
		ProfileID: profileID,
		Status:    req.Status,
	}
	if err := db.Create(record).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}
	if profile.Status == models.ProfileUnknown {
		result := db.Model(&models.Profile{}).
			Where("id = ? AND status = ? AND deleted_at IS NULL", profileID, models.ProfileUnknown).
			Updates(map[string]any{
				"status":     models.ProfileActive,
				"updated_at": time.Now().UTC(),
			})
		if result.Error != nil {
			cleanupErr := db.Unscoped().Delete(&models.Session{}, "id = ?", record.ID).Error
			return nil, status.Errorf(codes.Internal, "failed to activate account: %v", errors.Join(result.Error, cleanupErr))
		}
	}

	return &session.CreateResponse{Session: sessionToProto(record)}, nil
}

func (s *Session) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	id := uuid.MustParse(req.Id)

	_, db, err := ctxAccountDB(ctx)
	if err != nil {
		return nil, err
	}

	var record models.Session
	if err := db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	return &session.GetResponse{Session: sessionToProto(&record)}, nil
}

func (s *Session) Update(ctx context.Context, req *session.UpdateRequest) (*session.UpdateResponse, error) {
	id := uuid.MustParse(req.Id)
	profileID := uuid.MustParse(req.AccountId)
	if err := validateSessionStatus(req.Status); err != nil {
		return nil, err
	}

	_, db, err := ctxAccountDB(ctx)
	if err != nil {
		return nil, err
	}

	var checkProfile models.Profile
	if err := db.First(&checkProfile, "id = ?", profileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load account: %v", err)
	}

	var previous models.Session
	if err := db.First(&previous, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load session: %v", err)
	}

	result := db.Model(&models.Session{}).Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"profile_id": profileID,
			"status":     req.Status,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update session: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "session not found")
	}

	var updated models.Session
	if err := db.First(&updated, "id = ?", id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated session: %v", err)
	}

	return &session.UpdateResponse{Session: sessionToProto(&updated)}, nil
}

func (s *Session) Delete(ctx context.Context, req *session.DeleteRequest) (*session.DeleteResponse, error) {
	id := uuid.MustParse(req.Id)

	_, db, err := ctxAccountDB(ctx)
	if err != nil {
		return nil, err
	}

	var current models.Session
	if err := db.First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load session: %v", err)
	}

	result := db.Where("id = ?", id).Delete(&models.Session{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete session: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "session not found")
	}

	return &session.DeleteResponse{Success: true}, nil
}

func (s *Session) List(ctx context.Context, req *session.ListRequest) (*session.ListResponse, error) {
	page, size := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	_, db, err := ctxAccountDB(ctx)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := db.Model(&models.Session{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count sessions: %v", err)
	}

	var records []*models.Session
	offset := (int(page) - 1) * int(size)
	if err := db.Order("created_at DESC").Limit(int(size)).Offset(offset).Find(&records).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sessions: %v", err)
	}

	out := make([]*session.Session, len(records))
	for i, record := range records {
		out[i] = sessionToProto(record)
	}

	return &session.ListResponse{Sessions: out, Total: int32(total)}, nil
}

func sessionToProto(record *models.Session) *session.Session {
	return &session.Session{
		Id:        record.ID.String(),
		AccountId: record.ProfileID.String(),
		Status:    record.Status,
		CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
