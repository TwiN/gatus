package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/TwiN/gatus/v5/storage/store/apikey"
	"golang.org/x/crypto/bcrypt"
)

const (
	// APIKeyLength is the length of the generated API key in bytes (before base64 encoding)
	APIKeyLength = 32
	// APIKeyPrefix is the prefix for API keys to make them easily identifiable
	APIKeyPrefix = "gatus_"
)

var (
	// ErrInvalidAPIKey is returned when an API key is invalid
	ErrInvalidAPIKey = errors.New("invalid API key")
	// ErrAPIKeyNotFound is returned when an API key is not found
	ErrAPIKeyNotFound = errors.New("API key not found")
)

// APIKeyWithToken is used only when creating a new API key to return the full token once
type APIKeyWithToken struct {
	apikey.APIKey
	// Token is the full API key token (only available at creation time)
	Token string `json:"token"`
}

// GenerateAPIKey creates a new API key for the given user
func GenerateAPIKey(userSubject, name string) (*APIKeyWithToken, error) {
	// Generate random bytes for the token
	tokenBytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	// Encode to base64 and add prefix
	token := APIKeyPrefix + base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash the token for storage
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Generate a unique ID for the key (using a different random value)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	id := base64.URLEncoding.EncodeToString(idBytes)

	key := &APIKeyWithToken{
		APIKey: apikey.APIKey{
			ID:          id,
			Name:        name,
			TokenHash:   string(tokenHash),
			UserSubject: userSubject,
			CreatedAt:   time.Now(),
		},
		Token: token,
	}

	return key, nil
}

// ValidateAPIKey checks if the provided token matches the stored hash
func ValidateAPIKey(key *apikey.APIKey, token string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(key.TokenHash), []byte(token))
	return err == nil
}
