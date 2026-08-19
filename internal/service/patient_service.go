package service

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/crypto"
	"github.com/watt-siwat/agnos-backend/internal/model"
)

const (
	// DefaultSearchLimit applies when the caller doesn't specify a limit.
	DefaultSearchLimit = 20
	// MaxSearchLimit caps every request, specified or not — prevents a
	// single query from dumping an entire hospital's patient table.
	MaxSearchLimit = 100
)

// PatientSearchInput mirrors the request-side optional filters, using plaintext
// national_id/passport_id — the service converts these to blind-index hashes
// before they ever reach the repository layer.
type PatientSearchInput struct {
	NationalID  string
	PassportID  string
	FirstName   string
	MiddleName  string
	LastName    string
	PhoneNumber string
	Email       string
	DateOfBirth *time.Time
	// Limit < 0 means "not specified" — DefaultSearchLimit applies. Offset < 0
	// is clamped to 0.
	Limit  int
	Offset int
}

// PatientSearchResult is the paginated response envelope.
type PatientSearchResult struct {
	Patients []PatientDTO `json:"patients"`
	Total    int64        `json:"total"`
	Limit    int          `json:"limit"`
	Offset   int          `json:"offset"`
}

type PatientDTO struct {
	ID           uuid.UUID `json:"id"`
	FirstNameTH  string    `json:"first_name_th"`
	MiddleNameTH string    `json:"middle_name_th"`
	LastNameTH   string    `json:"last_name_th"`
	FirstNameEN  string    `json:"first_name_en"`
	MiddleNameEN string    `json:"middle_name_en"`
	LastNameEN   string    `json:"last_name_en"`
	DateOfBirth  string    `json:"date_of_birth"`
	PatientHN    string    `json:"patient_hn"`
	NationalID   string    `json:"national_id,omitempty"`
	PassportID   string    `json:"passport_id,omitempty"`
	PhoneNumber  string    `json:"phone_number"`
	Email        string    `json:"email"`
	Gender       string    `json:"gender"`
}

type PatientService struct {
	patients PatientRepository
	cryptor  *crypto.Cryptor
}

func NewPatientService(patients PatientRepository, cryptor *crypto.Cryptor) *PatientService {
	return &PatientService{patients: patients, cryptor: cryptor}
}

// Search scopes every query to hospitalID — derived from the caller's JWT by
// the handler, never taken from client input.
func (s *PatientService) Search(ctx context.Context, hospitalID uuid.UUID, input PatientSearchInput) (PatientSearchResult, error) {
	limit := input.Limit
	if limit < 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	filters := model.PatientSearchFilters{
		FirstName:   input.FirstName,
		MiddleName:  input.MiddleName,
		LastName:    input.LastName,
		PhoneNumber: input.PhoneNumber,
		Email:       input.Email,
		DateOfBirth: input.DateOfBirth,
		Limit:       limit,
		Offset:      offset,
	}
	if input.NationalID != "" {
		filters.NationalIDHash = s.cryptor.Hash(input.NationalID)
	}
	if input.PassportID != "" {
		filters.PassportIDHash = s.cryptor.Hash(input.PassportID)
	}

	rows, total, err := s.patients.Search(ctx, hospitalID, filters)
	if err != nil {
		return PatientSearchResult{}, err
	}

	dtos := make([]PatientDTO, 0, len(rows))
	for _, p := range rows {
		dto := PatientDTO{
			ID:           p.ID,
			FirstNameTH:  p.FirstNameTH,
			MiddleNameTH: p.MiddleNameTH,
			LastNameTH:   p.LastNameTH,
			FirstNameEN:  p.FirstNameEN,
			MiddleNameEN: p.MiddleNameEN,
			LastNameEN:   p.LastNameEN,
			DateOfBirth:  p.DateOfBirth.Format("2006-01-02"),
			PatientHN:    p.PatientHN,
			PhoneNumber:  p.PhoneNumber,
			Email:        p.Email,
			Gender:       p.Gender,
		}
		if p.NationalIDEncrypted != "" {
			if plain, err := s.cryptor.Decrypt(p.NationalIDEncrypted); err == nil {
				dto.NationalID = plain
			} else {
				log.Printf("decrypt national_id for patient %s: %v", p.ID, err)
			}
		}
		if p.PassportIDEncrypted != "" {
			if plain, err := s.cryptor.Decrypt(p.PassportIDEncrypted); err == nil {
				dto.PassportID = plain
			} else {
				log.Printf("decrypt passport_id for patient %s: %v", p.ID, err)
			}
		}
		dtos = append(dtos, dto)
	}
	return PatientSearchResult{Patients: dtos, Total: total, Limit: limit, Offset: offset}, nil
}
