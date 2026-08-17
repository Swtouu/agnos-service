package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/middleware"
	"github.com/watt-siwat/agnos-backend/internal/service"
)

type PatientHandler struct {
	patients *service.PatientService
}

func NewPatientHandler(patients *service.PatientService) *PatientHandler {
	return &PatientHandler{patients: patients}
}

// Search godoc
//
//	@Summary		Search patients
//	@Description	All filters are optional and AND-ed together. Identifiers/phone/email/DOB are exact match; name fields are partial match across Thai and English columns. Results are always scoped to the caller's own hospital.
//	@Tags			patient
//	@Produce		json
//	@Security		BearerAuth
//	@Param			national_id		query		string	false	"exact match"
//	@Param			passport_id		query		string	false	"exact match"
//	@Param			first_name		query		string	false	"partial match, TH or EN"
//	@Param			middle_name		query		string	false	"partial match, TH or EN"
//	@Param			last_name		query		string	false	"partial match, TH or EN"
//	@Param			date_of_birth	query		string	false	"YYYY-MM-DD, exact match"
//	@Param			phone_number	query		string	false	"exact match"
//	@Param			email			query		string	false	"exact match"
//	@Success		200				{array}		service.PatientDTO
//	@Failure		400				{object}	errorResponse	"invalid date_of_birth format"
//	@Failure		401				{object}	errorResponse	"missing or invalid token"
//	@Router			/patient/search [get]
func (h *PatientHandler) Search(c *gin.Context) {
	hospitalID := c.MustGet(middleware.ContextKeyHospitalID).(uuid.UUID)

	input := service.PatientSearchInput{
		NationalID:  c.Query("national_id"),
		PassportID:  c.Query("passport_id"),
		FirstName:   c.Query("first_name"),
		MiddleName:  c.Query("middle_name"),
		LastName:    c.Query("last_name"),
		PhoneNumber: c.Query("phone_number"),
		Email:       c.Query("email"),
	}

	if dobStr := c.Query("date_of_birth"); dobStr != "" {
		dob, err := time.Parse("2006-01-02", dobStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date_of_birth must be in YYYY-MM-DD format"})
			return
		}
		input.DateOfBirth = &dob
	}

	results, err := h.patients.Search(c.Request.Context(), hospitalID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, results)
}
