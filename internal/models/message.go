package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TalkID    uuid.UUID      `gorm:"type:uuid;not null"`
	ProfileID uuid.UUID      `gorm:"type:uuid;not null"`
	Text      string         `gorm:"type:text;not null;default:''"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Message) TableName() string {
	return "messages"
}
