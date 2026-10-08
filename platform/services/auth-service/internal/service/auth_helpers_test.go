package service

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
)

func TestHashRefreshTokenForLogoutUsesSHA256(t *testing.T) {
	raw := "logout-refresh-token-123"
	hash := hashRefreshTokenForLogout(raw)

	expected := sha256.Sum256([]byte(raw))
	if hash != hex.EncodeToString(expected[:]) {
		t.Fatalf("expected SHA256 hash %s, got %s", hex.EncodeToString(expected[:]), hash)
	}
}

func TestBuildUserSessionIncludesFullProfileData(t *testing.T) {
	userID := uuid.New()
	institutionID := uuid.New()
	departmentID := uuid.New()

	user := &model.User{
		ID:          userID,
		Username:    "jan.kowalski",
		Email:       "jan.kowalski@example.com",
		Role:        model.RoleUser,
		Permissions: []string{"user:read", "user:write"},
		EmployeeProfile: &model.EmployeeProfile{
			EmployeeNumber: "EMP-123",
			InstitutionID:  institutionID,
			DepartmentID:   departmentID,
			Permissions:    []string{"worker:read", "worker:approve"},
		},
	}

	session := (&authService{}).buildUserSession(user, "fp-123", "pub-key-1", true)

	if session.UserID != userID.String() {
		t.Fatalf("expected user id %s, got %s", userID, session.UserID)
	}
	if session.Role != string(user.Role) {
		t.Fatalf("expected role %s, got %s", user.Role, session.Role)
	}
	if session.DeviceID != "fp-123" {
		t.Fatalf("expected device id fp-123, got %s", session.DeviceID)
	}
	if session.Fingerprint != "fp-123" {
		t.Fatalf("expected fingerprint fp-123, got %s", session.Fingerprint)
	}
	if session.PublicKey != "pub-key-1" {
		t.Fatalf("expected public key pub-key-1, got %s", session.PublicKey)
	}
	if session.Username != user.Username {
		t.Fatalf("expected username %s, got %s", user.Username, session.Username)
	}
	if session.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, session.Email)
	}
	if session.IsReadOnly != true {
		t.Fatal("expected read-only session flag to be true")
	}
	if len(session.Permissions) != 2 || session.Permissions[0] != "user:read" || session.Permissions[1] != "user:write" {
		t.Fatalf("expected user-level permissions to win, got %#v", session.Permissions)
	}
	if session.Employee == nil {
		t.Fatal("expected employee context to be populated")
	}
	if session.Employee.EmployeeNumber != "EMP-123" {
		t.Fatalf("expected employee number EMP-123, got %s", session.Employee.EmployeeNumber)
	}
	if session.Employee.InstitutionID != institutionID.String() {
		t.Fatalf("expected institution id %s, got %s", institutionID, session.Employee.InstitutionID)
	}
	if session.Employee.DepartmentID != departmentID.String() {
		t.Fatalf("expected department id %s, got %s", departmentID, session.Employee.DepartmentID)
	}
	if session.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if time.Since(session.CreatedAt) < 0 {
		t.Fatal("expected CreatedAt to be in the past")
	}
}
