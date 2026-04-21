package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AccountUnknown  = "unknown"
	AccountActive   = "active"
	AccountDisabled = "disabled"
)

type Account struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string         `gorm:"type:varchar(255);not null;uniqueIndex"`
	Status    string         `gorm:"type:varchar(20);not null;default:'unknown'"`
	Sessions  []*Session     `gorm:"foreignKey:AccountID"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Account) TableName() string {
	return "accounts"
}
