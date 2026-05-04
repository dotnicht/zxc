package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payload struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Path      string         `gorm:"type:text;not null"`
	OwnerID   uuid.UUID      `gorm:"type:uuid;not null"`
	Config    string         `gorm:"type:text;not null;default:''"`
	Start     string         `gorm:"type:text;not null;default:''"`
	Stop      string         `gorm:"type:text;not null;default:''"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Payload) TableName() string {
	return "payloads"
}
