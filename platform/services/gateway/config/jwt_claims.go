package config

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaimsData reprezentuje wyciągnięte i sparsowane dane z claims JWT
type JWTClaimsData struct {
	SessionID       uuid.UUID
	UserID          uuid.UUID
	Scope           string
	FingerprintHash string
}

// ExtractJWTClaims pobiera i parsuje zdekodowane claims JWT do bezpiecznej struktury
func ExtractJWTClaims(claims jwt.MapClaims) (JWTClaimsData, error) {
	var data JWTClaimsData
	var err error

	// 1. Scope
	if scope, ok := claims["scope"].(string); ok {
		data.Scope = scope
	}

	// 2. Fingerprint (fpt)
	if fpt, ok := claims["fpt"].(string); ok {
		data.FingerprintHash = fpt
	}

	// 3. SessionID (sid)
	sidStr, ok := claims["sid"].(string)
	if !ok || sidStr == "" {
		return data, errors.New("missing sid claim in JWT")
	}

	data.SessionID, err = uuid.Parse(sidStr)
	if err != nil || data.SessionID == uuid.Nil {
		return data, errors.New("invalid sid format in JWT")
	}

	// 4. UserID (uid)
	uidStr, ok := claims["uid"].(string)
	if !ok || uidStr == "" {
		return data, errors.New("missing uid claim in JWT")
	}

	data.UserID, err = uuid.Parse(uidStr)
	if err != nil || data.UserID == uuid.Nil {
		return data, errors.New("invalid uid format in JWT")
	}

	return data, nil
}
