package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/crypto"
	appmiddleware "github.com/watt-siwat/agnos-backend/internal/middleware"
	"github.com/watt-siwat/agnos-backend/internal/model"
	"github.com/watt-siwat/agnos-backend/internal/service"
)

type fakePatientRepo struct {
	byHospital map[uuid.UUID][]model.Patient
}

// Search applies filters.Limit/Offset over the stored slice — enough to test
// that the handler/service correctly parse and pass pagination params
// through. Real LIMIT/OFFSET/ORDER BY SQL correctness is verified separately
// against a real Postgres instance in internal/repository's own tests.
func (f *fakePatientRepo) Search(ctx context.Context, hospitalID uuid.UUID, filters model.PatientSearchFilters) ([]model.Patient, int64, error) {
	all := f.byHospital[hospitalID]
	total := int64(len(all))

	start := filters.Offset
	if start > len(all) {
		start = len(all)
	}
	end := start + filters.Limit
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, nil
}

func testCryptor(t *testing.T) *crypto.Cryptor {
	t.Helper()
	c, err := crypto.New([]byte("01234567890123456789012345678901"[:32]), []byte("test-hmac-secret"))
	if err != nil {
		t.Fatalf("crypto.New: %v", err)
	}
	return c
}

func setupPatientRouter(t *testing.T, repo *fakePatientRepo) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	patientService := service.NewPatientService(repo, testCryptor(t))
	h := NewPatientHandler(patientService)

	r := gin.New()
	protected := r.Group("/")
	protected.Use(appmiddleware.JWTAuth(jwtSecret))
	protected.GET("/patient/search", h.Search)
	return r
}

func doSearch(t *testing.T, r *gin.Engine, token, query string) (int, service.PatientSearchResult) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/patient/search"+query, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var result service.PatientSearchResult
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal response: %v, body: %s", err, w.Body.String())
		}
	}
	return w.Code, result
}

func TestPatientSearch_ScopedToCallerHospital(t *testing.T) {
	hospitalAID := uuid.New()
	hospitalBID := uuid.New()
	repo := &fakePatientRepo{byHospital: map[uuid.UUID][]model.Patient{
		hospitalAID: {{ID: uuid.New(), FirstNameEN: "Somchai", DateOfBirth: mustDate(t, "1990-01-01")}},
		hospitalBID: {{ID: uuid.New(), FirstNameEN: "Somchai", DateOfBirth: mustDate(t, "1990-01-01")}},
	}}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), hospitalAID)

	code, result := doSearch(t, r, token, "")
	if code != http.StatusOK {
		t.Fatalf("got status %d, want %d", code, http.StatusOK)
	}
	if len(result.Patients) != 1 {
		t.Fatalf("got %d results, want exactly the 1 patient from the caller's own hospital", len(result.Patients))
	}
}

func TestPatientSearch_NoAuth_Unauthorized(t *testing.T) {
	repo := &fakePatientRepo{}
	r := setupPatientRouter(t, repo)

	req := httptest.NewRequest(http.MethodGet, "/patient/search", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestPatientSearch_InvalidDateOfBirth_BadRequest(t *testing.T) {
	repo := &fakePatientRepo{}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), uuid.New())

	code, _ := doSearch(t, r, token, "?date_of_birth=not-a-date")
	if code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", code, http.StatusBadRequest)
	}
}

func TestPatientSearch_DefaultLimit_AppliedWhenUnspecified(t *testing.T) {
	hospitalID := uuid.New()
	patients := make([]model.Patient, 30)
	for i := range patients {
		patients[i] = model.Patient{ID: uuid.New(), DateOfBirth: mustDate(t, "1990-01-01")}
	}
	repo := &fakePatientRepo{byHospital: map[uuid.UUID][]model.Patient{hospitalID: patients}}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), hospitalID)

	code, result := doSearch(t, r, token, "")
	if code != http.StatusOK {
		t.Fatalf("got status %d, want %d", code, http.StatusOK)
	}
	if len(result.Patients) != service.DefaultSearchLimit {
		t.Errorf("got %d patients, want default limit %d", len(result.Patients), service.DefaultSearchLimit)
	}
	if result.Total != 30 {
		t.Errorf("got total %d, want 30", result.Total)
	}
}

func TestPatientSearch_ExplicitLimitAndOffset_Respected(t *testing.T) {
	hospitalID := uuid.New()
	patients := make([]model.Patient, 10)
	for i := range patients {
		patients[i] = model.Patient{ID: uuid.New(), PatientHN: string(rune('A' + i)), DateOfBirth: mustDate(t, "1990-01-01")}
	}
	repo := &fakePatientRepo{byHospital: map[uuid.UUID][]model.Patient{hospitalID: patients}}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), hospitalID)

	code, result := doSearch(t, r, token, "?limit=3&offset=5")
	if code != http.StatusOK {
		t.Fatalf("got status %d, want %d", code, http.StatusOK)
	}
	if len(result.Patients) != 3 {
		t.Fatalf("got %d patients, want 3", len(result.Patients))
	}
	if result.Limit != 3 || result.Offset != 5 {
		t.Errorf("got limit=%d offset=%d, want limit=3 offset=5", result.Limit, result.Offset)
	}
	if result.Total != 10 {
		t.Errorf("got total %d, want 10", result.Total)
	}
}

func TestPatientSearch_LimitAboveMax_IsCapped(t *testing.T) {
	repo := &fakePatientRepo{}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), uuid.New())

	code, result := doSearch(t, r, token, "?limit=99999")
	if code != http.StatusOK {
		t.Fatalf("got status %d, want %d", code, http.StatusOK)
	}
	if result.Limit != service.MaxSearchLimit {
		t.Errorf("got limit %d, want capped at %d", result.Limit, service.MaxSearchLimit)
	}
}

func TestPatientSearch_NegativeLimit_BadRequest(t *testing.T) {
	repo := &fakePatientRepo{}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), uuid.New())

	code, _ := doSearch(t, r, token, "?limit=-1")
	if code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", code, http.StatusBadRequest)
	}
}

func TestPatientSearch_NonNumericLimit_BadRequest(t *testing.T) {
	repo := &fakePatientRepo{}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), uuid.New())

	code, _ := doSearch(t, r, token, "?limit=abc")
	if code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", code, http.StatusBadRequest)
	}
}

func TestPatientSearch_NegativeOffset_BadRequest(t *testing.T) {
	repo := &fakePatientRepo{}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), uuid.New())

	code, _ := doSearch(t, r, token, "?offset=-1")
	if code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", code, http.StatusBadRequest)
	}
}

func mustDate(t *testing.T, s string) (d time.Time) {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	return d
}
