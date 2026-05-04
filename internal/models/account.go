package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ProfileUnknown  = "unknown"
	ProfileActive   = "active"
	ProfileDisabled = "disabled"
)

type Profile struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string         `gorm:"type:varchar(255);not null;uniqueIndex"`
	Status    string         `gorm:"type:varchar(20);not null;default:'unknown'"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Profile) TableName() string {
	return "profiles"
}
