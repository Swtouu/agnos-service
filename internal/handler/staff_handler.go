package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/middleware"
	"github.com/watt-siwat/agnos-backend/internal/service"
)

type StaffHandler struct {
	auth *service.AuthService
}

func NewStaffHandler(auth *service.AuthService) *StaffHandler {
	return &StaffHandler{auth: auth}
}

type createStaffRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required" example:"hospital_a"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Create godoc
//
//	@Summary		Create a new staff member
//	@Description	Requires auth; the target hospital must match the caller's own hospital.
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		createStaffRequest	true	"New staff details"
//	@Success		201		{object}	map[string]any
//	@Failure		400		{object}	errorResponse	"unknown hospital or username taken"
//	@Failure		401		{object}	errorResponse	"missing or invalid token"
//	@Failure		403		{object}	errorResponse	"hospital does not match caller's hospital"
//	@Router			/staff/create [post]
func (h *StaffHandler) Create(c *gin.Context) {
	var req createStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	callerHospitalID := c.MustGet(middleware.ContextKeyHospitalID).(uuid.UUID)

	newStaff, err := h.auth.CreateStaff(c.Request.Context(), callerHospitalID, req.Username, req.Password, req.Hospital)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{
			"id":          newStaff.ID,
			"username":    newStaff.Username,
			"hospital_id": newStaff.HospitalID,
			"created_at":  newStaff.CreatedAt,
		})
	case errors.Is(err, service.ErrHospitalNotFound), errors.Is(err, service.ErrUsernameTaken):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrHospitalMismatch):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required" example:"hospital_a"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Login godoc
//
//	@Summary		Staff login
//	@Description	Verifies credentials scoped to a hospital and issues an access + refresh token pair.
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Param			request	body		loginRequest	true	"Login credentials"
//	@Success		200		{object}	tokenPairResponse
//	@Failure		401		{object}	errorResponse	"invalid username, password, or hospital"
//	@Router			/staff/login [post]
func (h *StaffHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.auth.Login(c.Request.Context(), req.Username, req.Password, req.Hospital)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": refreshToken})
	case errors.Is(err, service.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh godoc
//
//	@Summary		Rotate an access/refresh token pair
//	@Description	Bonus endpoint, beyond the assignment's stated spec. Rotates: the presented refresh token is invalidated and a new pair is issued.
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Param			request	body		refreshRequest	true	"Refresh token"
//	@Success		200		{object}	tokenPairResponse
//	@Failure		401		{object}	errorResponse	"invalid, expired, or already-rotated refresh token"
//	@Router			/staff/refresh [post]
func (h *StaffHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, newRefreshToken, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": newRefreshToken})
	case errors.Is(err, service.ErrInvalidRefreshToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Logout godoc
//
//	@Summary		Revoke a refresh token
//	@Description	Bonus endpoint, beyond the assignment's stated spec. Deletes the presented refresh token so it cannot be used again.
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body	logoutRequest	true	"Refresh token to revoke"
//	@Success		204		"no content"
//	@Failure		401		{object}	errorResponse	"missing or invalid access token"
//	@Router			/staff/logout [post]
func (h *StaffHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}
