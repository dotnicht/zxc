package service

import (
	"context"
	gosql "database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"zxc/api/tenant"
	"zxc/internal/config"
	"zxc/internal/infra"
	"zxc/internal/models"
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
	if err := validateTenantName(req.Name); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant name: %v", err)
	}

	var countN int64
	if err := s.db.Model(&models.Tenant{}).Where("name = ? AND deleted_at IS NULL", req.Name).Count(&countN).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check tenant existence: %v", err)
	}
	if countN > 0 {
		return nil, status.Error(codes.AlreadyExists, "tenant with this name already exists")
	}

	usersConnStr := s.usersConnStr(req.Name)
	deployConnStr := s.deployConnStr(req.Name)
	accountConnStr := s.accountConnStr(req.Name)

	if req.Database != "" {
		usersConnStr = req.Database
	}
	if req.Deploy != "" {
		deployConnStr = req.Deploy
	}
	if req.Account != "" {
		accountConnStr = req.Account
	}

	storageStr := req.Storage
	if storageStr == "" && s.cfg.Storage != "" {
		client, bucket, err := infra.StorageClientFromConnectionString(s.cfg.Storage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse storage config: %v", err)
		}
		if err := client.CreateFolder(ctx, bucket, req.Name); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create tenant storage folder: %v", err)
		}
		storageStr = strings.TrimRight(s.cfg.Storage, "/") + "/" + req.Name
	}

	t := &models.Tenant{
		Name:            req.Name,
		UsersDatabase:   usersConnStr,
		DeployDatabase:  deployConnStr,
		AccountDatabase: accountConnStr,
		Storage:         storageStr,
		OwnerID:         s.root.ID,
	}

	if err := s.db.Create(t).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "tenant with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create tenant: %v", err)
	}

	dbName := sanitizeDatabaseName(req.Name)
	for _, entry := range []struct {
		customDSN string
		suffix    string
	}{
		{req.Database, "_users"},
		{req.Deploy, "_deploy"},
		{req.Account, "_account"},
	} {
		if entry.customDSN != "" {
			continue
		}
		if err := s.createTenantDatabase(dbName + entry.suffix); err != nil {
			s.db.Delete(&models.Tenant{}, "id = ?", t.ID)
			return nil, status.Errorf(codes.Internal, "failed to create %s database: %v", entry.suffix, err)
		}
	}

	for _, m := range []struct {
		connStr string
		fn      func(*gorm.DB) error
		label   string
	}{
		{usersConnStr, infra.RunUsersMigrations, "users"},
		{deployConnStr, infra.RunDeployMigrations, "deploy"},
		{accountConnStr, infra.RunAccountMigrations, "account"},
	} {
		if err := s.runMigrationsOn(m.connStr, m.fn); err != nil {
			s.db.Delete(&models.Tenant{}, "id = ?", t.ID)
			return nil, status.Errorf(codes.Internal, "failed to run %s migrations: %v", m.label, err)
		}
	}

	if err := s.seedTenantOwner(usersConnStr, s.root); err != nil {
		s.db.Delete(&models.Tenant{}, "id = ?", t.ID)
		return nil, status.Errorf(codes.Internal, "failed to seed tenant owner: %v", err)
	}

	return &tenant.CreateResponse{Tenant: s.modelToProto(t)}, nil
}

func (s *Tenant) Get(ctx context.Context, req *tenant.GetRequest) (*tenant.GetResponse, error) {
	id := uuid.MustParse(req.Id)

	var t models.Tenant
	if err := s.db.First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get tenant: %v", err)
	}

	return &tenant.GetResponse{Tenant: s.modelToProto(&t)}, nil
}

func (s *Tenant) Update(ctx context.Context, req *tenant.UpdateRequest) (*tenant.UpdateResponse, error) {
	id := uuid.MustParse(req.Id)

	var t models.Tenant
	if err := s.db.First(&t, "id = ?", id).Error; err != nil {
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
		t.UsersDatabase = req.Database
	}
	if req.Deploy != "" {
		t.DeployDatabase = req.Deploy
	}
	if req.Account != "" {
		t.AccountDatabase = req.Account
	}
	if req.Storage != "" {
		t.Storage = req.Storage
	}
	if req.OwnerId != "" {
		t.OwnerID = uuid.MustParse(req.OwnerId)
	}

	t.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&t).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "tenant with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to update tenant: %v", err)
	}

	return &tenant.UpdateResponse{Tenant: s.modelToProto(&t)}, nil
}

func (s *Tenant) List(ctx context.Context, req *tenant.ListRequest) (*tenant.ListResponse, error) {
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	var total int64
	if err := s.db.Model(&models.Tenant{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tenants: %v", err)
	}

	var tenants []*models.Tenant
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&tenants).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tenants: %v", err)
	}

	out := make([]*tenant.Tenant, len(tenants))
	for i, t := range tenants {
		out[i] = s.modelToProto(t)
	}

	return &tenant.ListResponse{Tenants: out, Total: int32(total)}, nil
}

func (s *Tenant) usersConnStr(tenantName string) string {
	return s.connStrWithSuffix(tenantName, "_users")
}

func (s *Tenant) deployConnStr(tenantName string) string {
	return s.connStrWithSuffix(tenantName, "_deploy")
}

func (s *Tenant) accountConnStr(tenantName string) string {
	return s.connStrWithSuffix(tenantName, "_account")
}

func (s *Tenant) connStrWithSuffix(tenantName, suffix string) string {
	u, err := url.Parse(s.cfg.Database)
	if err != nil {
		return s.cfg.Database
	}
	u.Path = "/" + sanitizeDatabaseName(tenantName) + suffix
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
	sqlDB, err := gosql.Open("postgres", s.adminDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer sqlDB.Close()

	safeName := sanitizeDatabaseName(dbName)
	_, err = sqlDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, safeName))
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("database %s already exists", safeName)
		}
		return fmt.Errorf("failed to create database: %w", err)
	}
	return nil
}

func (s *Tenant) runMigrationsOn(connStr string, migrate func(*gorm.DB) error) error {
	db, err := infra.NewConnection(connStr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	return migrate(db)
}

func (s *Tenant) seedTenantOwner(connStr string, owner *models.User) error {
	sqldb := gosql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(connStr)))
	defer sqldb.Close()
	db := bun.NewDB(sqldb, pgdialect.New())
	u := &models.User{ID: owner.ID, Name: owner.Name}
	if _, err := db.NewInsert().Model(u).Exec(context.Background()); err != nil {
		return fmt.Errorf("failed to create owner user in tenant database: %w", err)
	}
	return nil
}

func (s *Tenant) modelToProto(m *models.Tenant) *tenant.Tenant {
	return &tenant.Tenant{
		Id:              m.ID.String(),
		Name:            m.Name,
		Database: m.UsersDatabase,
		Deploy:   m.DeployDatabase,
		Account:  m.AccountDatabase,
		Storage:         m.Storage,
		OwnerId:         m.OwnerID.String(),
		CreatedAt:       m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
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
