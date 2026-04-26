package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"zxc/api/tenant"
	"zxc/internal/authz"
	"zxc/internal/config"
	"zxc/internal/db"
	"zxc/internal/models"
	"zxc/internal/storage"
)

type Tenant struct {
	tenant.UnimplementedTenantServiceServer
	db   *gorm.DB
	cfg  *config.Config
	root *models.User
}

func NewTenant(db *gorm.DB, cfg *config.Config, root *models.User) *Tenant {
	return &Tenant{db: db, cfg: cfg, root: root}
}

func validateTenantName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 63 {
		return fmt.Errorf("name must be 63 characters or less")
	}
	if name[0] < 'a' || name[0] > 'z' {
		return fmt.Errorf("name must start with a lowercase letter")
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return fmt.Errorf("name must contain only lowercase letters, digits, and underscores")
		}
	}
	return nil
}

func (s *Tenant) Create(ctx context.Context, req *tenant.CreateRequest) (*tenant.CreateResponse, error) {
	if _, err := authorize(ctx, "tenant.create", nil, authz.Resource{Type: "tenant"}, authz.Related{}); err != nil {
		return nil, err
	}

	if err := validateTenantName(req.Name); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant name: %v", err)
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Tenant{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check tenant existence: %v", err)
	}
	if count > 0 {
		return nil, status.Error(codes.AlreadyExists, "tenant with this name already exists")
	}

	connStr := req.Database
	if connStr == "" {
		connStr = s.generateConnectionString(req.Name)
	}

	storageStr := req.Storage
	if storageStr == "" && s.cfg.Storage != "" {
		client, bucket, err := storage.ClientFromConnectionString(s.cfg.Storage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse storage config: %v", err)
		}
		if err := client.CreateFolder(ctx, bucket, req.Name); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create tenant storage folder: %v", err)
		}
		storageStr = strings.TrimRight(s.cfg.Storage, "/") + "/" + req.Name
	}

	t := &models.Tenant{
		Name:     req.Name,
		Database: connStr,
		Storage:  storageStr,
		OwnerID:  s.root.ID,
	}

	if err := s.db.WithContext(ctx).Create(t).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "tenant with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create tenant: %v", err)
	}

	if err := s.createTenantDatabase(req.Name); err != nil {
		cleanupErr := s.db.WithContext(ctx).Delete(&models.Tenant{}, "id = ?", t.ID).Error
		return nil, status.Errorf(codes.Internal, "failed to create tenant database: %v", errors.Join(err, cleanupErr))
	}

	if err := s.runTenantMigrations(connStr); err != nil {
		cleanupErr := errors.Join(s.dropTenantDatabase(req.Name), s.db.WithContext(ctx).Delete(&models.Tenant{}, "id = ?", t.ID).Error)
		return nil, status.Errorf(codes.Internal, "failed to run tenant migrations: %v", errors.Join(err, cleanupErr))
	}

	if err := s.seedTenantOwner(connStr, s.root); err != nil {
		cleanupErr := errors.Join(s.dropTenantDatabase(req.Name), s.db.WithContext(ctx).Delete(&models.Tenant{}, "id = ?", t.ID).Error)
		return nil, status.Errorf(codes.Internal, "failed to seed tenant owner: %v", errors.Join(err, cleanupErr))
	}

	return &tenant.CreateResponse{Tenant: s.modelToProto(t)}, nil
}

func (s *Tenant) Get(ctx context.Context, req *tenant.GetRequest) (*tenant.GetResponse, error) {
	if _, err := authorize(ctx, "tenant.get", nil, authz.Resource{Type: "tenant"}, authz.Related{}); err != nil {
		return nil, err
	}

	id, err := parseID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	var t models.Tenant
	if err := s.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get tenant: %v", err)
	}

	return &tenant.GetResponse{Tenant: s.modelToProto(&t)}, nil
}

func (s *Tenant) Update(ctx context.Context, req *tenant.UpdateRequest) (*tenant.UpdateResponse, error) {
	if _, err := authorize(ctx, "tenant.update", nil, authz.Resource{Type: "tenant"}, authz.Related{}); err != nil {
		return nil, err
	}

	id, err := parseID(req.Id, "id")
	if err != nil {
		return nil, err
	}

	var t models.Tenant
	if err := s.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get tenant: %v", err)
	}

	if req.Name != "" {
		if err := validateTenantName(req.Name); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid tenant name: %v", err)
		}
		t.Name = req.Name
	}
	if req.Database != "" {
		t.Database = req.Database
	}
	if req.Storage != "" {
		t.Storage = req.Storage
	}
	if req.OwnerId != "" {
		ownerID, err := parseID(req.OwnerId, "owner_id")
		if err != nil {
			return nil, err
		}
		t.OwnerID = ownerID
	}

	if err := s.db.WithContext(ctx).Save(&t).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "tenant with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to update tenant: %v", err)
	}

	return &tenant.UpdateResponse{Tenant: s.modelToProto(&t)}, nil
}

