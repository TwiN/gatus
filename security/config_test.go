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
	// Bcrypt
	c := &Config{Basic: &BasicConfig{
		Username:                        "john.doe",
		PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
	}}
	api := mux.NewRouter()
	api.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := c.ApplySecurityMiddleware(api); err != nil {
		t.Error("expected no error, but was", err)
	}
	// Try to access the route without basic auth
	request, _ := http.NewRequest("GET", "/test", http.NoBody)
	responseRecorder := httptest.NewRecorder()
	api.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusUnauthorized {
		t.Error("expected code to be 401, but was", responseRecorder.Code)
	}
	// Try again, but with basic auth
	request, _ = http.NewRequest("GET", "/test", http.NoBody)
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
	if err := c.ApplySecurityMiddleware(api); err != nil {
		t.Error("expected no error, but was", err)
	}
	// Try without any session cookie
	request, _ = http.NewRequest("GET", "/test", http.NoBody)
	responseRecorder = httptest.NewRecorder()
	api.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusUnauthorized {
		t.Error("expected code to be 401, but was", responseRecorder.Code)
	}
	// Try with a session cookie
	request, _ = http.NewRequest("GET", "/test", http.NoBody)
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
	request, _ := http.NewRequest("GET", "/oidc/login", http.NoBody)
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
	request, _ = http.NewRequest("GET", "/oidc/login", http.NoBody)
	responseRecorder = httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusFound {
		t.Error("expected code to be 302, but was", responseRecorder.Code)
	}
}
