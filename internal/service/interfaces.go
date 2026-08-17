package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/watt-siwat/agnos-backend/internal/model"
)

// HospitalRepository, StaffRepository, PatientRepository, RefreshTokenRepository
// are implemented by internal/repository (GORM) and satisfied by hand-written
// mocks in service tests — this file is the seam unit tests mock against.

type HospitalRepository interface {
	FindByCode(ctx context.Context, code string) (*model.Hospital, error)
}

type StaffRepository interface {
	Create(ctx context.Context, staff *model.Staff) error
	FindByHospitalAndUsername(ctx context.Context, hospitalID uuid.UUID, username string) (*model.Staff, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Staff, error)
}

type PatientRepository interface {
	Search(ctx context.Context, hospitalID uuid.UUID, filters model.PatientSearchFilters) ([]model.Patient, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	DeleteByHash(ctx context.Context, tokenHash string) error
}
