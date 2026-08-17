package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/watt-siwat/agnos-backend/internal/auth"
	"github.com/watt-siwat/agnos-backend/internal/model"
)

type mockHospitalRepo struct {
	hospital *model.Hospital
	err      error
}

func (m *mockHospitalRepo) FindByCode(ctx context.Context, code string) (*model.Hospital, error) {
	return m.hospital, m.err
}

type mockStaffRepo struct {
	existing     *model.Staff
	findErr      error
	createErr    error
	createdStaff *model.Staff
}

func (m *mockStaffRepo) Create(ctx context.Context, staff *model.Staff) error {
	m.createdStaff = staff
	return m.createErr
}

func (m *mockStaffRepo) FindByHospitalAndUsername(ctx context.Context, hospitalID uuid.UUID, username string) (*model.Staff, error) {
	return m.existing, m.findErr
}

func (m *mockStaffRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Staff, error) {
	if m.existing != nil && m.existing.ID == id {
		return m.existing, m.findErr
	}
	return nil, m.findErr
}

type mockRefreshTokenRepo struct {
	stored      map[string]*model.RefreshToken // by hash
	createErr   error
	findErr     error
	deleteErr   error
	deletedHash string
}

func newMockRefreshTokenRepo() *mockRefreshTokenRepo {
	return &mockRefreshTokenRepo{stored: make(map[string]*model.RefreshToken)}
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.stored[token.TokenHash] = token
	return nil
}

func (m *mockRefreshTokenRepo) FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.stored[tokenHash], nil
}

func (m *mockRefreshTokenRepo) DeleteByHash(ctx context.Context, tokenHash string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedHash = tokenHash
	delete(m.stored, tokenHash)
	return nil
}

func newTestHospital() *model.Hospital {
	return &model.Hospital{ID: uuid.New(), Code: "hospital_a", Name: "Hospital A"}
}

func TestCreateStaff_Success(t *testing.T) {
	hospital := newTestHospital()
	hospitals := &mockHospitalRepo{hospital: hospital}
	staff := &mockStaffRepo{existing: nil}
	svc := NewAuthService(hospitals, staff, newMockRefreshTokenRepo(), []byte("secret"))

	created, err := svc.CreateStaff(context.Background(), hospital.ID, "alice", "password123", "hospital_a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Username != "alice" {
		t.Errorf("got username %q, want alice", created.Username)
	}
	if created.HospitalID != hospital.ID {
		t.Errorf("got hospital_id %v, want %v", created.HospitalID, hospital.ID)
	}
	if staff.createdStaff == nil {
		t.Error("expected staff.Create to be called")
	}
}

func TestCreateStaff_HospitalNotFound(t *testing.T) {
	hospitals := &mockHospitalRepo{hospital: nil}
	staff := &mockStaffRepo{}
	svc := NewAuthService(hospitals, staff, newMockRefreshTokenRepo(), []byte("secret"))

	_, err := svc.CreateStaff(context.Background(), uuid.New(), "alice", "password123", "unknown_code")
	if !errors.Is(err, ErrHospitalNotFound) {
		t.Errorf("got err %v, want ErrHospitalNotFound", err)
	}
}

func TestCreateStaff_HospitalMismatch(t *testing.T) {
	hospital := newTestHospital()
	hospitals := &mockHospitalRepo{hospital: hospital}
	staff := &mockStaffRepo{}
	svc := NewAuthService(hospitals, staff, newMockRefreshTokenRepo(), []byte("secret"))

	callerHospitalID := uuid.New() // different from hospital.ID
	_, err := svc.CreateStaff(context.Background(), callerHospitalID, "alice", "password123", "hospital_a")
	if !errors.Is(err, ErrHospitalMismatch) {
		t.Errorf("got err %v, want ErrHospitalMismatch", err)
	}
}

func TestCreateStaff_UsernameTaken(t *testing.T) {
	hospital := newTestHospital()
	hospitals := &mockHospitalRepo{hospital: hospital}
	staff := &mockStaffRepo{existing: &model.Staff{ID: uuid.New(), Username: "alice"}}
	svc := NewAuthService(hospitals, staff, newMockRefreshTokenRepo(), []byte("secret"))

	_, err := svc.CreateStaff(context.Background(), hospital.ID, "alice", "password123", "hospital_a")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Errorf("got err %v, want ErrUsernameTaken", err)
	}
}

func TestCreateStaff_ConcurrentUsernameRace_ReturnsUsernameTaken(t *testing.T) {
	// FindByHospitalAndUsername found nothing (no race detected pre-check), but
	// the DB's unique constraint catches a concurrent insert of the same
	// (hospital, username) — repository surfaces this as model.ErrDuplicateKey.
	hospital := newTestHospital()
	hospitals := &mockHospitalRepo{hospital: hospital}
	staff := &mockStaffRepo{existing: nil, createErr: model.ErrDuplicateKey}
	svc := NewAuthService(hospitals, staff, newMockRefreshTokenRepo(), []byte("secret"))

	_, err := svc.CreateStaff(context.Background(), hospital.ID, "alice", "password123", "hospital_a")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Errorf("got err %v, want ErrUsernameTaken", err)
	}
}

