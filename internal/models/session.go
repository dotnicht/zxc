package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SessionOnline  = "online"
	SessionOffline = "offline"
	SessionSync    = "sync"
)

type Session struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AccountID uuid.UUID      `gorm:"type:uuid;not null;index"`
	Account   *Account       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Status    string         `gorm:"type:varchar(20);not null;default:'offline'"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Session) TableName() string {
	return "sessions"
}
