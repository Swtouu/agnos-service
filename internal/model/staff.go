package model

import (
	"time"

	"github.com/google/uuid"
)

type Staff struct {
	ID           uuid.UUID `gorm:"column:id;primaryKey"`
	HospitalID   uuid.UUID `gorm:"column:hospital_id"`
	Username     string    `gorm:"column:username"`
	PasswordHash string    `gorm:"column:password_hash"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (Staff) TableName() string { return "staff" }
