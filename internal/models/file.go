package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TalkID    uuid.UUID      `gorm:"type:uuid;not null"`
	ProfileID uuid.UUID      `gorm:"type:uuid;not null"`
	Name      string         `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (File) TableName() string {
	return "files"
}
