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

func (f *fakePatientRepo) Search(ctx context.Context, hospitalID uuid.UUID, filters model.PatientSearchFilters) ([]model.Patient, error) {
	return f.byHospital[hospitalID], nil
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

func TestPatientSearch_ScopedToCallerHospital(t *testing.T) {
	hospitalAID := uuid.New()
	hospitalBID := uuid.New()
	repo := &fakePatientRepo{byHospital: map[uuid.UUID][]model.Patient{
		hospitalAID: {{ID: uuid.New(), FirstNameEN: "Somchai", DateOfBirth: mustDate(t, "1990-01-01")}},
		hospitalBID: {{ID: uuid.New(), FirstNameEN: "Somchai", DateOfBirth: mustDate(t, "1990-01-01")}},
	}}
	r := setupPatientRouter(t, repo)
	token := tokenFor(t, uuid.New(), hospitalAID)

	req := httptest.NewRequest(http.MethodGet, "/patient/search", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var results []service.PatientDTO
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want exactly the 1 patient from the caller's own hospital", len(results))
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

	req := httptest.NewRequest(http.MethodGet, "/patient/search?date_of_birth=not-a-date", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
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
