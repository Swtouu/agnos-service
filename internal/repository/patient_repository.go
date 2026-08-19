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
// Returns the page of matching rows plus the total match count (pre-paging),
// so callers can render "N of M" / next-page-exists without a second request.
func (r *PatientRepository) Search(ctx context.Context, hospitalID uuid.UUID, f model.PatientSearchFilters) ([]model.Patient, int64, error) {
	var total int64
	if err := r.filtered(ctx, hospitalID, f).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ORDER BY is required for LIMIT/OFFSET to return stable, non-overlapping
	// pages — without it Postgres gives no guarantee about row order at all,
	// let alone a consistent one across two paginated calls.
	//
	// Limit is applied unconditionally, including when it's 0 — GORM's clause
	// builder treats Limit(0) as a literal "LIMIT 0" (zero rows), distinct
	// from omitting Limit() entirely (unbounded). The service layer guarantees
	// f.Limit is never negative by the time it reaches here.
	var rows []model.Patient
	q := r.filtered(ctx, hospitalID, f).Order("created_at, id").Limit(f.Limit).Offset(f.Offset)
	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// filtered builds a fresh query with every WHERE clause applied, no LIMIT/OFFSET/ORDER.
// Called separately for the Count and the Find so each gets its own *gorm.DB —
// reusing one chain across two terminal calls (Count then Find) is a known
// GORM pitfall where clauses can leak between them.
func (r *PatientRepository) filtered(ctx context.Context, hospitalID uuid.UUID, f model.PatientSearchFilters) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&model.Patient{}).Where("hospital_id = ?", hospitalID)

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
	return q
}
