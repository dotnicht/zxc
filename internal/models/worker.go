package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Worker struct {
	ID          uuid.UUID                 `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string                    `gorm:"type:varchar(255);not null;uniqueIndex"`
	Assignments []*WorkerTenantAssignment `gorm:"foreignKey:WorkerID"`
	CreatedAt   time.Time                 `gorm:"not null;default:now()"`
	UpdatedAt   time.Time                 `gorm:"not null;default:now()"`
	DeletedAt   gorm.DeletedAt            `gorm:"index"`
}

func (Worker) TableName() string {
	return "workers"
}

type WorkerTenantAssignment struct {
	WorkerID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	Worker    *Worker   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Tenant    *Tenant   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (WorkerTenantAssignment) TableName() string {
	return "worker_tenant_assignments"
}
