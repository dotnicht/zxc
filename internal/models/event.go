package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID            int64           `gorm:"primaryKey;autoIncrement"`
	Kind          string          `gorm:"type:text;not null;index"`
	AggregateType string          `gorm:"type:text;not null;index"`
	AggregateID   string          `gorm:"type:text;not null;index"`
	TenantID      *uuid.UUID      `gorm:"type:uuid;index"`
	Payload       json.RawMessage `gorm:"type:jsonb;not null"`
	CreatedAt     time.Time       `gorm:"not null;default:now();index"`
}

func (Event) TableName() string {
	return "events"
}
