package model

import "github.com/google/uuid"

// Actor reprezentuje tożsamość użytkownika lub systemu wykonującego operację.
type Actor struct {
	ID            uuid.UUID
	Name          string
	Role          string
	DepartmentID  *uuid.UUID
	InstitutionID *uuid.UUID
	ClientIP      string
}

// Helpery (opcjonalnie) dla czytelności w serwisie:
func (a Actor) IsOfficer() bool {
	return a.Role == "OFFICER"
}

func (a Actor) DepartmentIDString() string {
	if a.DepartmentID != nil {
		return a.DepartmentID.String()
	}
	return "-"
}

func (a Actor) InstitutionIDString() string {
	if a.InstitutionID != nil {
		return a.InstitutionID.String()
	}
	return "-"
}
