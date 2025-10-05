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
