package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/model"
)

// testLimit is a generously large limit for tests that aren't specifically
// about pagination — without an explicit Limit, model.PatientSearchFilters'
// zero value (0) means "LIMIT 0" (zero rows), by design (see patient_repository.go).
const testLimit = 1000

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

	results, _, err := repo.Search(context.Background(), hospitalA.ID, model.PatientSearchFilters{FirstName: "Somchai", Limit: testLimit})
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

	results, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{NationalIDHash: "hash-abc", Limit: testLimit})
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

	results, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{PassportIDHash: "pp-hash-1", Limit: testLimit})
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
	byEN, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{FirstName: "chai", Limit: testLimit})
	if err != nil {
		t.Fatalf("search by EN substring: %v", err)
	}
	if len(byEN) != 1 || byEN[0].PatientHN != "th-match" {
		t.Fatalf("EN substring search: got %+v, want th-match", byEN)
	}

	byTH, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{FirstName: "สมชาย", Limit: testLimit})
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

	results, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{FirstName: "SOMCHAI", Limit: testLimit})
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

	results, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{FirstName: "Somchai", DateOfBirth: &dob, Limit: testLimit})
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

	results, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{DateOfBirth: &dob, Limit: testLimit})
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

	byPhone, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{PhoneNumber: "0812345671", Limit: testLimit})
	if err != nil {
		t.Fatalf("search by phone: %v", err)
	}
	if len(byPhone) != 1 || byPhone[0].PatientHN != "match" {
		t.Fatalf("phone exact match: got %+v", byPhone)
	}

	byEmail, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{Email: "a@example.com", Limit: testLimit})
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

	results, _, err := repo.Search(context.Background(), hospitalA.ID, model.PatientSearchFilters{Limit: testLimit})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("no filters should return all of hospitalA's patients: got %d, want 2", len(results))
	}
}

// --- Pagination ---

func seedNPatients(t *testing.T, hospitalID uuid.UUID, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		seedPatient(t, model.Patient{HospitalID: hospitalID, PatientHN: string(rune('A' + i))})
	}
}

func TestPatientRepository_Search_LimitRestrictsPageSize(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)
	hospital := seedHospital(t, "hosp_a")
	seedNPatients(t, hospital.ID, 10)

	results, total, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{Limit: 3})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d rows, want 3 (limit)", len(results))
	}
	if total != 10 {
		t.Errorf("got total %d, want 10 (unaffected by limit)", total)
	}
}

func TestPatientRepository_Search_ZeroLimit_ReturnsNoRowsButRealTotal(t *testing.T) {
	// limit=0 is a deliberate, valid request ("just give me the total") — must
	// not be confused with "no limit specified" (which is the caller's job to
	// resolve before calling the repository; see service.PatientSearchInput).
	truncateAll(t)
	repo := NewPatientRepository(testDB)
	hospital := seedHospital(t, "hosp_a")
	seedNPatients(t, hospital.ID, 5)

	results, total, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{Limit: 0})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d rows, want 0 (LIMIT 0)", len(results))
	}
	if total != 5 {
		t.Errorf("got total %d, want 5", total)
	}
}

func TestPatientRepository_Search_OffsetSkipsRows(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)
	hospital := seedHospital(t, "hosp_a")
	seedNPatients(t, hospital.ID, 10)

	firstPage, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{Limit: 4, Offset: 0})
	if err != nil {
		t.Fatalf("search page 1: %v", err)
	}
	secondPage, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{Limit: 4, Offset: 4})
	if err != nil {
		t.Fatalf("search page 2: %v", err)
	}

	if len(firstPage) != 4 || len(secondPage) != 4 {
		t.Fatalf("got page sizes %d, %d, want 4, 4", len(firstPage), len(secondPage))
	}

	seen := map[string]bool{}
	for _, p := range firstPage {
		seen[p.PatientHN] = true
	}
	for _, p := range secondPage {
		if seen[p.PatientHN] {
			t.Errorf("patient_hn %q appeared in both page 1 and page 2 — pages overlap", p.PatientHN)
		}
	}
}

func TestPatientRepository_Search_PagesCoverAllRowsExactlyOnce(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)
	hospital := seedHospital(t, "hosp_a")
	seedNPatients(t, hospital.ID, 9)

	seen := map[uuid.UUID]int{}
	for offset := 0; offset < 9; offset += 3 {
		page, _, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{Limit: 3, Offset: offset})
		if err != nil {
			t.Fatalf("search offset=%d: %v", offset, err)
		}
		for _, p := range page {
			seen[p.ID]++
		}
	}
	if len(seen) != 9 {
		t.Fatalf("got %d distinct patients seen across all pages, want 9", len(seen))
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("patient %s seen %d times across pages, want exactly 1", id, count)
		}
	}
}

func TestPatientRepository_Search_OffsetBeyondTotal_ReturnsEmptyNotError(t *testing.T) {
	truncateAll(t)
	repo := NewPatientRepository(testDB)
	hospital := seedHospital(t, "hosp_a")
	seedNPatients(t, hospital.ID, 3)

	results, total, err := repo.Search(context.Background(), hospital.ID, model.PatientSearchFilters{Limit: 10, Offset: 1000})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d rows, want 0 (offset past end)", len(results))
	}
	if total != 3 {
		t.Errorf("got total %d, want 3 (total unaffected by offset)", total)
	}
}
