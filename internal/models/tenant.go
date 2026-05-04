package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tenant struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name            string         `gorm:"type:varchar(255);not null;uniqueIndex"`
	OwnerID         uuid.UUID      `gorm:"type:uuid;not null"`
	Owner           *User          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	MainDatabase    string         `gorm:"type:text"`
	DeployDatabase  string         `gorm:"type:text"`
	AccountDatabase string         `gorm:"type:text"`
	Storage         string         `gorm:"type:text"`
	CreatedAt       time.Time      `gorm:"not null;default:now()"`
	UpdatedAt       time.Time      `gorm:"not null;default:now()"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (Tenant) TableName() string {
	return "tenants"
}
