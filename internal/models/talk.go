package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Talk struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Talk) TableName() string {
	return "talks"
}
