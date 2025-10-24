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

func TestOIDCConfig_isAuthorized(t *testing.T) {
	tests := []struct {
		name           string
		config         *OIDCConfig
		subject        string
		claimsMap      map[string]any
		expectedResult bool
		description    string
	}{
		{
			name: "authorized with custom claim matching string value",
			config: &OIDCConfig{
				ClaimToCheck:    "groups",
				AllowedSubjects: []string{"admin", "users"},
			},
			subject: "user123",
			claimsMap: map[string]any{
				"groups": "aadmin",
			},
			expectedResult: true,
			description:    "Should authorize when custom claim value matches allowed subject",
		},
		{
			name: "not authorized with custom claim not matching",
			config: &OIDCConfig{
				ClaimToCheck:    "groups",
				AllowedSubjects: []string{"admin", "users"},
			},
			subject: "user123",
			claimsMap: map[string]any{
				"groups": "guests",
			},
			expectedResult: false,
			description:    "Should not authorize when custom claim value doesn't match any allowed subject",
		},
		{
			name: "not authorized when custom claim is missing",
			config: &OIDCConfig{
				ClaimToCheck:    "groups",
				AllowedSubjects: []string{"admin", "users"},
			},
			subject: "user123",
			claimsMap: map[string]any{
				"email": "user@example.com",
			},
			expectedResult: false,
			description:    "Should not authorize when custom claim is not present in claims map",
		},
		{
			name: "authorized with custom claim matching second value",
			config: &OIDCConfig{
				ClaimToCheck:    "role",
				AllowedSubjects: []string{"viewer", "editor", "admin"},
			},
			subject: "user456",
			claimsMap: map[string]any{
				"role": "editor",
			},
			expectedResult: true,
			description:    "Should authorize when custom claim matches second allowed subject",
		},
		{
			name: "not authorized with empty claims map",
			config: &OIDCConfig{
				ClaimToCheck:    "groups",
				AllowedSubjects: []string{"admin"},
			},
			subject:        "user789",
			claimsMap:      map[string]any{},
			expectedResult: false,
			description:    "Should not authorize when claims map is empty",
		},
		{
			name: "authorized with matching subject case insensitive",
			config: &OIDCConfig{
				ClaimToCheck:    "",
				AllowedSubjects: []string{"user@example.com", "admin@example.com"},
			},
			subject: "User@Example.Com",
			claimsMap: map[string]any{
				"email": "user@example.com",
			},
			expectedResult: true,
			description:    "Should authorize when subject matches allowed subject (case insensitive)",
		},
		{
			name: "authorized with exact subject match",
			config: &OIDCConfig{
				ClaimToCheck:    "",
				AllowedSubjects: []string{"user@example.com", "admin@example.com"},
			},
			subject: "user@example.com",
			claimsMap: map[string]any{
				"email": "user@example.com",
			},
			expectedResult: true,
			description:    "Should authorize when subject exactly matches allowed subject",
		},
		{
			name: "not authorized with non-matching subject",
			config: &OIDCConfig{
				ClaimToCheck:    "",
				AllowedSubjects: []string{"user@example.com", "admin@example.com"},
			},
			subject: "guest@example.com",
			claimsMap: map[string]any{
				"email": "guest@example.com",
			},
			expectedResult: false,
			description:    "Should not authorize when subject doesn't match any allowed subject",
		},
		{
			name: "authorized with subject matching last in list",
			config: &OIDCConfig{
				ClaimToCheck:    "",
				AllowedSubjects: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
			},
			subject: "user3@example.com",
			claimsMap: map[string]any{
				"email": "user3@example.com",
			},
			expectedResult: true,
			description:    "Should authorize when subject matches last allowed subject",
		},
		{
			name: "not authorized with empty allowed subjects",
			config: &OIDCConfig{
				ClaimToCheck:    "",
				AllowedSubjects: []string{},
			},
			subject: "user@example.com",
			claimsMap: map[string]any{
				"email": "user@example.com",
			},
			expectedResult: false,
			description:    "Should not authorize when allowed subjects list is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.isAuthorized(tt.subject, tt.claimsMap)
			if result != tt.expectedResult {
				t.Errorf("%s: expected authorization=%v, got=%v", tt.description, tt.expectedResult, result)
			}
		})
	}
}
