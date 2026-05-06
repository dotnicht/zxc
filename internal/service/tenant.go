package service

import (
	"context"
	gosql "database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/lib/pq"
	wfclient "github.com/cschleiden/go-workflows/client"
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
	"zxc/internal/jobs"
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

func (s *Tenant) validate(name string) error {
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
	if err := s.validate(req.Name); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant name: %v", err)
	}

	var countN int64
	if err := s.db.Model(&models.Tenant{}).Where("name = ? AND deleted_at IS NULL", req.Name).Count(&countN).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check tenant existence: %v", err)
	}
	if countN > 0 {
		return nil, status.Error(codes.AlreadyExists, "tenant with this name already exists")
	}

	main := s.main(req.Name)
	deploy := s.deploy(req.Name)
	account := s.account(req.Name)
	jobs_ := s.jobs(req.Name)

	if req.Database != "" {
		main = req.Database
	}
	if req.Deploy != "" {
		deploy = req.Deploy
	}
	if req.Account != "" {
		account = req.Account
	}
	if req.Jobs != "" {
		jobs_ = req.Jobs
	}

	storage := req.Storage
	if storage == "" && s.cfg.Storage != "" {
		sc, bucket, err := infra.Storage(s.cfg.Storage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse storage config: %v", err)
		}
		if err := sc.CreateFolder(ctx, bucket, req.Name); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create tenant storage folder: %v", err)
		}
		storage = strings.TrimRight(s.cfg.Storage, "/") + "/" + req.Name
	}

	t := &models.Tenant{
		Name:    req.Name,
		Main:    main,
		Deploy:  deploy,
		Account: account,
		Jobs:    jobs_,
		Storage: storage,
		OwnerID: s.root.ID,
	}

	if err := s.db.Create(t).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "tenant with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create tenant: %v", err)
	}

	dbName := sanitize(req.Name)
	if req.Database == "" {
		if err := s.create(dbName); err != nil {
			s.db.Delete(&models.Tenant{}, "id = ?", t.ID)
			return nil, status.Errorf(codes.Internal, "failed to create tenant database: %v", err)
		}
	}
	if req.Jobs == "" {
		if err := s.create(dbName + "_jobs"); err != nil {
			s.db.Delete(&models.Tenant{}, "id = ?", t.ID)
			return nil, status.Errorf(codes.Internal, "failed to create tenant jobs database: %v", err)
		}
	}

	for _, m := range []struct {
		conn   string
		schema string
		fn     func(infra.Migrator) error
		label  string
	}{
		{main, "main", func(mg infra.Migrator) error { return mg.Main() }, "main"},
		{deploy, "deploy", func(mg infra.Migrator) error { return mg.Deploy() }, "deploy"},
		{account, "account", func(mg infra.Migrator) error { return mg.Account() }, "account"},
	} {
		if err := s.migrate(m.conn, m.schema, m.fn); err != nil {
			s.db.Delete(&models.Tenant{}, "id = ?", t.ID)
			return nil, status.Errorf(codes.Internal, "failed to run %s migrations: %v", m.label, err)
		}
	}

	// Initialise the jobs backend (runs migrations on first call, then cached).
	wfb, err := infra.Backend(jobs_)
	if err != nil {
		s.db.Delete(&models.Tenant{}, "id = ?", t.ID)
		return nil, status.Errorf(codes.Internal, "failed to initialise jobs backend: %v", err)
	}

	if err := s.seed(main, s.root); err != nil {
		s.db.Delete(&models.Tenant{}, "id = ?", t.ID)
		return nil, status.Errorf(codes.Internal, "failed to seed tenant owner: %v", err)
	}

	wfc := wfclient.New(wfb)
	_, _ = wfc.CreateWorkflowInstance(ctx,
		wfclient.WorkflowInstanceOptions{InstanceID: "generate:" + t.ID.String()},
		jobs.Generate, jobs.GenerateArgs{TenantID: t.ID},
	)

	return &tenant.CreateResponse{Tenant: s.proto(t)}, nil
}

func (s *Tenant) Get(ctx context.Context, req *tenant.GetRequest) (*tenant.GetResponse, error) {
	id := uuid.UUID(req.Id)

	var t models.Tenant
	if err := s.db.First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "tenant not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get tenant: %v", err)
	}

	return &tenant.GetResponse{Tenant: s.proto(&t)}, nil
}

func (s *Tenant) List(ctx context.Context, req *tenant.ListRequest) (*tenant.ListResponse, error) {
	page := int(req.Page)
	size := int(req.PageSize)
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	var total int64
	if err := s.db.Model(&models.Tenant{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tenants: %v", err)
	}

	var tenants []*models.Tenant
	offset := (page - 1) * size
	if err := s.db.Order("created_at DESC").Limit(size).Offset(offset).Find(&tenants).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tenants: %v", err)
	}

	out := make([]*tenant.Tenant, len(tenants))
	for i, t := range tenants {
		out[i] = s.proto(t)
	}

	return &tenant.ListResponse{Tenants: out, Total: int32(total)}, nil
}

func (s *Tenant) main(name string) string {
	return s.conn(name, "main")
}

func (s *Tenant) deploy(name string) string {
	return s.conn(name, "deploy")
}

func (s *Tenant) account(name string) string {
	return s.conn(name, "account")
}

func (s *Tenant) jobs(name string) string {
	u, err := url.Parse(s.cfg.Database)
	if err != nil {
		return s.cfg.Database
	}
	u.Path = "/" + sanitize(name) + "_jobs"
	u.RawQuery = ""
	return u.String()
}

func (s *Tenant) conn(tenantName, schema string) string {
	u, err := url.Parse(s.cfg.Database)
	if err != nil {
		return s.cfg.Database
	}
	u.Path = "/" + sanitize(tenantName)
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *Tenant) admin() string {
	u, err := url.Parse(s.cfg.Database)
	if err != nil {
		return s.cfg.Database
	}
	u.Path = "/postgres"
	return u.String()
}

func (s *Tenant) create(dbName string) error {
	sqlDB, err := gosql.Open("postgres", s.admin())
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer sqlDB.Close()

	name := sanitize(dbName)
	_, err = sqlDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, name))
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("database %s already exists", name)
		}
		return fmt.Errorf("failed to create database: %w", err)
	}
	return nil
}

func (s *Tenant) migrate(conn, schema string, fn func(infra.Migrator) error) error {
	db, err := infra.Connect(conn)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	if err := infra.Schema(db, schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return fn(infra.Migrator{DB: db})
}

func (s *Tenant) seed(conn string, owner *models.User) error {
	sqldb := gosql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(conn)))
	defer sqldb.Close()
	db := bun.NewDB(sqldb, pgdialect.New())
	u := &models.User{ID: owner.ID, Name: owner.Name}
	if _, err := db.NewInsert().Model(u).Exec(context.Background()); err != nil {
		return fmt.Errorf("failed to create owner user in tenant database: %w", err)
	}
	return nil
}

func (s *Tenant) proto(m *models.Tenant) *tenant.Tenant {
	return &tenant.Tenant{
		Id:        m.ID[:],
		Name:      m.Name,
		Database:  m.Main,
		Deploy:    m.Deploy,
		Account:   m.Account,
		Jobs:      m.Jobs,
		Storage:   m.Storage,
		OwnerId:   m.OwnerID[:],
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func sanitize(name string) string {
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
