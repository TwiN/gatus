package security

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
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

func newPKCETestConfig(tokenURL string, pkce bool) *OIDCConfig {
	return &OIDCConfig{
		RedirectURL: "http://localhost:80/authorization-code/callback",
		ClientID:    "client-id",
		PKCE:        pkce,
		oauth2Config: oauth2.Config{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			Scopes:       []string{"openid"},
			RedirectURL:  "http://localhost:80/authorization-code/callback",
			Endpoint:     oauth2.Endpoint{AuthURL: "https://idp.example.com/authorize", TokenURL: tokenURL},
		},
	}
}

func TestOIDCConfig_loginHandler_PKCE(t *testing.T) {
	c := newPKCETestConfig("https://idp.example.com/token", true)
	app := fiber.New()
	app.Get("/login", c.loginHandler)
	response, err := app.Test(httptest.NewRequest("GET", "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusFound {
		t.Fatal("expected code to be 302, but was", response.StatusCode)
	}
	location, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if method := location.Query().Get("code_challenge_method"); method != "S256" {
		t.Errorf("expected code_challenge_method to be S256, but was %q", method)
	}
	var verifier string
	for _, cookie := range response.Cookies() {
		if cookie.Name == cookieNamePKCEVerifier {
			verifier = cookie.Value
			if !cookie.HttpOnly {
				t.Error("expected the verifier cookie to be HttpOnly")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Error("expected the verifier cookie to be SameSite=Lax")
			}
		}
	}
	if len(verifier) == 0 {
		t.Fatal("expected the verifier cookie to be set")
	}
	if challenge := location.Query().Get("code_challenge"); challenge != oauth2.S256ChallengeFromVerifier(verifier) {
		t.Errorf("expected code_challenge to be the S256 challenge of the verifier cookie, but was %q", challenge)
	}
	if strings.Contains(location.String(), verifier) {
		t.Error("the verifier must not appear in the authorization URL")
	}
}

func TestOIDCConfig_loginHandler_PKCEDisabled(t *testing.T) {
	c := newPKCETestConfig("https://idp.example.com/token", false)
	app := fiber.New()
	app.Get("/login", c.loginHandler)
	response, err := app.Test(httptest.NewRequest("GET", "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	location, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Query().Has("code_challenge") || location.Query().Has("code_challenge_method") {
		t.Error("expected no PKCE parameters when PKCE is disabled")
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == cookieNamePKCEVerifier {
			t.Error("expected no verifier cookie when PKCE is disabled")
		}
	}
}

func TestOIDCConfig_callbackHandler_PKCE(t *testing.T) {
	var receivedVerifier string
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		receivedVerifier = r.PostForm.Get("code_verifier")
		w.Header().Set("Content-Type", "application/json")
		// No id_token on purpose: the test only cares about what was sent to the token endpoint.
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"Bearer"}`))
	}))
	defer tokenServer.Close()

	t.Run("missing-verifier", func(t *testing.T) {
		c := newPKCETestConfig(tokenServer.URL, true)
		request := httptest.NewRequest("GET", "/authorization-code/callback?state=fake-state&code=fake-code", nil)
		request.AddCookie(&http.Cookie{Name: cookieNameState, Value: "fake-state"})
		responseRecorder := httptest.NewRecorder()
		c.callbackHandler(responseRecorder, request)
		if responseRecorder.Code != http.StatusBadRequest {
			t.Error("expected code to be 400, but was", responseRecorder.Code)
		}
	})
	t.Run("verifier-sent-and-cleared", func(t *testing.T) {
		receivedVerifier = ""
		c := newPKCETestConfig(tokenServer.URL, true)
		request := httptest.NewRequest("GET", "/authorization-code/callback?state=fake-state&code=fake-code", nil)
		request.AddCookie(&http.Cookie{Name: cookieNameState, Value: "fake-state"})
		request.AddCookie(&http.Cookie{Name: cookieNamePKCEVerifier, Value: "fake-verifier"})
		responseRecorder := httptest.NewRecorder()
		c.callbackHandler(responseRecorder, request)
		if receivedVerifier != "fake-verifier" {
			t.Errorf("expected the token endpoint to receive code_verifier=fake-verifier, but was %q", receivedVerifier)
		}
		// The missing id_token makes the handler fail after the exchange.
		if responseRecorder.Code != http.StatusInternalServerError {
			t.Error("expected code to be 500, but was", responseRecorder.Code)
		}
		cleared := false
		for _, cookie := range responseRecorder.Result().Cookies() {
			if cookie.Name == cookieNamePKCEVerifier && cookie.MaxAge < 0 {
				cleared = true
			}
		}
		if !cleared {
			t.Error("expected the verifier cookie to be cleared")
		}
	})
	t.Run("disabled-sends-no-verifier", func(t *testing.T) {
		receivedVerifier = "unset"
		c := newPKCETestConfig(tokenServer.URL, false)
		request := httptest.NewRequest("GET", "/authorization-code/callback?state=fake-state&code=fake-code", nil)
		request.AddCookie(&http.Cookie{Name: cookieNameState, Value: "fake-state"})
		request.AddCookie(&http.Cookie{Name: cookieNamePKCEVerifier, Value: "fake-verifier"})
		c.callbackHandler(httptest.NewRecorder(), request)
		if receivedVerifier != "" {
			t.Errorf("expected no code_verifier when PKCE is disabled, but was %q", receivedVerifier)
		}
	})
}
