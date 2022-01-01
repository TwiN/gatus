package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

func TestConfig_IsValid(t *testing.T) {
	c := &Config{
		Basic: nil,
		OIDC:  nil,
	}
	if c.IsValid() {
		t.Error("expected empty config to be valid")
	}
}

func TestConfig_ApplySecurityMiddleware(t *testing.T) {
	///////////
	// BASIC //
	///////////
	c := &Config{Basic: &BasicConfig{
		Username:           "john.doe",
		PasswordSha512Hash: "6b97ed68d14eb3f1aa959ce5d49c7dc612e1eb1dafd73b1e705847483fd6a6c809f2ceb4e8df6ff9984c6298ff0285cace6614bf8daa9f0070101b6c89899e22",
	}}
	api := mux.NewRouter()
	api.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	c.ApplySecurityMiddleware(api)
	// Try to access the route without basic auth
	request, _ := http.NewRequest("GET", "/test", nil)
	responseRecorder := httptest.NewRecorder()
	api.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusUnauthorized {
		t.Error("expected code to be 401, but was", responseRecorder.Code)
	}
	// Try again, but with basic auth
	request, _ = http.NewRequest("GET", "/test", nil)
	responseRecorder = httptest.NewRecorder()
	request.SetBasicAuth("john.doe", "hunter2")
	api.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusOK {
		t.Error("expected code to be 200, but was", responseRecorder.Code)
	}
	//////////
	// OIDC //
	//////////
	api = mux.NewRouter()
	api.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	c.OIDC = &OIDCConfig{
		IssuerURL:       "https://sso.gatus.io/",
		RedirectURL:     "http://localhost:80/authorization-code/callback",
		Scopes:          []string{"openid"},
		AllowedSubjects: []string{"user1@example.com"},
		oauth2Config:    oauth2.Config{},
		verifier:        nil,
	}
	c.Basic = nil
	c.ApplySecurityMiddleware(api)
	// Try without any session cookie
	request, _ = http.NewRequest("GET", "/test", nil)
	responseRecorder = httptest.NewRecorder()
	api.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusUnauthorized {
		t.Error("expected code to be 401, but was", responseRecorder.Code)
	}
	// Try with a session cookie
	request, _ = http.NewRequest("GET", "/test", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "123"})
	responseRecorder = httptest.NewRecorder()
	api.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusUnauthorized {
		t.Error("expected code to be 401, but was", responseRecorder.Code)
	}
}

func TestConfig_RegisterHandlers(t *testing.T) {
	c := &Config{}
	router := mux.NewRouter()
	c.RegisterHandlers(router)
	// Try to access the OIDC handler. This should fail, because the security config doesn't have OIDC
	request, _ := http.NewRequest("GET", "/oidc/login", nil)
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusNotFound {
		t.Error("expected code to be 404, but was", responseRecorder.Code)
	}
	// Set an empty OIDC config. This should fail, because the IssuerURL is required.
	c.OIDC = &OIDCConfig{}
	if err := c.RegisterHandlers(router); err == nil {
		t.Fatal("expected an error, but got none")
	}
	// Set the OIDC config and try again
	c.OIDC = &OIDCConfig{
		IssuerURL:       "https://sso.gatus.io/",
		RedirectURL:     "http://localhost:80/authorization-code/callback",
		Scopes:          []string{"openid"},
		AllowedSubjects: []string{"user1@example.com"},
	}
	if err := c.RegisterHandlers(router); err != nil {
		t.Fatal("expected no error, but got", err)
	}
	request, _ = http.NewRequest("GET", "/oidc/login", nil)
	responseRecorder = httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusFound {
		t.Error("expected code to be 302, but was", responseRecorder.Code)
	}
}
