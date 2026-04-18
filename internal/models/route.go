package models

import "github.com/google/uuid"

type Route struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key"`
	TenantID uuid.UUID `gorm:"type:uuid;not null"`
	Tenant   *Tenant   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (Route) TableName() string {
	return "targets"
}
