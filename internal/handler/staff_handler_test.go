package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/watt-siwat/agnos-backend/internal/auth"
	"github.com/watt-siwat/agnos-backend/internal/middleware"
	"github.com/watt-siwat/agnos-backend/internal/model"
	"github.com/watt-siwat/agnos-backend/internal/service"
)

var jwtSecret = []byte("test-jwt-secret")

type fakeHospitalRepo struct {
	hospital *model.Hospital
}

func (f *fakeHospitalRepo) FindByCode(ctx context.Context, code string) (*model.Hospital, error) {
	if f.hospital != nil && f.hospital.Code == code {
		return f.hospital, nil
	}
	return nil, nil
}

type fakeStaffRepo struct {
	existing *model.Staff
}

func (f *fakeStaffRepo) Create(ctx context.Context, staff *model.Staff) error {
	f.existing = staff
	return nil
}

func (f *fakeStaffRepo) FindByHospitalAndUsername(ctx context.Context, hospitalID uuid.UUID, username string) (*model.Staff, error) {
	if f.existing != nil && f.existing.HospitalID == hospitalID && f.existing.Username == username {
		return f.existing, nil
	}
	return nil, nil
}

func (f *fakeStaffRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Staff, error) {
	if f.existing != nil && f.existing.ID == id {
		return f.existing, nil
	}
	return nil, nil
}

type fakeRefreshTokenRepo struct {
	stored map[string]*model.RefreshToken
}

func newFakeRefreshTokenRepo() *fakeRefreshTokenRepo {
	return &fakeRefreshTokenRepo{stored: make(map[string]*model.RefreshToken)}
}

func (f *fakeRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	f.stored[token.TokenHash] = token
	return nil
}

func (f *fakeRefreshTokenRepo) FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	return f.stored[tokenHash], nil
}

func (f *fakeRefreshTokenRepo) DeleteByHash(ctx context.Context, tokenHash string) error {
	delete(f.stored, tokenHash)
	return nil
}

func setupStaffRouter(t *testing.T) (*gin.Engine, *model.Hospital, *fakeStaffRepo, *fakeRefreshTokenRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	hospital := &model.Hospital{ID: uuid.New(), Code: "hospital_a", Name: "Hospital A"}
	hospitals := &fakeHospitalRepo{hospital: hospital}
	staffRepo := &fakeStaffRepo{}
	refreshTokenRepo := newFakeRefreshTokenRepo()
	authService := service.NewAuthService(hospitals, staffRepo, refreshTokenRepo, jwtSecret)
	h := NewStaffHandler(authService)

	r := gin.New()
	r.POST("/staff/login", h.Login)
	r.POST("/staff/refresh", h.Refresh)
	protected := r.Group("/")
	protected.Use(middleware.JWTAuth(jwtSecret))
	protected.POST("/staff/create", h.Create)
	protected.POST("/staff/logout", h.Logout)

	return r, hospital, staffRepo, refreshTokenRepo
}

func tokenFor(t *testing.T, staffID, hospitalID uuid.UUID) string {
	t.Helper()
	tok, err := auth.IssueAccessToken(jwtSecret, staffID, hospitalID, service.AccessTokenTTL)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return tok
}

func TestStaffCreate_Success(t *testing.T) {
	r, hospital, _, _ := setupStaffRouter(t)
	token := tokenFor(t, uuid.New(), hospital.ID)

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "password123", "hospital": "hospital_a"})
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestStaffCreate_NoAuth_Unauthorized(t *testing.T) {
	r, _, _, _ := setupStaffRouter(t)

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "password123", "hospital": "hospital_a"})
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestStaffCreate_HospitalMismatch_Forbidden(t *testing.T) {
	r, _, _, _ := setupStaffRouter(t)
	// token belongs to a different hospital than the one in the request body
	token := tokenFor(t, uuid.New(), uuid.New())

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "password123", "hospital": "hospital_a"})
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want %d, body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestStaffLogin_Success(t *testing.T) {
	r, hospital, staffRepo, refreshTokenRepo := setupStaffRouter(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	staffRepo.existing = &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice", PasswordHash: string(hash)}

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "password123", "hospital": "hospital_a"})
	req := httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["access_token"] == "" || resp["refresh_token"] == "" {
		t.Error("expected both access_token and refresh_token in response")
	}
	if len(refreshTokenRepo.stored) != 1 {
		t.Errorf("expected 1 refresh token stored, got %d", len(refreshTokenRepo.stored))
	}
}

func TestStaffLogin_WrongPassword_Unauthorized(t *testing.T) {
	r, hospital, staffRepo, _ := setupStaffRouter(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	staffRepo.existing = &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice", PasswordHash: string(hash)}

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "wrong-password", "hospital": "hospital_a"})
	req := httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestStaffRefresh_Success(t *testing.T) {
	r, hospital, staffRepo, refreshTokenRepo := setupStaffRouter(t)
	staffID := uuid.New()
	staffRepo.existing = &model.Staff{ID: staffID, HospitalID: hospital.ID, Username: "alice"}

	rawToken, _ := auth.GenerateRefreshToken()
	hash := auth.HashRefreshToken(rawToken)
	refreshTokenRepo.stored[hash] = &model.RefreshToken{ID: uuid.New(), StaffID: staffID, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)}

	body, _ := json.Marshal(map[string]string{"refresh_token": rawToken})
	req := httptest.NewRequest(http.MethodPost, "/staff/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestStaffRefresh_InvalidToken_Unauthorized(t *testing.T) {
	r, _, _, _ := setupStaffRouter(t)

	body, _ := json.Marshal(map[string]string{"refresh_token": "not-a-real-token"})
	req := httptest.NewRequest(http.MethodPost, "/staff/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestStaffLogout_Success(t *testing.T) {
	r, hospital, _, refreshTokenRepo := setupStaffRouter(t)
	staffID := uuid.New()

	rawToken, _ := auth.GenerateRefreshToken()
	hash := auth.HashRefreshToken(rawToken)
	refreshTokenRepo.stored[hash] = &model.RefreshToken{ID: uuid.New(), StaffID: staffID, TokenHash: hash}

	accessToken := tokenFor(t, staffID, hospital.ID)
	body, _ := json.Marshal(map[string]string{"refresh_token": rawToken})
	req := httptest.NewRequest(http.MethodPost, "/staff/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("got status %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
	if _, exists := refreshTokenRepo.stored[hash]; exists {
		t.Error("expected refresh token to be deleted after logout")
	}
}

func TestStaffLogout_NoAuth_Unauthorized(t *testing.T) {
	r, _, _, _ := setupStaffRouter(t)

	body, _ := json.Marshal(map[string]string{"refresh_token": "some-token"})
	req := httptest.NewRequest(http.MethodPost, "/staff/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
