package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/watt-siwat/agnos-backend/internal/model"
)

type HospitalRepository struct {
	db *gorm.DB
}

func NewHospitalRepository(db *gorm.DB) *HospitalRepository {
	return &HospitalRepository{db: db}
}

func (r *HospitalRepository) FindByCode(ctx context.Context, code string) (*model.Hospital, error) {
	var h model.Hospital
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&h).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}
