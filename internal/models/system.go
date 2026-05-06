package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type System struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string         `gorm:"type:varchar(255);not null"`
	Sync      string         `gorm:"type:varchar(255);not null;default:'generator'"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (System) TableName() string {
	return "systems"
}
