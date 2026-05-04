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

func (s *Session) Start(ctx context.Context, req *session.StartRequest) (*session.StartResponse, error) {
	return s.setStatus(ctx, req.Id, models.SessionOnline)
}

func (s *Session) Stop(ctx context.Context, req *session.StopRequest) (*session.StopResponse, error) {
	r, err := s.setStatus(ctx, req.Id, models.SessionOffline)
	if err != nil {
		return nil, err
	}
	return &session.StopResponse{Session: r.Session}, nil
}

func (s *Session) setStatus(ctx context.Context, rawID string, newStatus string) (*session.StartResponse, error) {
	id := uuid.MustParse(rawID)

	_, db, err := ctxAccountDB(ctx)
	if err != nil {
		return nil, err
	}

	result := db.Model(&models.Session{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"status":     newStatus,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to update session: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "session not found")
	}

	var record models.Session
	if err := db.First(&record, "id = ?", id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch session: %v", err)
	}

	return &session.StartResponse{Session: sessionToProto(&record)}, nil
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
