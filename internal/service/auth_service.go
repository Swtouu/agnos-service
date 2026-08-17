package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/watt-siwat/agnos-backend/internal/auth"
	"github.com/watt-siwat/agnos-backend/internal/model"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

type AuthService struct {
	hospitals     HospitalRepository
	staff         StaffRepository
	refreshTokens RefreshTokenRepository
	jwtSecret     []byte
}

func NewAuthService(hospitals HospitalRepository, staff StaffRepository, refreshTokens RefreshTokenRepository, jwtSecret []byte) *AuthService {
	return &AuthService{hospitals: hospitals, staff: staff, refreshTokens: refreshTokens, jwtSecret: jwtSecret}
}

// CreateStaff creates a new staff account. callerHospitalID is the hospital of
// the already-authenticated staff making the request; the target hospitalCode
// must resolve to that same hospital, or the request is rejected.
func (s *AuthService) CreateStaff(ctx context.Context, callerHospitalID uuid.UUID, username, password, hospitalCode string) (*model.Staff, error) {
	hospital, err := s.hospitals.FindByCode(ctx, hospitalCode)
	if err != nil {
		return nil, err
	}
	if hospital == nil {
		return nil, ErrHospitalNotFound
	}
	if hospital.ID != callerHospitalID {
		return nil, ErrHospitalMismatch
	}

	existing, err := s.staff.FindByHospitalAndUsername(ctx, hospital.ID, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newStaff := &model.Staff{
		ID:           uuid.New(),
		HospitalID:   hospital.ID,
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
	if err := s.staff.Create(ctx, newStaff); err != nil {
		// A concurrent /staff/create for the same (hospital, username) can race
		// past the FindByHospitalAndUsername check above; the DB's unique
		// constraint is the real guard, translated back to the same 400 here.
		if errors.Is(err, model.ErrDuplicateKey) {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}
	return newStaff, nil
}

// Login verifies credentials and issues a short-lived access token plus a
// long-lived refresh token (bonus, beyond the assignment's stated spec).
func (s *AuthService) Login(ctx context.Context, username, password, hospitalCode string) (accessToken, refreshToken string, err error) {
	hospital, err := s.hospitals.FindByCode(ctx, hospitalCode)
	if err != nil {
		return "", "", err
	}
	if hospital == nil {
		return "", "", ErrInvalidCredentials
	}

	staffRow, err := s.staff.FindByHospitalAndUsername(ctx, hospital.ID, username)
	if err != nil {
		return "", "", err
	}
	if staffRow == nil {
		return "", "", ErrInvalidCredentials
	}

	if bcrypt.CompareHashAndPassword([]byte(staffRow.PasswordHash), []byte(password)) != nil {
		return "", "", ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, staffRow)
}

// Refresh rotates a refresh token: the presented token is deleted and a new
// access+refresh pair is issued. No reuse-detection cascade — presenting an
// already-rotated token simply fails as "invalid" (kept deliberately simple).
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (accessToken, newRefreshToken string, err error) {
	hash := auth.HashRefreshToken(rawRefreshToken)
	row, err := s.refreshTokens.FindByHash(ctx, hash)
	if err != nil {
		return "", "", err
	}
	if row == nil || row.RevokedAt != nil || row.ExpiresAt.Before(time.Now()) {
		return "", "", ErrInvalidRefreshToken
	}

	staffRow, err := s.staff.FindByID(ctx, row.StaffID)
	if err != nil {
		return "", "", err
	}
	if staffRow == nil {
		return "", "", ErrInvalidRefreshToken
	}

	if err := s.refreshTokens.DeleteByHash(ctx, hash); err != nil {
		return "", "", err
	}

	return s.issueTokenPair(ctx, staffRow)
}

// Logout deletes the presented refresh token so it can no longer be used,
// even if it hasn't naturally expired yet.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := auth.HashRefreshToken(rawRefreshToken)
	return s.refreshTokens.DeleteByHash(ctx, hash)
}

func (s *AuthService) issueTokenPair(ctx context.Context, staffRow *model.Staff) (accessToken, refreshToken string, err error) {
	accessToken, err = auth.IssueAccessToken(s.jwtSecret, staffRow.ID, staffRow.HospitalID, AccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	rawRefreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}
	if err := s.refreshTokens.Create(ctx, &model.RefreshToken{
		ID:        uuid.New(),
		StaffID:   staffRow.ID,
		TokenHash: auth.HashRefreshToken(rawRefreshToken),
		ExpiresAt: time.Now().Add(RefreshTokenTTL),
		CreatedAt: time.Now(),
	}); err != nil {
		return "", "", err
	}

	return accessToken, rawRefreshToken, nil
}
