package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/watt-siwat/agnos-backend/internal/model"
)

// postgres unique_violation error code.
const pgUniqueViolation = "23505"

type StaffRepository struct {
	db *gorm.DB
}

func NewStaffRepository(db *gorm.DB) *StaffRepository {
	return &StaffRepository{db: db}
}

func (r *StaffRepository) Create(ctx context.Context, staff *model.Staff) error {
	err := r.db.WithContext(ctx).Create(staff).Error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return model.ErrDuplicateKey
	}
	return err
}

func (r *StaffRepository) FindByHospitalAndUsername(ctx context.Context, hospitalID uuid.UUID, username string) (*model.Staff, error) {
	var s model.Staff
	err := r.db.WithContext(ctx).
		Where("hospital_id = ? AND username = ?", hospitalID, username).
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *StaffRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Staff, error) {
	var s model.Staff
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
