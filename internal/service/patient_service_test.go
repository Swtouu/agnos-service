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
	err                error
}

func (m *mockPatientRepo) Search(ctx context.Context, hospitalID uuid.UUID, filters model.PatientSearchFilters) ([]model.Patient, error) {
	m.capturedHospitalID = hospitalID
	m.capturedFilters = filters
	return m.rows, m.err
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
	_, err := svc.Search(context.Background(), hospitalID, PatientSearchInput{NationalID: "1234567890123"})
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

	_, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{FirstName: "Somchai"})
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
	if _, err := svc.Search(context.Background(), hospitalID, PatientSearchInput{}); err != nil {
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
	}
	svc := NewPatientService(repo, cryptor)

	results, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].NationalID != "1234567890123" {
		t.Errorf("got decrypted national_id %q, want 1234567890123", results[0].NationalID)
	}
}

func TestPatientSearch_NoResults_EmptySliceNoError(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{rows: nil}
	svc := NewPatientService(repo, cryptor)

	results, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{FirstName: "Nobody"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestPatientSearch_RepositoryError(t *testing.T) {
	cryptor := testCryptor(t)
	repo := &mockPatientRepo{err: errors.New("db exploded")}
	svc := NewPatientService(repo, cryptor)

	_, err := svc.Search(context.Background(), uuid.New(), PatientSearchInput{})
	if err == nil {
		t.Error("expected error to propagate from repository")
	}
}
