package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TargetUnknown = "unknown"
	TargetOnline  = "online"
	TargetOffline = "offline"
)

type Target struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Address     string         `gorm:"type:text;not null"`
	User        string         `gorm:"type:text;not null;default:''"`
	Key         string         `gorm:"type:text;not null;default:''"`
	Status      string         `gorm:"type:varchar(20);not null;default:'unknown'"`
	Deploying   bool           `gorm:"not null;default:false"`
	DeployingAt *time.Time     `gorm:"index"`
	OwnerID     uuid.UUID      `gorm:"type:uuid;not null"`
	CreatedAt   time.Time      `gorm:"not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Target) TableName() string {
	return "targets"
}
