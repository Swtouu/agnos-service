package model

import (
	"time"

	"github.com/google/uuid"
)

// Patient mirrors the Hospital A HIS response shape (bilingual name columns)
// per task 2's "compatible with hospital data structures" requirement.
//
// NationalIDEncrypted/PassportIDEncrypted hold AES-GCM ciphertext for display;
// NationalIDHash/PassportIDHash hold deterministic HMAC-SHA256 digests used for
// exact-match search. See internal/crypto for the blind-index implementation.
type Patient struct {
	ID         uuid.UUID `gorm:"column:id;primaryKey"`
	HospitalID uuid.UUID `gorm:"column:hospital_id"`

	FirstNameTH  string `gorm:"column:first_name_th"`
	MiddleNameTH string `gorm:"column:middle_name_th"`
	LastNameTH   string `gorm:"column:last_name_th"`
	FirstNameEN  string `gorm:"column:first_name_en"`
	MiddleNameEN string `gorm:"column:middle_name_en"`
	LastNameEN   string `gorm:"column:last_name_en"`

	DateOfBirth time.Time `gorm:"column:date_of_birth"`
	PatientHN   string    `gorm:"column:patient_hn"`

	NationalIDEncrypted string `gorm:"column:national_id_encrypted"`
	NationalIDHash      string `gorm:"column:national_id_hash"`
	PassportIDEncrypted string `gorm:"column:passport_id_encrypted"`
	PassportIDHash      string `gorm:"column:passport_id_hash"`

	PhoneNumber string `gorm:"column:phone_number"`
	Email       string `gorm:"column:email"`
	Gender      string `gorm:"column:gender"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Patient) TableName() string { return "patients" }