func TestLogin_Success(t *testing.T) {
	hospital := newTestHospital()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	existing := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice", PasswordHash: string(hash)}

	hospitals := &mockHospitalRepo{hospital: hospital}
	staff := &mockStaffRepo{existing: existing}
	refreshTokens := newMockRefreshTokenRepo()
	svc := NewAuthService(hospitals, staff, refreshTokens, []byte("secret"))

	accessToken, refreshToken, err := svc.Login(context.Background(), "alice", "password123", "hospital_a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}
	if refreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if len(refreshTokens.stored) != 1 {
		t.Errorf("expected 1 refresh token stored, got %d", len(refreshTokens.stored))
	}
}

func TestLogin_HospitalNotFound(t *testing.T) {
	hospitals := &mockHospitalRepo{hospital: nil}
	staff := &mockStaffRepo{}
	svc := NewAuthService(hospitals, staff, newMockRefreshTokenRepo(), []byte("secret"))

	_, _, err := svc.Login(context.Background(), "alice", "password123", "unknown_code")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got err %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_StaffNotFound(t *testing.T) {
	hospital := newTestHospital()
	hospitals := &mockHospitalRepo{hospital: hospital}
	staff := &mockStaffRepo{existing: nil}
	svc := NewAuthService(hospitals, staff, newMockRefreshTokenRepo(), []byte("secret"))

	_, _, err := svc.Login(context.Background(), "ghost", "password123", "hospital_a")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got err %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hospital := newTestHospital()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	existing := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice", PasswordHash: string(hash)}

	hospitals := &mockHospitalRepo{hospital: hospital}
	staff := &mockStaffRepo{existing: existing}
	svc := NewAuthService(hospitals, staff, newMockRefreshTokenRepo(), []byte("secret"))

	_, _, err := svc.Login(context.Background(), "alice", "wrong-password", "hospital_a")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("got err %v, want ErrInvalidCredentials", err)
	}
}

func TestRefresh_Success_RotatesToken(t *testing.T) {
	hospital := newTestHospital()
	existingStaff := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice"}
	staff := &mockStaffRepo{existing: existingStaff}
	refreshTokens := newMockRefreshTokenRepo()
	svc := NewAuthService(&mockHospitalRepo{hospital: hospital}, staff, refreshTokens, []byte("secret"))

	rawToken, _ := auth.GenerateRefreshToken()
	oldHash := auth.HashRefreshToken(rawToken)
	refreshTokens.stored[oldHash] = &model.RefreshToken{
		ID: uuid.New(), StaffID: existingStaff.ID, TokenHash: oldHash, ExpiresAt: time.Now().Add(time.Hour),
	}

	newAccess, newRefresh, err := svc.Refresh(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Error("expected non-empty new access and refresh tokens")
	}
	if newRefresh == rawToken {
		t.Error("expected a different refresh token after rotation")
	}
	if _, stillExists := refreshTokens.stored[oldHash]; stillExists {
		t.Error("expected old refresh token to be deleted after rotation")
	}
}

func TestRefresh_UnknownToken_Invalid(t *testing.T) {
	svc := NewAuthService(&mockHospitalRepo{}, &mockStaffRepo{}, newMockRefreshTokenRepo(), []byte("secret"))

	rawToken, _ := auth.GenerateRefreshToken()
	_, _, err := svc.Refresh(context.Background(), rawToken)
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("got err %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRefresh_ExpiredToken_Invalid(t *testing.T) {
	staffID := uuid.New()
	staff := &mockStaffRepo{existing: &model.Staff{ID: staffID}}
	refreshTokens := newMockRefreshTokenRepo()
	svc := NewAuthService(&mockHospitalRepo{}, staff, refreshTokens, []byte("secret"))

	rawToken, _ := auth.GenerateRefreshToken()
	hash := auth.HashRefreshToken(rawToken)
	refreshTokens.stored[hash] = &model.RefreshToken{
		ID: uuid.New(), StaffID: staffID, TokenHash: hash, ExpiresAt: time.Now().Add(-time.Hour), // already expired
	}

	_, _, err := svc.Refresh(context.Background(), rawToken)
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("got err %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRefresh_ReuseAfterRotation_Invalid(t *testing.T) {
	// The first Refresh call rotates (deletes) the token; presenting the same
	// raw token a second time must fail, not silently succeed again.
	hospital := newTestHospital()
	staffID := uuid.New()
	staff := &mockStaffRepo{existing: &model.Staff{ID: staffID, HospitalID: hospital.ID}}
	refreshTokens := newMockRefreshTokenRepo()
	svc := NewAuthService(&mockHospitalRepo{hospital: hospital}, staff, refreshTokens, []byte("secret"))

	rawToken, _ := auth.GenerateRefreshToken()
	hash := auth.HashRefreshToken(rawToken)
	refreshTokens.stored[hash] = &model.RefreshToken{
		ID: uuid.New(), StaffID: staffID, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour),
	}

	if _, _, err := svc.Refresh(context.Background(), rawToken); err != nil {
		t.Fatalf("first refresh should succeed: %v", err)
	}
	if _, _, err := svc.Refresh(context.Background(), rawToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("reusing token after rotation: got err %v, want ErrInvalidRefreshToken", err)
	}
}

func TestLogout_DeletesToken(t *testing.T) {
	refreshTokens := newMockRefreshTokenRepo()
	svc := NewAuthService(&mockHospitalRepo{}, &mockStaffRepo{}, refreshTokens, []byte("secret"))

	rawToken, _ := auth.GenerateRefreshToken()
	hash := auth.HashRefreshToken(rawToken)
	refreshTokens.stored[hash] = &model.RefreshToken{ID: uuid.New(), TokenHash: hash}

	if err := svc.Logout(context.Background(), rawToken); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, stillExists := refreshTokens.stored[hash]; stillExists {
		t.Error("expected refresh token to be deleted after logout")
	}
}
