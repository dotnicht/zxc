package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	tenantapi "zxc/api/tenant"
	workerapi "zxc/api/worker"
	"zxc/internal/authz"
	"zxc/internal/models"
)

type Worker struct {
	workerapi.UnimplementedWorkerServiceServer
	db *gorm.DB
}

func NewWorker(db *gorm.DB) *Worker {
	return &Worker{db: db}
}

func (s *Worker) Create(ctx context.Context, req *workerapi.CreateRequest) (*workerapi.CreateResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if _, err := authorize(ctx, "worker.create", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}

	record := &models.Worker{Name: req.Name}
	if req.Id != "" {
		id, err := parseID(req.Id, "id")
		if err != nil {
			return nil, err
		}
		record.ID = id
	}

	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "worker with this id or name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create worker: %v", err)
	}
	return &workerapi.CreateResponse{Worker: workerToProto(record)}, nil
}

func (s *Worker) Get(ctx context.Context, req *workerapi.GetRequest) (*workerapi.GetResponse, error) {
	if _, err := authorize(ctx, "worker.get", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	id, err := parseID(req.Id, "id")
	if err != nil {
		return nil, err
	}
	var record models.Worker
	if err := s.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "worker not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get worker: %v", err)
	}
	return &workerapi.GetResponse{Worker: workerToProto(&record)}, nil
}

func (s *Worker) Update(ctx context.Context, req *workerapi.UpdateRequest) (*workerapi.UpdateResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if _, err := authorize(ctx, "worker.update", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	id, err := parseID(req.Id, "id")
	if err != nil {
		return nil, err
	}
	var record models.Worker
	if err := s.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "worker not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get worker: %v", err)
	}
	record.Name = req.Name
	if err := s.db.WithContext(ctx).Save(&record).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "worker with this name already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to update worker: %v", err)
	}
	return &workerapi.UpdateResponse{Worker: workerToProto(&record)}, nil
}

func (s *Worker) Delete(ctx context.Context, req *workerapi.DeleteRequest) (*workerapi.DeleteResponse, error) {
	if _, err := authorize(ctx, "worker.delete", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	id, err := parseID(req.Id, "id")
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&models.Worker{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("worker_id = ?", id).Delete(&models.WorkerTenantAssignment{}).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "worker not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete worker: %v", err)
	}
	return &workerapi.DeleteResponse{Success: true}, nil
}

func (s *Worker) List(ctx context.Context, req *workerapi.ListRequest) (*workerapi.ListResponse, error) {
	if _, err := authorize(ctx, "worker.list", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	var total int64
	if err := s.db.WithContext(ctx).Model(&models.Worker{}).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count workers: %v", err)
	}
	var records []*models.Worker
	if err := s.db.WithContext(ctx).Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&records).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list workers: %v", err)
	}
	items := make([]*workerapi.Worker, len(records))
	for i, record := range records {
		items[i] = workerToProto(record)
	}
	return &workerapi.ListResponse{Workers: items, Total: int32(total)}, nil
}

func (s *Worker) Search(ctx context.Context, req *workerapi.SearchRequest) (*workerapi.SearchResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "search query is required")
	}
	if _, err := authorize(ctx, "worker.search", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	pattern := "%" + req.Query + "%"
	var total int64
	if err := s.db.WithContext(ctx).Model(&models.Worker{}).Where("name ILIKE ?", pattern).Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count workers: %v", err)
	}
	var records []*models.Worker
	if err := s.db.WithContext(ctx).Where("name ILIKE ?", pattern).Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&records).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search workers: %v", err)
	}
	items := make([]*workerapi.Worker, len(records))
	for i, record := range records {
		items[i] = workerToProto(record)
	}
	return &workerapi.SearchResponse{Workers: items, Total: int32(total)}, nil
}

func (s *Worker) AssignTenant(ctx context.Context, req *workerapi.AssignTenantRequest) (*workerapi.AssignTenantResponse, error) {
	if _, err := authorize(ctx, "worker.assign_tenant", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	workerID, tenantID, err := parseWorkerTenantIDs(req.WorkerId, req.TenantId)
	if err != nil {
		return nil, err
	}
	if err := s.ensureWorkerAndTenant(ctx, workerID, tenantID); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).FirstOrCreate(&models.WorkerTenantAssignment{}, models.WorkerTenantAssignment{
		WorkerID: workerID,
		TenantID: tenantID,
	}).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign tenant: %v", err)
	}
	return &workerapi.AssignTenantResponse{Success: true}, nil
}

