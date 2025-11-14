package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAPIConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *APIConfig
		wantErr bool
	}{
		{
			name:    "nil config should be valid",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid tokens",
			config: &APIConfig{
				Tokens: []string{"token1", "token2"},
			},
			wantErr: false,
		},
		{
			name: "empty token list should be valid",
			config: &APIConfig{
				Tokens: []string{},
			},
			wantErr: false,
		},
		{
			name: "empty token string should be invalid",
			config: &APIConfig{
				Tokens: []string{"valid-token", ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("APIConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != ErrAPITokensEmpty {
				t.Errorf("APIConfig.Validate() error = %v, expected %v", err, ErrAPITokensEmpty)
			}
		})
	}
}

func TestAPIConfig_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		config *APIConfig
		token  string
		want   bool
	}{
		{
			name:   "nil config",
			config: nil,
			token:  "any-token",
			want:   false,
		},
		{
			name: "empty token list",
			config: &APIConfig{
				Tokens: []string{},
			},
			token: "any-token",
			want:  false,
		},
		{
			name: "empty token string",
			config: &APIConfig{
				Tokens: []string{"valid-token"},
			},
			token: "",
			want:  false,
		},
		{
			name: "valid token matches",
			config: &APIConfig{
				Tokens: []string{"token1", "token2", "token3"},
			},
			token: "token2",
			want:  true,
		},
		{
			name: "token does not match",
			config: &APIConfig{
				Tokens: []string{"token1", "token2"},
			},
			token: "invalid-token",
			want:  false,
		},
		{
			name: "partial token match should not work",
			config: &APIConfig{
				Tokens: []string{"my-secret-token"},
			},
			token: "my-secret",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsValid(tt.token); got != tt.want {
				t.Errorf("APIConfig.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_IsAuthenticated_WithAPITokens(t *testing.T) {
	c := &Config{
		API: &APIConfig{
			Tokens: []string{"valid-token-123"},
		},
	}
	app := fiber.New()

	// Simulate /api/v1/config endpoint behavior
	app.Get("/api/v1/config", func(ctx *fiber.Ctx) error {
		isAuthenticated := c.IsAuthenticated(ctx)
		response := map[string]interface{}{
			"authenticated": isAuthenticated,
			"oidc":          false,
		}
		return ctx.JSON(response)
	})

	// Test with valid API token
	t.Run("valid api token returns authenticated true", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/api/v1/config", http.NoBody)
		request.Header.Set("Authorization", "Bearer valid-token-123")
		response, err := app.Test(request)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if response.StatusCode != 200 {
			t.Error("expected code to be 200, but was", response.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal("failed to decode response:", err)
		}

		if result["authenticated"] != true {
			t.Error("expected authenticated to be true with valid token, but was", result["authenticated"])
		}
	})

	// Test with invalid API token
	t.Run("invalid api token returns authenticated false", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/api/v1/config", http.NoBody)
		request.Header.Set("Authorization", "Bearer invalid-token")
		response, err := app.Test(request)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if response.StatusCode != 200 {
			t.Error("expected code to be 200, but was", response.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal("failed to decode response:", err)
		}

		if result["authenticated"] != false {
			t.Error("expected authenticated to be false with invalid token, but was", result["authenticated"])
		}
	})

	// Test without Authorization header
	t.Run("no authorization header returns authenticated false", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/api/v1/config", http.NoBody)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if response.StatusCode != 200 {
			t.Error("expected code to be 200, but was", response.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal("failed to decode response:", err)
		}

		if result["authenticated"] != false {
			t.Error("expected authenticated to be false without auth header, but was", result["authenticated"])
		}
	})

	// Test with malformed Bearer header
	t.Run("malformed bearer header returns authenticated false", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/api/v1/config", http.NoBody)
		request.Header.Set("Authorization", "bearer valid-token-123")
		response, err := app.Test(request)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if response.StatusCode != 200 {
			t.Error("expected code to be 200, but was", response.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal("failed to decode response:", err)
		}

		if result["authenticated"] != false {
			t.Error("expected authenticated to be false with lowercase bearer, but was", result["authenticated"])
		}
	})
}
