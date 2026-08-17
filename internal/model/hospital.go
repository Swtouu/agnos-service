package model

import (
	"time"

	"github.com/google/uuid"
)

type Hospital struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey"`
	Code      string    `gorm:"column:code"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Hospital) TableName() string { return "hospitals" }
