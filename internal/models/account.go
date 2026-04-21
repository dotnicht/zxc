package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string         `gorm:"type:varchar(255);not null;uniqueIndex:accounts_name_active_idx,where:deleted_at IS NULL"`
	Requests  []*Request     `gorm:"foreignKey:AccountID"`
	Sessions  []*Session     `gorm:"foreignKey:AccountID"`
	CreatedAt time.Time      `gorm:"not null;default:now()"`
	UpdatedAt time.Time      `gorm:"not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Account) TableName() string {
	return "accounts"
}
