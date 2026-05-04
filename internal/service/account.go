package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/account"
	"zxc/internal/models"
)

type Account struct {
	account.UnimplementedAccountServiceServer
}

func NewAccount() *Account {
	return &Account{}
}

func (s *Account) Get(ctx context.Context, req *account.GetRequest) (*account.GetResponse, error) {
	id := uuid.MustParse(req.Id)

	_, db, err := ctxAccountDB(ctx)
	if err != nil {
		return nil, err
	}

	var a models.Profile
	if err := db.First(&a, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get account: %v", err)
	}

	return &account.GetResponse{Account: profileToProto(&a)}, nil
}

func (s *Account) List(ctx context.Context, req *account.ListRequest) (*account.ListResponse, error) {
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
	if err := db.Model(&models.Profile{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count accounts: %v", err)
	}

	var profiles []*models.Profile
	offset := (int(page) - 1) * int(size)
	if err := db.Order("created_at DESC").Limit(int(size)).Offset(offset).Find(&profiles).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list accounts: %v", err)
	}

	out := make([]*account.Account, len(profiles))
	for i, a := range profiles {
		out[i] = profileToProto(a)
	}

	return &account.ListResponse{Accounts: out, Total: int32(total)}, nil
}

func (s *Account) Disable(ctx context.Context, req *account.DisableRequest) (*account.DisableResponse, error) {
	id := uuid.MustParse(req.Id)

	_, db, err := ctxAccountDB(ctx)
	if err != nil {
		return nil, err
	}

	var current models.Profile
	if err := db.First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load account: %v", err)
	}

	if err := db.Model(&models.Profile{}).
		Where("id = ? AND status <> ? AND deleted_at IS NULL", id, models.ProfileDisabled).
		Updates(map[string]any{
			"status":     models.ProfileDisabled,
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to disable account: %v", err)
	}

	current.Status = models.ProfileDisabled
	return &account.DisableResponse{Account: profileToProto(&current)}, nil
}

func profileToProto(a *models.Profile) *account.Account {
	return &account.Account{
		Id:        a.ID.String(),
		Name:      a.Name,
		Status:    a.Status,
		CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
