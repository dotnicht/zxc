package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Request struct {
	ID        uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReleaseID uuid.UUID       `gorm:"type:uuid;not null"`
	Release   *Release        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Data      json.RawMessage `gorm:"type:jsonb;not null"`
	CreatedAt time.Time       `gorm:"not null;default:now()"`
	UpdatedAt time.Time       `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt  `gorm:"index"`
}

func (Request) TableName() string {
	return "requests"
}
