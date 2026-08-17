// Package mockhis simulates the external Hospital A HIS API described in task 1
// of the assignment. It is intentionally standalone: hardcoded in-memory data,
// no DB, not called by the middleware's own /patient/search at request time.
package mockhis

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type patientRecord struct {
	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`
	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`
	DateOfBirth  string `json:"date_of_birth"`
	PatientHN    string `json:"patient_hn"`
	NationalID   string `json:"national_id"`
	PassportID   string `json:"passport_id"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
}

var fixtures = []patientRecord{
	{
		FirstNameTH: "สมชาย", LastNameTH: "ใจดี",
		FirstNameEN: "Somchai", LastNameEN: "Jaidee",
		DateOfBirth: "1985-03-12", PatientHN: "HIS-HA-0001",
		NationalID: "1100000000001", PhoneNumber: "0812345671",
		Email: "somchai.a@example.com", Gender: "M",
	},
	{
		FirstNameTH: "สุดา", LastNameTH: "ศรีสุข",
		FirstNameEN: "Suda", LastNameEN: "Srisuk",
		DateOfBirth: "1990-07-21", PatientHN: "HIS-HA-0002",
		NationalID: "1100000000002", PhoneNumber: "0812345672",
		Email: "suda@example.com", Gender: "F",
	},
	{
		FirstNameTH: "ปราณี", LastNameTH: "ทองดี",
		FirstNameEN: "Pranee", LastNameEN: "Thongdee",
		DateOfBirth: "1982-05-17", PatientHN: "HIS-HA-0005",
		PassportID: "P1234567", PhoneNumber: "0812345675",
		Email: "pranee@example.com", Gender: "F",
	},
}

// Search godoc
//
//	@Summary		Mock Hospital A HIS patient lookup
//	@Description	Standalone simulator of the external Hospital A HIS API (task 1). Hardcoded in-memory fixtures, not called by the middleware's own /patient/search.
//	@Tags			mock-his
//	@Produce		json
//	@Param			id	path		string	true	"national_id or passport_id"
//	@Success		200	{object}	patientRecord
//	@Failure		404	{object}	map[string]string
//	@Router			/mock-his/patient/search/{id} [get]
func Search(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	for _, p := range fixtures {
		if p.NationalID == id || p.PassportID == id {
			c.JSON(http.StatusOK, p)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
}
