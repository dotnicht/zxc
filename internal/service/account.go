package service

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/account"
	"zxc/internal/authz"
	"zxc/internal/db"
	"zxc/internal/models"
)

type Account struct {
	account.UnimplementedAccountServiceServer
	cache *db.Cache
}

func NewAccount(_ *gorm.DB, cache *db.Cache, _ any) *Account {
	return &Account{cache: cache}
}

func (s *Account) Get(ctx context.Context, req *account.GetRequest) (*account.GetResponse, error) {
	id, err := parseUUID(req.Id, "id")
	if err != nil {
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
	if _, err := authorizeAction(ctx, "account.get", tenant, authz.Resource{Type: "account"}, authz.Related{}); err != nil {
		return nil, err
	}

	var a models.Account
	if err := tenantDB.WithContext(ctx).First(&a, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get account: %v", err)
	}

	return &account.GetResponse{Account: accountToProto(&a)}, nil
}

func (s *Account) List(ctx context.Context, req *account.ListRequest) (*account.ListResponse, error) {
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

	tenant, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorizeAction(ctx, "account.list", tenant, authz.Resource{Type: "account"}, authz.Related{}); err != nil {
		return nil, err
	}

	var total int64
	if err := tenantDB.WithContext(ctx).Model(&models.Account{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count accounts: %v", err)
	}

	var accounts []*models.Account
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.WithContext(ctx).Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&accounts).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list accounts: %v", err)
	}

	protoAccounts := make([]*account.Account, len(accounts))
	for i, a := range accounts {
		protoAccounts[i] = accountToProto(a)
	}

	return &account.ListResponse{Accounts: protoAccounts, Total: int32(total)}, nil
}

func (s *Account) Search(ctx context.Context, req *account.SearchRequest) (*account.SearchResponse, error) {
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

	tenant, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorizeAction(ctx, "account.search", tenant, authz.Resource{Type: "account"}, authz.Related{}); err != nil {
		return nil, err
	}

	pattern := "%" + req.Query + "%"
	var total int64
	if err := tenantDB.WithContext(ctx).Model(&models.Account{}).Where("name ILIKE ?", pattern).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count accounts: %v", err)
	}

	var accounts []*models.Account
	offset := (int(page) - 1) * int(pageSize)
	if err := tenantDB.WithContext(ctx).Where("name ILIKE ?", pattern).Order("created_at DESC").Limit(int(pageSize)).Offset(offset).Find(&accounts).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search accounts: %v", err)
	}

	protoAccounts := make([]*account.Account, len(accounts))
	for i, a := range accounts {
		protoAccounts[i] = accountToProto(a)
	}

	return &account.SearchResponse{Accounts: protoAccounts, Total: int32(total)}, nil
}

func accountToProto(a *models.Account) *account.Account {
	return &account.Account{
		Id:        a.ID.String(),
		Name:      a.Name,
		CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
