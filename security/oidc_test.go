package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

func TestOIDCConfig_ValidateAndSetDefaults(t *testing.T) {
	c := &OIDCConfig{
		IssuerURL:       "https://sso.gatus.io/",
		RedirectURL:     "http://localhost:80/authorization-code/callback",
		ClientID:        "client-id",
		ClientSecret:    "client-secret",
		Scopes:          []string{"openid"},
		AllowedSubjects: []string{"user1@example.com"},
		SessionTTL:      0, // Not set! ValidateAndSetDefaults should set it to DefaultOIDCSessionTTL
	}
	if !c.ValidateAndSetDefaults() {
		t.Error("OIDCConfig should be valid")
	}
	if c.SessionTTL != DefaultOIDCSessionTTL {
		t.Error("expected SessionTTL to be set to DefaultOIDCSessionTTL")
	}
}

func TestOIDCConfig_callbackHandler(t *testing.T) {
	c := &OIDCConfig{
		IssuerURL:       "https://sso.gatus.io/",
		RedirectURL:     "http://localhost:80/authorization-code/callback",
		ClientID:        "client-id",
		ClientSecret:    "client-secret",
		Scopes:          []string{"openid"},
		AllowedSubjects: []string{"user1@example.com"},
	}
	if err := c.initialize(); err != nil {
		t.Fatal("expected no error, but got", err)
	}
	// Try with no state cookie
	request, _ := http.NewRequest("GET", "/authorization-code/callback", nil)
	responseRecorder := httptest.NewRecorder()
	c.callbackHandler(responseRecorder, request)
	if responseRecorder.Code != http.StatusBadRequest {
		t.Error("expected code to be 400, but was", responseRecorder.Code)
	}
	// Try with state cookie
	request, _ = http.NewRequest("GET", "/authorization-code/callback", nil)
	request.AddCookie(&http.Cookie{Name: cookieNameState, Value: "fake-state"})
	responseRecorder = httptest.NewRecorder()
	c.callbackHandler(responseRecorder, request)
	if responseRecorder.Code != http.StatusBadRequest {
		t.Error("expected code to be 400, but was", responseRecorder.Code)
	}
	// Try with state cookie and state query parameter
	request, _ = http.NewRequest("GET", "/authorization-code/callback?state=fake-state", nil)
	request.AddCookie(&http.Cookie{Name: cookieNameState, Value: "fake-state"})
	responseRecorder = httptest.NewRecorder()
	c.callbackHandler(responseRecorder, request)
	// Exchange should fail, so 500.
	if responseRecorder.Code != http.StatusInternalServerError {
		t.Error("expected code to be 500, but was", responseRecorder.Code)
	}
}

func TestOIDCConfig_setSessionCookie(t *testing.T) {
	c := &OIDCConfig{}
	responseRecorder := httptest.NewRecorder()
	c.setSessionCookie(responseRecorder, &oidc.IDToken{Subject: "test@example.com"})
	if len(responseRecorder.Result().Cookies()) == 0 {
		t.Error("expected cookie to be set")
	}
}

func TestOIDCConfig_setSessionCookieWithCustomTTL(t *testing.T) {
	customTTL := 30 * time.Minute
	c := &OIDCConfig{SessionTTL: customTTL}
	responseRecorder := httptest.NewRecorder()
	c.setSessionCookie(responseRecorder, &oidc.IDToken{Subject: "test@example.com"})
	cookies := responseRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Error("expected cookie to be set")
	}
	sessionCookie := cookies[0]
	if sessionCookie.MaxAge != int(customTTL.Seconds()) {
		t.Errorf("expected cookie MaxAge to be %d, but was %d", int(customTTL.Seconds()), sessionCookie.MaxAge)
	}
}

