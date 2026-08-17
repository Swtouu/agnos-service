package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/model"
)

func TestStaffRepository_Create_Success(t *testing.T) {
	truncateAll(t)
	repo := NewStaffRepository(testDB)
	hospital := seedHospital(t, "hosp_a")

	staff := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice", PasswordHash: "hash", CreatedAt: time.Now()}
	if err := repo.Create(context.Background(), staff); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.FindByHospitalAndUsername(context.Background(), hospital.ID, "alice")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find just-created staff")
	}
}

func TestStaffRepository_Create_DuplicateUsernameSameHospital_ReturnsErrDuplicateKey(t *testing.T) {
	// Exercises the real Postgres UNIQUE (hospital_id, username) constraint and
	// the pgconn.PgError -> model.ErrDuplicateKey translation added to fix the
	// race where two concurrent /staff/create calls for the same username
	// previously surfaced as an opaque 500 instead of "username taken".
	truncateAll(t)
	repo := NewStaffRepository(testDB)
	hospital := seedHospital(t, "hosp_a")

	first := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "race_user", PasswordHash: "hash", CreatedAt: time.Now()}
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	second := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "race_user", PasswordHash: "hash", CreatedAt: time.Now()}
	err := repo.Create(context.Background(), second)
	if !errors.Is(err, model.ErrDuplicateKey) {
		t.Errorf("got err %v, want model.ErrDuplicateKey", err)
	}
}

func TestStaffRepository_Create_SameUsernameDifferentHospital_Allowed(t *testing.T) {
	// Uniqueness is per (hospital_id, username), not global — the same
	// username must be allowed to exist independently in two hospitals.
	truncateAll(t)
	repo := NewStaffRepository(testDB)
	hospitalA := seedHospital(t, "hosp_a")
	hospitalB := seedHospital(t, "hosp_b")

	a := &model.Staff{ID: uuid.New(), HospitalID: hospitalA.ID, Username: "alice", PasswordHash: "hash", CreatedAt: time.Now()}
	b := &model.Staff{ID: uuid.New(), HospitalID: hospitalB.ID, Username: "alice", PasswordHash: "hash", CreatedAt: time.Now()}

	if err := repo.Create(context.Background(), a); err != nil {
		t.Fatalf("create in hospitalA: %v", err)
	}
	if err := repo.Create(context.Background(), b); err != nil {
		t.Fatalf("create same username in hospitalB should succeed: %v", err)
	}
}

func TestStaffRepository_FindByHospitalAndUsername_NotFound(t *testing.T) {
	truncateAll(t)
	repo := NewStaffRepository(testDB)
	hospital := seedHospital(t, "hosp_a")

	found, err := repo.FindByHospitalAndUsername(context.Background(), hospital.ID, "ghost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for nonexistent username, got %+v", found)
	}
}

func TestStaffRepository_FindByHospitalAndUsername_WrongHospital_NotFound(t *testing.T) {
	// A username that exists in hospitalA must not be found when looked up
	// under hospitalB's ID — this is the tenant-isolation guarantee at the
	// SQL level for staff lookups (used by /staff/login).
	truncateAll(t)
	repo := NewStaffRepository(testDB)
	hospitalA := seedHospital(t, "hosp_a")
	hospitalB := seedHospital(t, "hosp_b")

	staff := &model.Staff{ID: uuid.New(), HospitalID: hospitalA.ID, Username: "alice", PasswordHash: "hash", CreatedAt: time.Now()}
	if err := repo.Create(context.Background(), staff); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.FindByHospitalAndUsername(context.Background(), hospitalB.ID, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Error("expected nil — staff belongs to hospitalA, not hospitalB")
	}
}

func TestStaffRepository_FindByID_FoundAndNotFound(t *testing.T) {
	truncateAll(t)
	repo := NewStaffRepository(testDB)
	hospital := seedHospital(t, "hosp_a")

	staff := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice", PasswordHash: "hash", CreatedAt: time.Now()}
	if err := repo.Create(context.Background(), staff); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.FindByID(context.Background(), staff.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil || found.Username != "alice" {
		t.Fatalf("got %+v, want alice", found)
	}

	notFound, err := repo.FindByID(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for unknown ID")
	}
}
