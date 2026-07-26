package security_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zerodayz7/platform/services/auth-service/internal/shared/security"
)

func TestGenerateAndValidateJWT(t *testing.T) {
	secret := "super-secret-key-for-testing"

	t.Run("Valid Token Generation and Validation", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":  "user-123",
			"role": "admin",
		}

		tokenStr, err := security.GenerateJWT(claims, secret, 15*time.Minute)
		require.NoError(t, err)
		require.NotEmpty(t, tokenStr)

		parsedToken, err := security.ValidateJWT(tokenStr, secret)
		require.NoError(t, err)
		assert.True(t, parsedToken.Valid)

		parsedClaims, ok := parsedToken.Claims.(jwt.MapClaims)
		require.True(t, ok)
		assert.Equal(t, "user-123", parsedClaims["sub"])
		assert.Equal(t, "admin", parsedClaims["role"])
	})

	t.Run("Invalid Secret Key", func(t *testing.T) {
		claims := jwt.MapClaims{"sub": "user-123"}
		tokenStr, err := security.GenerateJWT(claims, secret, 15*time.Minute)
		require.NoError(t, err)

		_, err = security.ValidateJWT(tokenStr, "wrong-secret-key")
		assert.Error(t, err)
	})

	t.Run("Expired Token", func(t *testing.T) {
		claims := jwt.MapClaims{"sub": "user-123"}
		// Tworzymy token wygasły chwilę temu (-1 sekunda)
		tokenStr, err := security.GenerateJWT(claims, secret, -1*time.Second)
		require.NoError(t, err)

		_, err = security.ValidateJWT(tokenStr, secret)
		assert.Error(t, err)
	})

	t.Run("Tampered Token Signature", func(t *testing.T) {
		claims := jwt.MapClaims{"sub": "user-123"}
		tokenStr, err := security.GenerateJWT(claims, secret, 15*time.Minute)
		require.NoError(t, err)

		tamperedToken := tokenStr + "extra_bytes"
		_, err = security.ValidateJWT(tamperedToken, secret)
		assert.Error(t, err)
	})
}

func TestGenerateRandomToken(t *testing.T) {
	t.Run("Generates Correct Length and Unique Tokens", func(t *testing.T) {
		token1, err := security.GenerateRandomToken(32)
		require.NoError(t, err)
		require.NotEmpty(t, token1)

		token2, err := security.GenerateRandomToken(32)
		require.NoError(t, err)
		require.NotEmpty(t, token2)

		assert.NotEqual(t, token1, token2, "Two generated random tokens should not be identical")
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	t.Run("Generates Valid Refresh Token", func(t *testing.T) {
		refreshToken, err := security.GenerateRefreshToken()
		require.NoError(t, err)
		require.NotEmpty(t, refreshToken)

		// 64 bajty w Base64 RawURLEncoding dają długość około 86 znaków
		assert.GreaterOrEqual(t, len(refreshToken), 80)
	})
}
