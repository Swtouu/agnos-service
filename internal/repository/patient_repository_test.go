package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/model"
)

func seedHospital(t *testing.T, code string) model.Hospital {
	t.Helper()
	h := model.Hospital{ID: uuid.New(), Code: code, Name: code, CreatedAt: time.Now()}
	if err := testDB.Create(&h).Error; err != nil {
		t.Fatalf("seed hospital: %v", err)
	}
	return h
}

func seedPatient(t *testing.T, p model.Patient) model.Patient {
	t.Helper()
	if p.ID == (uuid.UUID{}) {
		p.ID = uuid.New()
	}
	if p.DateOfBirth.IsZero() {
		p.DateOfBirth = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if p.Gender == "" {
		p.Gender = "M"
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	if err := testDB.Create(&p).Error; err != nil {
		t.Fatalf("seed patient: %v", err)
	}
	return p
}

func TestPatientRepository_Search_ScopesToHospitalID(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospitalA := seedHospital(t, "hosp_a")
	hospitalB := seedHospital(t, "hosp_b")
	seedPatient(t, model.Patient{HospitalID: hospitalA.ID, FirstNameEN: "Somchai", PatientHN: "A-1"})
	seedPatient(t, model.Patient{HospitalID: hospitalB.ID, FirstNameEN: "Somchai", PatientHN: "B-1"})

	results, err := repo.Search(context.Background(), hospitalA.ID, model.PatientSearchFilters{FirstName: "Somchai"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want exactly the 1 patient in hospitalA", len(results))
	}
	if results[0].PatientHN != "A-1" {
		t.Errorf("got patient %q, want A-1 (hospitalA's patient, not hospitalB's)", results[0].PatientHN)
	}
}

func TestPatientRepository_Search_ExactMatchOnNationalIDHash(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospital := seedHospital(t, "hosp_a")
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "target", NationalIDHash: "hash-abc"})
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "decoy", NationalIDHash: "hash-abcd"}) // similar but not equal

	results, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{NationalIDHash: "hash-abc"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].PatientHN != "target" {
		t.Fatalf("expected exactly the exact-hash match, got %+v", results)
	}
}

func TestPatientRepository_Search_ExactMatchOnPassportIDHash(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospital := seedHospital(t, "hosp_a")
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "target", PassportIDHash: "pp-hash-1"})
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "other", PassportIDHash: "pp-hash-2"})

	results, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{PassportIDHash: "pp-hash-1"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].PatientHN != "target" {
		t.Fatalf("expected exactly the exact-hash match, got %+v", results)
	}
}

func TestPatientRepository_Search_PartialMatchAcrossThaiAndEnglishNames(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospital := seedHospital(t, "hosp_a")
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "th-match", FirstNameTH: "สมชาย", FirstNameEN: "Somchai"})
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "unrelated", FirstNameTH: "อนงค์", FirstNameEN: "Anong"})

	// Search by an English substring must also find a patient whose Thai name is what's stored differently —
	// here we confirm a partial EN substring finds the EN-matching patient, and a Thai substring finds it too.
	byEN, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{FirstName: "chai"})
	if err != nil {
		t.Fatalf("search by EN substring: %v", err)
	}
	if len(byEN) != 1 || byEN[0].PatientHN != "th-match" {
		t.Fatalf("EN substring search: got %+v, want th-match", byEN)
	}

	byTH, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{FirstName: "สมชาย"})
	if err != nil {
		t.Fatalf("search by TH substring: %v", err)
	}
	if len(byTH) != 1 || byTH[0].PatientHN != "th-match" {
		t.Fatalf("TH substring search: got %+v, want th-match", byTH)
	}
}

func TestPatientRepository_Search_PartialMatchIsCaseInsensitive(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospital := seedHospital(t, "hosp_a")
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "target", FirstNameEN: "Somchai"})

	results, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{FirstName: "SOMCHAI"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ILIKE should be case-insensitive: got %d results searching \"SOMCHAI\" for \"Somchai\"", len(results))
	}
}

func TestPatientRepository_Search_ANDCombinesMultipleFilters(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospital := seedHospital(t, "hosp_a")
	dob := time.Date(1985, 3, 12, 0, 0, 0, 0, time.UTC)
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "both-match", FirstNameEN: "Somchai", DateOfBirth: dob})
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "name-only", FirstNameEN: "Somchai", DateOfBirth: time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)})

	results, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{FirstName: "Somchai", DateOfBirth: &dob})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].PatientHN != "both-match" {
		t.Fatalf("AND of first_name+date_of_birth: got %+v, want only both-match", results)
	}
}

func TestPatientRepository_Search_ExactMatchDateOfBirth(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospital := seedHospital(t, "hosp_a")
	dob := time.Date(1985, 3, 12, 0, 0, 0, 0, time.UTC)
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "match", DateOfBirth: dob})
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "no-match", DateOfBirth: dob.AddDate(0, 0, 1)})

	results, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{DateOfBirth: &dob})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].PatientHN != "match" {
		t.Fatalf("got %+v, want only exact DOB match", results)
	}
}

func TestPatientRepository_Search_PhoneAndEmailExactMatch(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospital := seedHospital(t, "hosp_a")
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "match", PhoneNumber: "0812345671", Email: "a@example.com"})
	seedPatient(t, model.Patient{HospitalID: hospital.ID, PatientHN: "no-match", PhoneNumber: "0812345672", Email: "b@example.com"})

	byPhone, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{PhoneNumber: "0812345671"})
	if err != nil {
		t.Fatalf("search by phone: %v", err)
	}
	if len(byPhone) != 1 || byPhone[0].PatientHN != "match" {
		t.Fatalf("phone exact match: got %+v", byPhone)
	}

	byEmail, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{Email: "a@example.com"})
	if err != nil {
		t.Fatalf("search by email: %v", err)
	}
	if len(byEmail) != 1 || byEmail[0].PatientHN != "match" {
		t.Fatalf("email exact match: got %+v", byEmail)
	}
}

func TestPatientRepository_Search_NoFilters_ReturnsAllInHospital(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)

	hospitalA := seedHospital(t, "hosp_a")
	hospitalB := seedHospital(t, "hosp_b")
	seedPatient(t, model.Patient{HospitalID: hospitalA.ID, PatientHN: "a-1"})
	seedPatient(t, model.Patient{HospitalID: hospitalA.ID, PatientHN: "a-2"})
	seedPatient(t, model.Patient{HospitalID: hospitalB.ID, PatientHN: "b-1"})

	results, err := repo.Search(context.Background(), hospitalA.ID, model.PatientSearchFilters{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("no filters should return all of hospitalA's patients: got %d, want 2", len(results))
	}
}
