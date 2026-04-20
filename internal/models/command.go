package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	CommandPending = "pending"
	CommandRunning = "running"
	CommandDone    = "done"
	CommandFailed  = "failed"
)

type Command struct {
	ID            int64           `gorm:"primaryKey;autoIncrement"`
	Kind          string          `gorm:"type:text;not null;index"`
	AggregateType string          `gorm:"type:text;not null;index"`
	AggregateID   string          `gorm:"type:text;not null;index"`
	TenantID      *uuid.UUID      `gorm:"type:uuid;index"`
	Payload       json.RawMessage `gorm:"type:jsonb;not null"`
	Status        string          `gorm:"type:text;not null;default:'pending';index"`
	RunAt         time.Time       `gorm:"not null;default:now();index"`
	LeaseUntil    *time.Time      `gorm:"index"`
	Attempt       int             `gorm:"not null;default:0"`
	MaxAttempts   int             `gorm:"not null;default:5"`
	DedupeKey     string          `gorm:"type:text"`
	IsActive      bool            `gorm:"not null;default:true"`
	FinishedAt    *time.Time
	LastError     string    `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"not null;default:now();index"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

func (Command) TableName() string {
	return "commands"
}
