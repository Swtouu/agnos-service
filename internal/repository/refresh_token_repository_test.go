package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/watt-siwat/agnos-backend/internal/model"
)

func TestRefreshTokenRepository_CreateAndFindByHash(t *testing.T) {
	truncateAll(t)
	repo := NewRefreshTokenRepository(testDB)
	hospital := seedHospital(t, "hosp_a")
	staffRepo := NewStaffRepository(testDB)
	staff := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice", PasswordHash: "hash", CreatedAt: time.Now()}
	if err := staffRepo.Create(context.Background(), staff); err != nil {
		t.Fatalf("seed staff: %v", err)
	}

	token := &model.RefreshToken{ID: uuid.New(), StaffID: staff.ID, TokenHash: "hash-1", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := repo.FindByHash(context.Background(), "hash-1")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found == nil || found.StaffID != staff.ID {
		t.Fatalf("got %+v, want token for staff %v", found, staff.ID)
	}
}

func TestRefreshTokenRepository_FindByHash_NotFound(t *testing.T) {
	truncateAll(t)
	repo := NewRefreshTokenRepository(testDB)

	found, err := repo.FindByHash(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for unknown hash, got %+v", found)
	}
}

func TestRefreshTokenRepository_DeleteByHash_RemovesRow(t *testing.T) {
	truncateAll(t)
	repo := NewRefreshTokenRepository(testDB)
	hospital := seedHospital(t, "hosp_a")
	staffRepo := NewStaffRepository(testDB)
	staff := &model.Staff{ID: uuid.New(), HospitalID: hospital.ID, Username: "alice", PasswordHash: "hash", CreatedAt: time.Now()}
	if err := staffRepo.Create(context.Background(), staff); err != nil {
		t.Fatalf("seed staff: %v", err)
	}

	token := &model.RefreshToken{ID: uuid.New(), StaffID: staff.ID, TokenHash: "hash-1", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.DeleteByHash(context.Background(), "hash-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	found, err := repo.FindByHash(context.Background(), "hash-1")
	if err != nil {
		t.Fatalf("find after delete: %v", err)
	}
	if found != nil {
		t.Error("expected token to be gone after DeleteByHash")
	}
}

func TestRefreshTokenRepository_DeleteByHash_NoMatchingRow_NoError(t *testing.T) {
	truncateAll(t)
	repo := NewRefreshTokenRepository(testDB)

	if err := repo.DeleteByHash(context.Background(), "never-existed"); err != nil {
		t.Errorf("deleting a nonexistent hash should not error, got: %v", err)
	}
}
