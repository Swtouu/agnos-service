package model

import "time"

// PatientSearchFilters is the shape of a patient search query — owned here
// (not by service or repository) so repository can implement queries against
// it without depending on service, and service can build it without
// depending on repository.
type PatientSearchFilters struct {
	NationalIDHash string
	PassportIDHash string
	FirstName      string
	MiddleName     string
	LastName       string
	PhoneNumber    string
	Email          string
	DateOfBirth    *time.Time
	Limit          int
	Offset         int
}
