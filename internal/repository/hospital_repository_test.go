package repository

import (
	"context"
	"testing"
)

func TestHospitalRepository_FindByCode_Found(t *testing.T) {
	truncateAll(t)
	repo := NewHospitalRepository(testDB)
	hospital := seedHospital(t, "hospital_a")

	found, err := repo.FindByCode(context.Background(), "hospital_a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil || found.ID != hospital.ID {
		t.Fatalf("got %+v, want hospital %v", found, hospital.ID)
	}
}

func TestHospitalRepository_FindByCode_NotFound(t *testing.T) {
	truncateAll(t)
	repo := NewHospitalRepository(testDB)

	found, err := repo.FindByCode(context.Background(), "does_not_exist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for unknown code, got %+v", found)
	}
}
