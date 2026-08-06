package service

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"

	"github.com/zerodayz7/platform/pkg/schemas"
)

type KeyService interface {
	GetJWKS(ctx context.Context) (*schemas.JWKSResponse, error)
}

type keyService struct {
	cachedJWKS *schemas.JWKSResponse
}

func NewKeyService(keyID string, pubKey ed25519.PublicKey) KeyService {
	pubKeyX := ""
	if len(pubKey) == ed25519.PublicKeySize {
		pubKeyX = base64.RawURLEncoding.EncodeToString(pubKey)
	}

	jwks := &schemas.JWKSResponse{
		Keys: []schemas.JWK{
			{
				Kty: "OKP",
				Crv: "Ed25519",
				Alg: "EdDSA",
				Use: "sig",
				Kid: keyID,
				X:   pubKeyX,
			},
		},
	}

	return &keyService{
		cachedJWKS: jwks,
	}
}

func (s *keyService) GetJWKS(ctx context.Context) (*schemas.JWKSResponse, error) {
	return s.cachedJWKS, nil
}
