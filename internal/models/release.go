package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ReleaseUnknown  = "unknown"
	ReleaseWait     = "wait"
	ReleaseDeployed = "deployed"
	ReleaseDead     = "dead"
)

type Release struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Status      string         `gorm:"type:varchar(20);not null;default:'unknown'"`
	OwnerID     uuid.UUID      `gorm:"type:uuid;not null"`
	Target      *Target        `gorm:"-"`
	TargetID    *uuid.UUID     `gorm:"type:uuid"`
	Payload     *Payload       `gorm:"-"`
	PayloadID   *uuid.UUID     `gorm:"type:uuid"`
	ChangedByID uuid.UUID      `gorm:"type:uuid;not null"`
	CreatedAt   time.Time      `gorm:"not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Release) TableName() string {
	return "releases"
}
