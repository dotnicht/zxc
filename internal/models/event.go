package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID            int64           `gorm:"primaryKey;autoIncrement"`
	Kind          string          `gorm:"type:text;not null;index"`
	AggregateType string          `gorm:"type:text;not null;index;index:events_aggregate_idx,priority:1"`
	AggregateID   uuid.UUID       `gorm:"type:uuid;not null;index;index:events_aggregate_idx,priority:2"`
	Payload       json.RawMessage `gorm:"type:jsonb;not null"`
	CreatedAt     time.Time       `gorm:"not null;default:now();index;index:events_aggregate_idx,priority:3,sort:desc"`
}

func (Event) TableName() string {
	return "events"
}