func TestOIDCConfig_getClaimToCheck(t *testing.T) {
	tests := []struct {
		name         string
		claimToCheck string
		subject      string
		claimsMap    map[string]any
		expected     string
	}{
		{
			name:         "returns subject when ClaimToCheck is empty",
			claimToCheck: "",
			subject:      "user@example.com",
			claimsMap:    map[string]any{},
			expected:     "user@example.com",
		},
		{
			name:         "returns subject when ClaimToCheck is sub",
			claimToCheck: "sub",
			subject:      "user@example.com",
			claimsMap:    map[string]any{},
			expected:     "user@example.com",
		},
		{
			name:         "returns claim value when ClaimToCheck exists in claims",
			claimToCheck: "preferred_username",
			subject:      "user@example.com",
			claimsMap: map[string]any{
				"preferred_username": "john.doe",
				"email":              "john@example.com",
			},
			expected: "john.doe",
		},
		{
			name:         "returns empty string when ClaimToCheck doesn't exist in claims",
			claimToCheck: "non_existent_claim",
			subject:      "user@example.com",
			claimsMap: map[string]any{
				"preferred_username": "john.doe",
				"email":              "john@example.com",
			},
			expected: "",
		},
		{
			name:         "returns claim value when claims map has multiple fields",
			claimToCheck: "email",
			subject:      "sub-12345",
			claimsMap: map[string]any{
				"email":              "admin@example.com",
				"preferred_username": "admin",
				"groups":             []string{"admins", "users"},
			},
			expected: "admin@example.com",
		},
		{
			name:         "returns empty string when claims map is nil",
			claimToCheck: "email",
			subject:      "user@example.com",
			claimsMap:    nil,
			expected:     "",
		},
		{
			name:         "returns empty string when ClaimToCheck is empty and subject is empty",
			claimToCheck: "",
			subject:      "",
			claimsMap:    map[string]any{},
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OIDCConfig{
				ClaimToCheck: tt.claimToCheck,
			}

			result := config.getClaimToCheck(tt.subject, tt.claimsMap)

			if result != tt.expected {
				t.Errorf("getClaimToCheck() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOIDCConfig_isAuthorized(t *testing.T) {
	tests := []struct {
		name            string
		allowedSubjects []string
		claimValue      string
		expected        bool
	}{
		{
			name:            "authorized subject matches first element",
			allowedSubjects: []string{"user1", "user2", "user3"},
			claimValue:      "user1",
			expected:        true,
		},
		{
			name:            "authorized subject matches middle element",
			allowedSubjects: []string{"user1", "user2", "user3"},
			claimValue:      "user2",
			expected:        true,
		},
		{
			name:            "authorized subject matches last element",
			allowedSubjects: []string{"user1", "user2", "user3"},
			claimValue:      "user3",
			expected:        true,
		},
		{
			name:            "unauthorized subject not in list",
			allowedSubjects: []string{"user1", "user2", "user3"},
			claimValue:      "user4",
			expected:        false,
		},
		{
			name:            "empty claim value",
			allowedSubjects: []string{"user1", "user2", "user3"},
			claimValue:      "",
			expected:        false,
		},
		{
			name:            "empty allowed subjects list",
			allowedSubjects: []string{},
			claimValue:      "user1",
			expected:        false,
		},
		{
			name:            "nil allowed subjects list",
			allowedSubjects: nil,
			claimValue:      "user1",
			expected:        false,
		},
		{
			name:            "case sensitive match - different case",
			allowedSubjects: []string{"User1", "user2"},
			claimValue:      "user1",
			expected:        false,
		},
		{
			name:            "exact match with special characters",
			allowedSubjects: []string{"user@example.com", "admin-user"},
			claimValue:      "user@example.com",
			expected:        true,
		},
		{
			name:            "single allowed subject - match",
			allowedSubjects: []string{"single-user"},
			claimValue:      "single-user",
			expected:        true,
		},
		{
			name:            "single allowed subject - no match",
			allowedSubjects: []string{"single-user"},
			claimValue:      "other-user",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OIDCConfig{
				AllowedSubjects: tt.allowedSubjects,
			}

			result := c.isAuthorized(tt.claimValue)

			if result != tt.expected {
				t.Errorf("isAuthorized() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
