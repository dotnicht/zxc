package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/account"
	"zxc/internal/authz"
	"zxc/internal/db"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

type Account struct {
	account.UnimplementedAccountServiceServer
	db    *gorm.DB
	cache *db.Cache
	store *workflow.Store
}

func NewAccount(db *gorm.DB, cache *db.Cache, store *workflow.Store) *Account {
	return &Account{db: db, cache: cache, store: store}
}

func validateAccountName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("name must be 255 characters or less")
	}
	return nil
}

func (s *Account) Create(ctx context.Context, req *account.CreateRequest) (*account.CreateResponse, error) {
	if err := validateAccountName(req.Name); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid account name: %v", err)
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorizeAction(ctx, "account.create", tenant, authz.Resource{Type: "account"}, authz.Related{}); err != nil {
		return nil, err
	}

	a := &models.Account{Name: req.Name}
	if err := tenantDB.WithContext(ctx).Create(a).Error; err != nil {
		if isDuplicateKeyError(err) {
			return nil, status.Error(codes.AlreadyExists, "account with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}
	if err := s.store.RootTransaction(ctx, func(tx *gorm.DB) error {
		return s.store.RecordEvent(ctx, tx, workflow.EventInput{
			Kind:          "account_created",
			AggregateType: "account",
			AggregateID:   a.ID.String(),
			TenantID:      &tenantID,
			Payload: map[string]any{
				"account_id": a.ID.String(),
				"name":       a.Name,
			},
		})
	}); err != nil {
		cleanupErr := tenantDB.WithContext(ctx).Unscoped().Delete(&models.Account{}, "id = ?", a.ID).Error
		return nil, status.Errorf(codes.Internal, "failed to persist account creation: %v", errors.Join(err, cleanupErr))
	}

	return &account.CreateResponse{Account: accountToProto(a)}, nil
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

func (s *Account) Update(ctx context.Context, req *account.UpdateRequest) (*account.UpdateResponse, error) {
	id, err := parseUUID(req.Id, "id")
	if err != nil {
		return nil, err
	}
	if err := validateAccountName(req.Name); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid account name: %v", err)
	}

	tenantID, err := parseUUID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}

	tenant, tenantDB, err := resolveTenantDB(ctx, s.cache, tenantID)
	if err != nil {
		return nil, err
	}
	if _, err := authorizeAction(ctx, "account.update", tenant, authz.Resource{Type: "account"}, authz.Related{}); err != nil {
		return nil, err
	}

	var current models.Account
	if err := tenantDB.WithContext(ctx).First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load account: %v", err)
	}

	result := tenantDB.WithContext(ctx).Model(&models.Account{}).Where("id = ?", id).Update("name", req.Name)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return nil, status.Error(codes.AlreadyExists, "account with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to update account: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "account not found")
	}

	var updated models.Account
	if err := tenantDB.WithContext(ctx).First(&updated, "id = ?", id).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch updated account: %v", err)
	}
	if err := s.store.RootTransaction(ctx, func(tx *gorm.DB) error {
		return s.store.RecordEvent(ctx, tx, workflow.EventInput{
			Kind:          "account_updated",
			AggregateType: "account",
			AggregateID:   updated.ID.String(),
			TenantID:      &tenantID,
			Payload: map[string]any{
				"account_id": updated.ID.String(),
				"name":       updated.Name,
			},
		})
	}); err != nil {
		revertErr := tenantDB.WithContext(ctx).Model(&models.Account{}).Where("id = ?", id).Update("name", current.Name).Error
		return nil, status.Errorf(codes.Internal, "failed to persist account update: %v", errors.Join(err, revertErr))
	}

	return &account.UpdateResponse{Account: accountToProto(&updated)}, nil
}

func (s *Account) Delete(ctx context.Context, req *account.DeleteRequest) (*account.DeleteResponse, error) {
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
	if _, err := authorizeAction(ctx, "account.delete", tenant, authz.Resource{Type: "account"}, authz.Related{}); err != nil {
		return nil, err
	}

	var current models.Account
	if err := tenantDB.WithContext(ctx).First(&current, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "account not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load account: %v", err)
	}

	result := tenantDB.WithContext(ctx).Where("id = ?", id).Delete(&models.Account{})
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete account: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "account not found")
	}
	if err := s.store.RootTransaction(ctx, func(tx *gorm.DB) error {
		return s.store.RecordEvent(ctx, tx, workflow.EventInput{
			Kind:          "account_deleted",
			AggregateType: "account",
			AggregateID:   current.ID.String(),
			TenantID:      &tenantID,
			Payload: map[string]any{
				"account_id": current.ID.String(),
				"name":       current.Name,
			},
		})
	}); err != nil {
		revertErr := tenantDB.WithContext(ctx).Unscoped().Model(&models.Account{}).Where("id = ?", current.ID).Updates(map[string]any{
			"deleted_at": nil,
			"updated_at": current.UpdatedAt,
		}).Error
		return nil, status.Errorf(codes.Internal, "failed to persist account deletion: %v", errors.Join(err, revertErr))
	}

	return &account.DeleteResponse{Success: true}, nil
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

func isDuplicateKeyError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "accounts_name_active_idx"))
}