func (s *Worker) UnassignTenant(ctx context.Context, req *workerapi.UnassignTenantRequest) (*workerapi.UnassignTenantResponse, error) {
	if _, err := authorize(ctx, "worker.unassign_tenant", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	workerID, tenantID, err := parseWorkerTenantIDs(req.WorkerId, req.TenantId)
	if err != nil {
		return nil, err
	}
	result := s.db.WithContext(ctx).Delete(&models.WorkerTenantAssignment{}, "worker_id = ? AND tenant_id = ?", workerID, tenantID)
	if result.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to unassign tenant: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "worker assignment not found")
	}
	return &workerapi.UnassignTenantResponse{Success: true}, nil
}

func (s *Worker) ListTenants(ctx context.Context, req *workerapi.ListTenantsRequest) (*workerapi.ListTenantsResponse, error) {
	if _, err := authorize(ctx, "worker.list_tenants", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	workerID, err := parseID(req.WorkerId, "worker_id")
	if err != nil {
		return nil, err
	}
	if err := s.ensureWorkerExists(ctx, workerID); err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	query := s.db.WithContext(ctx).Model(&models.Tenant{}).
		Joins("JOIN worker_tenant_assignments ON worker_tenant_assignments.tenant_id = tenants.id").
		Where("worker_tenant_assignments.worker_id = ?", workerID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count assigned tenants: %v", err)
	}
	var tenants []*models.Tenant
	if err := query.Order("tenants.created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&tenants).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list assigned tenants: %v", err)
	}
	items := make([]*tenantapi.Tenant, len(tenants))
	for i, tenant := range tenants {
		items[i] = tenantToProto(tenant)
	}
	return &workerapi.ListTenantsResponse{Tenants: items, Total: int32(total)}, nil
}

func (s *Worker) ListWorkersForTenant(ctx context.Context, req *workerapi.ListWorkersForTenantRequest) (*workerapi.ListWorkersForTenantResponse, error) {
	if _, err := authorize(ctx, "worker.list_workers_for_tenant", nil, authz.Resource{Type: "worker"}, authz.Related{}); err != nil {
		return nil, err
	}
	tenantID, err := parseID(req.TenantId, "tenant_id")
	if err != nil {
		return nil, err
	}
	if err := s.ensureTenantExists(ctx, tenantID); err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	query := s.db.WithContext(ctx).Model(&models.Worker{}).
		Joins("JOIN worker_tenant_assignments ON worker_tenant_assignments.worker_id = workers.id").
		Where("worker_tenant_assignments.tenant_id = ?", tenantID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count workers for tenant: %v", err)
	}
	var workers []*models.Worker
	if err := query.Order("workers.created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&workers).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list workers for tenant: %v", err)
	}
	items := make([]*workerapi.Worker, len(workers))
	for i, worker := range workers {
		items[i] = workerToProto(worker)
	}
	return &workerapi.ListWorkersForTenantResponse{Workers: items, Total: int32(total)}, nil
}

func parseWorkerTenantIDs(workerRaw, tenantRaw string) (uuid.UUID, uuid.UUID, error) {
	workerID, err := parseID(workerRaw, "worker_id")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	tenantID, err := parseID(tenantRaw, "tenant_id")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return workerID, tenantID, nil
}

func (s *Worker) ensureWorkerAndTenant(ctx context.Context, workerID, tenantID uuid.UUID) error {
	if err := s.ensureWorkerExists(ctx, workerID); err != nil {
		return err
	}
	return s.ensureTenantExists(ctx, tenantID)
}

func (s *Worker) ensureWorkerExists(ctx context.Context, id uuid.UUID) error {
	var record models.Worker
	if err := s.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "worker not found")
		}
		return status.Errorf(codes.Internal, "failed to load worker: %v", err)
	}
	return nil
}

func (s *Worker) ensureTenantExists(ctx context.Context, id uuid.UUID) error {
	var record models.Tenant
	if err := s.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "tenant not found")
		}
		return status.Errorf(codes.Internal, "failed to load tenant: %v", err)
	}
	return nil
}

func normalizePage(page, pageSize int32) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return int(page), int(pageSize)
}

func workerToProto(record *models.Worker) *workerapi.Worker {
	return &workerapi.Worker{
		Id:        record.ID.String(),
		Name:      record.Name,
		CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func tenantToProto(record *models.Tenant) *tenantapi.Tenant {
	return &tenantapi.Tenant{
		Id:        record.ID.String(),
		Name:      record.Name,
		Database:  record.Database,
		Storage:   record.Storage,
		CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		OwnerId:   record.OwnerID.String(),
	}
}
