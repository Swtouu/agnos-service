package service

import "errors"

var (
	ErrHospitalNotFound    = errors.New("hospital not found")
	ErrHospitalMismatch    = errors.New("hospital does not match caller's hospital")
	ErrUsernameTaken       = errors.New("username already taken in this hospital")
	ErrInvalidCredentials  = errors.New("invalid username, password, or hospital")
	ErrInvalidRefreshToken = errors.New("invalid, expired, or revoked refresh token")
)
