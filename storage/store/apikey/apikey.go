package apikey

import "time"

// APIKey represents an API key for authenticating API requests
// Defined in a separate package to avoid import cycles
type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	TokenHash   string     `json:"-"`
	UserSubject string     `json:"user_subject"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}
