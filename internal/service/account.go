package service

import (
	"context"
	"errors"
	"sort"
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
	id := uuid.UUID(req.Id)

	_, db, err := accountDB(ctx)
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

	return &account.GetResponse{Account: s.proto(&a)}, nil
}

func (s *Account) List(ctx context.Context, req *account.ListRequest) (*account.ListResponse, error) {
	page, size := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	_, db, err := accountDB(ctx)
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
		out[i] = s.proto(a)
	}

	return &account.ListResponse{Accounts: out, Total: int32(total)}, nil
}

func (s *Account) Disable(ctx context.Context, req *account.DisableRequest) (*account.DisableResponse, error) {
	id := uuid.UUID(req.Id)

	_, db, err := accountDB(ctx)
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
	return &account.DisableResponse{Account: s.proto(&current)}, nil
}

func (s *Account) GetTalks(ctx context.Context, req *account.GetTalksRequest) (*account.GetTalksResponse, error) {
	profileID := uuid.UUID(req.ProfileId)

	_, db, err := accountDB(ctx)
	if err != nil {
		return nil, err
	}

	var talks []*models.Talk
	if err := db.Where("profile_id = ?", profileID).Find(&talks).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get talks: %v", err)
	}

	out := make([]*account.Talk, len(talks))
	for i, t := range talks {
		var posts []*models.Post
		if err := db.Where("talk_id = ?", t.ID).Find(&posts).Error; err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get posts: %v", err)
		}

		var files []*models.File
		if err := db.Where("talk_id = ?", t.ID).Find(&files).Error; err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get files: %v", err)
		}

		items := make([]*account.TalkItem, 0, len(posts)+len(files))
		for _, p := range posts {
			items = append(items, &account.TalkItem{
				CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Content:   &account.TalkItem_Post{Post: &account.Post{Id: p.ID[:], ContactId: p.ContactID[:], Text: p.Text, CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}},
			})
		}
		for _, f := range files {
			items = append(items, &account.TalkItem{
				CreatedAt: f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Content:   &account.TalkItem_File{File: &account.File{Id: f.ID[:], ContactId: f.ContactID[:], Name: f.Name, CreatedAt: f.CreatedAt.Format("2006-01-02T15:04:05Z07:00")}},
			})
		}

		sort.Slice(items, func(a, b int) bool {
			return items[a].CreatedAt < items[b].CreatedAt
		})

		out[i] = &account.Talk{Id: t.ID[:], Items: items}
	}

	return &account.GetTalksResponse{Talks: out}, nil
}

func (s *Account) proto(a *models.Profile) *account.Account {
	return &account.Account{
		Id:        a.ID[:],
		Name:      a.Name,
		Status:    a.Status,
		CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
