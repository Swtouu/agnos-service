package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/crypto"
	"github.com/watt-siwat/agnos-backend/internal/model"
)

type mockPatientRepo struct {
	capturedHospitalID uuid.UUID
	capturedFilters    model.PatientSearchFilters
	rows               []model.Patient
	total              int64
	err                error
}

func (m *mockPatientRepo) Search(ctx context.Context, hospitalID uuid.UUID, filters model.PatientSearchFilters) ([]model.Patient, int64, error) {
	m.capturedHospitalID = hospitalID
	m.capturedFilters = filters
	return m.rows, m.total, m.err
}

func testCryptor(t *testing.T) *crypto.Cryptor {
	t.Helper()
	c, err := crypto.New([]byte("01234567890123456789012345678901"[:32]), []byte("test-hmac-secret"))
	if err != nil {
		t.Fatalf("crypto.New: %v", err)
	}
	return c
}

func TestPatientSearch_HashesIdentifierFilters(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{}
	svc := NewPatientService(repo, cryptor)

	hospitalID := uuid.New()
	_, err := svc.Search(context.Background(), hospitalID, PatientSearchInput{NationalID: "1234567890123", Limit: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := cryptor.Hash("1234567890123")
	if repo.capturedFilters.NationalIDHash != want {
		t.Errorf("got national_id_hash %q, want %q", repo.capturedFilters.NationalIDHash, want)
	}
}

func TestPatientSearch_NoIdentifierInput_NoHash(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{}
	svc := NewPatientService(repo, cryptor)

	_, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{FirstName: "Somchai", Limit: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedFilters.NationalIDHash != "" {
		t.Errorf("expected empty national_id_hash when no national_id given, got %q", repo.capturedFilters.NationalIDHash)
	}
	if repo.capturedFilters.FirstName != "Somchai" {
		t.Errorf("expected FirstName passed through unmodified, got %q", repo.capturedFilters.FirstName)
	}
}

func TestPatientSearch_ScopesToHospitalID(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{}
	svc := NewPatientService(repo, cryptor)

	hospitalID := uuid.New()
	if _, err := svc.Search(context.Background(), hospitalID, PatientSearchInput{Limit: -1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedHospitalID != hospitalID {
		t.Errorf("got hospitalID %v, want %v", repo.capturedHospitalID, hospitalID)
	}
}

func TestPatientSearch_DecryptsIdentifiersForDisplay(t *testing.T) {
	cryptor := testCryptor(t)
	encryptedNationalID, err := cryptor.Encrypt("1234567890123")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	repo := &mockPatientRepo{
		rows: []model.Patient{
			{
				ID:                  uuid.New(),
				NationalIDEncrypted: encryptedNationalID,
				DateOfBirth:         time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		total: 1,
	}
	svc := NewPatientService(repo, cryptor)

	result, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{Limit: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Patients) != 1 {
		t.Fatalf("got %d results, want 1", len(result.Patients))
	}
	if result.Patients[0].NationalID != "1234567890123" {
		t.Errorf("got decrypted national_id %q, want 1234567890123", result.Patients[0].NationalID)
	}
}

func TestPatientSearch_NoResults_EmptySliceNoError(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{rows: nil}
	svc := NewPatientService(repo, cryptor)

	result, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{FirstName: "Nobody", Limit: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Patients) != 0 {
		t.Errorf("got %d results, want 0", len(result.Patients))
	}
}

func TestPatientSearch_RepositoryError(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{err: errors.New("db exploded")}
	svc := NewPatientService(repo, cryptor)

	_, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{Limit: -1})
	if err == nil {
		t.Error("expected error to propagate from repository")
	}
}

func TestPatientSearch_UnspecifiedLimit_AppliesDefault(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{}
	svc := NewPatientService(repo, cryptor)

	result, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{Limit: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedFilters.Limit != DefaultSearchLimit {
		t.Errorf("got repo limit %d, want DefaultSearchLimit (%d)", repo.capturedFilters.Limit, DefaultSearchLimit)
	}
	if result.Limit != DefaultSearchLimit {
		t.Errorf("got result.Limit %d, want %d", result.Limit, DefaultSearchLimit)
	}
}

func TestPatientSearch_LimitAboveMax_IsCapped(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{}
	svc := NewPatientService(repo, cryptor)

	result, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{Limit: 9999})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedFilters.Limit != MaxSearchLimit {
		t.Errorf("got repo limit %d, want MaxSearchLimit (%d)", repo.capturedFilters.Limit, MaxSearchLimit)
	}
	if result.Limit != MaxSearchLimit {
		t.Errorf("got result.Limit %d, want %d", result.Limit, MaxSearchLimit)
	}
}

func TestPatientSearch_ExplicitZeroLimit_PassedThroughUncapped(t *testing.T) {
	// limit=0 is a deliberate, valid request (e.g. "just give me the total
	// count, no rows") — distinct from "unspecified" (Limit: -1).
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{total: 42}
	svc := NewPatientService(repo, cryptor)

	result, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{Limit: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedFilters.Limit != 0 {
		t.Errorf("got repo limit %d, want 0", repo.capturedFilters.Limit)
	}
	if result.Total != 42 {
		t.Errorf("got total %d, want 42", result.Total)
	}
}

func TestPatientSearch_NegativeOffset_ClampedToZero(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{}
	svc := NewPatientService(repo, cryptor)

	result, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{Limit: -1, Offset: -5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedFilters.Offset != 0 {
		t.Errorf("got repo offset %d, want 0", repo.capturedFilters.Offset)
	}
	if result.Offset != 0 {
		t.Errorf("got result.Offset %d, want 0", result.Offset)
	}
}

func TestPatientSearch_OffsetPassedThrough(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{}
	svc := NewPatientService(repo, cryptor)

	result, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{Limit: -1, Offset: 40})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.capturedFilters.Offset != 40 {
		t.Errorf("got repo offset %d, want 40", repo.capturedFilters.Offset)
	}
	if result.Offset != 40 {
		t.Errorf("got result.Offset %d, want 40", result.Offset)
	}
}

func TestPatientSearch_TotalReflectsRepositoryCount(t *testing.T) {
	// Total must reflect the full match count, independent of how many rows
	// this particular page returned (e.g. 3 rows on this page, 57 total matches).
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{
		rows: []model.Patient{
			{ID: uuid.New(), DateOfBirth: time.Now()},
			{ID: uuid.New(), DateOfBirth: time.Now()},
			{ID: uuid.New(), DateOfBirth: time.Now()},
		},
		total: 57,
	}
	svc := NewPatientService(repo, cryptor)

	result, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{Limit: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Patients) != 3 {
		t.Errorf("got %d patients on this page, want 3", len(result.Patients))
	}
	if result.Total != 57 {
		t.Errorf("got total %d, want 57", result.Total)
	}
}
