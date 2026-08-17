package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken stores only a hash of the raw refresh token, never the token itself.
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"column:id;primaryKey"`
	StaffID   uuid.UUID  `gorm:"column:staff_id"`
	TokenHash string     `gorm:"column:token_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
