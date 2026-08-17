package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/watt-siwat/agnos-backend/internal/model"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) DeleteByHash(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&model.RefreshToken{}).Error
}
