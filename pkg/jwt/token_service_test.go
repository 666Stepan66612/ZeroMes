package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func createTestToken(userID, secret string, signingMethod jwt.SigningMethod) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	return token.SignedString([]byte(secret))
}

func TestValidateAccessToken_ValidToken(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"

	tokenString, err := createTestToken(userID, secret, jwt.SigningMethodHS256)
	assert.NoError(t, err)

	extractedUserID, err := ValidateAccessToken(tokenString, secret)

	assert.NoError(t, err)
	assert.Equal(t, userID, extractedUserID)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	correctSecret := "correct-secret"
	wrongSecret := "wrong-secret"
	userID := "user-123"

	tokenString, err := createTestToken(userID, correctSecret, jwt.SigningMethodHS256)
	assert.NoError(t, err)

	extractedUserID, err := ValidateAccessToken(tokenString, wrongSecret)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Empty(t, extractedUserID)
}

func TestValidateAccessToken_ExpiredToken(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	extractedUserID, err := ValidateAccessToken(tokenString, secret)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Empty(t, extractedUserID)
}

func TestValidateAccessToken_WrongAlgorithm(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"

	tokenString, err := createTestToken(userID, secret, jwt.SigningMethodHS512)
	assert.NoError(t, err)

	extractedUserID, err := ValidateAccessToken(tokenString, secret)
	assert.NoError(t, err)
	assert.Equal(t, userID, extractedUserID)
}

func TestValidateAccessToken_InvalidTokenString(t *testing.T) {
	secret := "test-secret-key"

	testCases := []struct {
		name        string
		tokenString string
	}{
		{"empty string", ""},
		{"random string", "not-a-jwt-token"},
		{"malformed jwt", "header.payload"},
		{"invalid base64", "ey!invalid.ey!invalid.signature"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extractedUserID, err := ValidateAccessToken(tc.tokenString, secret)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidToken, err)
			assert.Empty(t, extractedUserID)
		})
	}
}

func TestValidateAccessToken_MissingUserID(t *testing.T) {
	secret := "test-secret-key"

	claims := &Claims{
		UserID: "",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	extractedUserID, err := ValidateAccessToken(tokenString, secret)

	assert.NoError(t, err)
	assert.Empty(t, extractedUserID)
}
