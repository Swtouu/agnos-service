package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/watt-siwat/agnos-backend/internal/model"
)

type PatientRepository struct {
	db *gorm.DB
}

func NewPatientRepository(db *gorm.DB) *PatientRepository {
	return &PatientRepository{db: db}
}

// Search always filters by hospitalID first — this is where tenant isolation
// is enforced at the SQL level. Every other filter is AND-ed on top of it.
func (r *PatientRepository) Search(ctx context.Context, hospitalID uuid.UUID, f model.PatientSearchFilters) ([]model.Patient, error) {
	q := r.db.WithContext(ctx).Where("hospital_id = ?", hospitalID)

	if f.NationalIDHash != "" {
		q = q.Where("national_id_hash = ?", f.NationalIDHash)
	}
	if f.PassportIDHash != "" {
		q = q.Where("passport_id_hash = ?", f.PassportIDHash)
	}
	if f.FirstName != "" {
		like := "%" + f.FirstName + "%"
		q = q.Where("(first_name_th ILIKE ? OR first_name_en ILIKE ?)", like, like)
	}
	if f.MiddleName != "" {
		like := "%" + f.MiddleName + "%"
		q = q.Where("(middle_name_th ILIKE ? OR middle_name_en ILIKE ?)", like, like)
	}
	if f.LastName != "" {
		like := "%" + f.LastName + "%"
		q = q.Where("(last_name_th ILIKE ? OR last_name_en ILIKE ?)", like, like)
	}
	if f.PhoneNumber != "" {
		q = q.Where("phone_number = ?", f.PhoneNumber)
	}
	if f.Email != "" {
		q = q.Where("email = ?", f.Email)
	}
	if f.DateOfBirth != nil {
		q = q.Where("date_of_birth = ?", *f.DateOfBirth)
	}

	var rows []model.Patient
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
