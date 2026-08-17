// Command seed populates Hospital/Staff/Patient fixtures for local development
// and grading. Safe to re-run: skips entirely if hospitals already exist.
package main

import (
	"encoding/base64"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/google/uuid"
	appcrypto "github.com/watt-siwat/agnos-backend/internal/crypto"
	"github.com/watt-siwat/agnos-backend/internal/model"
)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}

func hashPassword(pw string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}
	return string(hash)
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

type patientSeed struct {
	firstTH, middleTH, lastTH string
	firstEN, middleEN, lastEN string
	dob                       time.Time
	hn                        string
	nationalID                string // empty if using passport instead
	passportID                string
	phone, email, gender      string
}

func main() {
	dsn := mustEnv("DATABASE_URL")
	encKeyB64 := mustEnv("ENCRYPTION_KEY")
	hmacSecret := mustEnv("HMAC_SECRET")

	encKey, err := base64.StdEncoding.DecodeString(encKeyB64)
	if err != nil {
		log.Fatalf("decode ENCRYPTION_KEY (expected base64): %v", err)
	}
	cryptor, err := appcrypto.New(encKey, []byte(hmacSecret))
	if err != nil {
		log.Fatalf("init crypto: %v", err)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	var hospitalCount int64
	db.Model(&model.Hospital{}).Count(&hospitalCount)
	if hospitalCount > 0 {
		log.Println("hospitals already present, skipping seed")
		return
	}

	hospitalA := model.Hospital{ID: uuid.New(), Code: "hospital_a", Name: "Bangkok Central Hospital", CreatedAt: time.Now()}
	hospitalB := model.Hospital{ID: uuid.New(), Code: "hospital_b", Name: "Chiang Mai General Hospital", CreatedAt: time.Now()}
	if err := db.Create(&[]model.Hospital{hospitalA, hospitalB}).Error; err != nil {
		log.Fatalf("seed hospitals: %v", err)
	}

	staffPassword := hashPassword("Password123!")
	staffRows := []model.Staff{
		{ID: uuid.New(), HospitalID: hospitalA.ID, Username: "staff_a1", PasswordHash: staffPassword, CreatedAt: time.Now()},
		{ID: uuid.New(), HospitalID: hospitalA.ID, Username: "staff_a2", PasswordHash: staffPassword, CreatedAt: time.Now()},
		{ID: uuid.New(), HospitalID: hospitalB.ID, Username: "staff_b1", PasswordHash: staffPassword, CreatedAt: time.Now()},
		{ID: uuid.New(), HospitalID: hospitalB.ID, Username: "staff_b2", PasswordHash: staffPassword, CreatedAt: time.Now()},
	}
	if err := db.Create(&staffRows).Error; err != nil {
		log.Fatalf("seed staff: %v", err)
	}

	// Intentional name overlap between hospitals (both have a "Somchai Jaidee")
	// so partial-name-match tests actually exercise the hospital_id filter.
	hospitalAPatients := []patientSeed{
		{"สมชาย", "", "ใจดี", "Somchai", "", "Jaidee", date(1985, 3, 12), "HN-A-0001", "1100000000001", "", "0812345671", "somchai.a@example.com", "M"},
		{"สุดา", "", "ศรีสุข", "Suda", "", "Srisuk", date(1990, 7, 21), "HN-A-0002", "1100000000002", "", "0812345672", "suda@example.com", "F"},
		{"อนงค์", "", "วงศา", "Anong", "", "Wongsa", date(1978, 11, 2), "HN-A-0003", "1100000000003", "", "0812345673", "anong@example.com", "F"},
		{"วิชัย", "", "บุญมี", "Wichai", "", "Boonmee", date(1995, 1, 30), "HN-A-0004", "1100000000004", "", "0812345674", "wichai@example.com", "M"},
		{"ปราณี", "", "ทองดี", "Pranee", "", "Thongdee", date(1982, 5, 17), "HN-A-0005", "", "P1234567", "0812345675", "pranee@example.com", "F"},
	}
	hospitalBPatients := []patientSeed{
		{"สมชาย", "", "ใจดี", "Somchai", "", "Jaidee", date(1988, 9, 9), "HN-B-0001", "1100000000005", "", "0898765431", "somchai.b@example.com", "M"},
		{"ณัฐพงศ์", "", "ชัย", "Nattapong", "", "Chai", date(1992, 4, 14), "HN-B-0002", "1100000000006", "", "0898765432", "nattapong@example.com", "M"},
		{"กัญญา", "", "รัตนากร", "Kanya", "", "Rattanakorn", date(1975, 12, 25), "HN-B-0003", "1100000000007", "", "0898765433", "kanya@example.com", "F"},
		{"สมศักดิ์", "", "มีสุข", "Somsak", "", "Meesuk", date(1980, 6, 6), "HN-B-0004", "1100000000008", "", "0898765434", "somsak@example.com", "M"},
	}

	seedPatients := func(hospitalID uuid.UUID, seeds []patientSeed) {
		rows := make([]model.Patient, 0, len(seeds))
		for _, s := range seeds {
			p := model.Patient{
				ID:           uuid.New(),
				HospitalID:   hospitalID,
				FirstNameTH:  s.firstTH,
				MiddleNameTH: s.middleTH,
				LastNameTH:   s.lastTH,
				FirstNameEN:  s.firstEN,
				MiddleNameEN: s.middleEN,
				LastNameEN:   s.lastEN,
				DateOfBirth:  s.dob,
				PatientHN:    s.hn,
				PhoneNumber:  s.phone,
				Email:        s.email,
				Gender:       s.gender,
				CreatedAt:    time.Now(),
			}
			if s.nationalID != "" {
				enc, err := cryptor.Encrypt(s.nationalID)
				if err != nil {
					log.Fatalf("encrypt national_id: %v", err)
				}
				p.NationalIDEncrypted = enc
				p.NationalIDHash = cryptor.Hash(s.nationalID)
			}
			if s.passportID != "" {
				enc, err := cryptor.Encrypt(s.passportID)
				if err != nil {
					log.Fatalf("encrypt passport_id: %v", err)
				}
				p.PassportIDEncrypted = enc
				p.PassportIDHash = cryptor.Hash(s.passportID)
			}
			rows = append(rows, p)
		}
		if err := db.Create(&rows).Error; err != nil {
			log.Fatalf("seed patients: %v", err)
		}
	}

	seedPatients(hospitalA.ID, hospitalAPatients)
	seedPatients(hospitalB.ID, hospitalBPatients)

	log.Println("seed complete: 2 hospitals, 4 staff, 9 patients")
}