func (s *Tenant) List(ctx context.Context, req *tenant.ListRequest) (*tenant.ListResponse, error) {
	if _, err := authorize(ctx, "tenant.list", nil, authz.Resource{Type: "tenant"}, authz.Related{}); err != nil {
		return nil, err
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	var total int64
	if err := s.db.WithContext(ctx).Model(&models.Tenant{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tenants: %v", err)
	}

	var tenants []*models.Tenant
	offset := (page - 1) * pageSize
	if err := s.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tenants).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tenants: %v", err)
	}

	out := make([]*tenant.Tenant, len(tenants))
	for i, t := range tenants {
		out[i] = s.modelToProto(t)
	}

	return &tenant.ListResponse{Tenants: out, Total: int32(total)}, nil
}

func (s *Tenant) Search(ctx context.Context, req *tenant.SearchRequest) (*tenant.SearchResponse, error) {
	if _, err := authorize(ctx, "tenant.search", nil, authz.Resource{Type: "tenant"}, authz.Related{}); err != nil {
		return nil, err
	}

	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "search query is required")
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	pattern := "%" + req.Query + "%"
	var total int64
	if err := s.db.WithContext(ctx).Model(&models.Tenant{}).Where("name ILIKE ?", pattern).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search tenants: %v", err)
	}

	var tenants []*models.Tenant
	offset := (page - 1) * pageSize
	if err := s.db.WithContext(ctx).Where("name ILIKE ?", pattern).Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&tenants).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search tenants: %v", err)
	}

	out := make([]*tenant.Tenant, len(tenants))
	for i, t := range tenants {
		out[i] = s.modelToProto(t)
	}

	return &tenant.SearchResponse{Tenants: out, Total: int32(total)}, nil
}

func (s *Tenant) generateConnectionString(tenantName string) string {
	u, err := url.Parse(s.cfg.Database)
	if err != nil {
		return s.cfg.Database
	}
	u.Path = "/" + sanitizeDatabaseName(tenantName)
	return u.String()
}

func (s *Tenant) adminDSN() string {
	u, err := url.Parse(s.cfg.Database)
	if err != nil {
		return s.cfg.Database
	}
	u.Path = "/postgres"
	return u.String()
}

func (s *Tenant) createTenantDatabase(dbName string) error {
	sqlDB, err := sql.Open("postgres", s.adminDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer sqlDB.Close()

	safeName := sanitizeDatabaseName(dbName)
	_, err = sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s", safeName))
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("database %s already exists", safeName)
		}
		return fmt.Errorf("failed to create database: %w", err)
	}
	return nil
}

func (s *Tenant) dropTenantDatabase(dbName string) error {
	sqlDB, err := sql.Open("postgres", s.adminDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer sqlDB.Close()

	safeName := sanitizeDatabaseName(dbName)
	_, err = sqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", safeName))
	if err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	return nil
}

func (s *Tenant) seedTenantOwner(connStr string, owner *models.User) error {
	tenantDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	sqlDB, err := tenantDB.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	u := &models.User{ID: owner.ID, Name: owner.Name}
	if err := tenantDB.Create(u).Error; err != nil {
		return fmt.Errorf("failed to create owner user in tenant database: %w", err)
	}
	return nil
}

func (s *Tenant) runTenantMigrations(connStr string) error {
	tenantDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	if err := db.RunTenantMigrations(tenantDB); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	sqlDB, err := tenantDB.DB()
	if err == nil {
		sqlDB.Close()
	}
	return nil
}

func (s *Tenant) modelToProto(m *models.Tenant) *tenant.Tenant {
	t := &tenant.Tenant{
		Id:        m.ID.String(),
		Name:      m.Name,
		Database:  m.Database,
		Storage:   m.Storage,
		OwnerId:   m.OwnerID.String(),
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	return t
}

func sanitizeDatabaseName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			result.WriteRune(r + 32)
		}
	}
	return result.String()
}
