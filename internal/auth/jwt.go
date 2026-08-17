// Package auth issues and validates the JWT access tokens used across the
// middleware. Shared by service (issuance on login) and middleware (validation
// on protected routes) to avoid a service->middleware or middleware->service
// dependency in either direction.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type Claims struct {
	StaffID    uuid.UUID `json:"staff_id"`
	HospitalID uuid.UUID `json:"hospital_id"`
	jwt.RegisteredClaims
}

func IssueAccessToken(secret []byte, staffID, hospitalID uuid.UUID, ttl time.Duration) (string, error) {
	claims := Claims{
		StaffID:    staffID,
		HospitalID: hospitalID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ParseAccessToken(secret []byte, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
