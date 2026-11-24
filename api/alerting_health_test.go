package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/health"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/ui"
	"github.com/TwiN/gatus/v5/security"
	"github.com/gofiber/fiber/v2"
)

func TestAlertingHealth_UnauthorizedWithBasicAuth(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{UI: &ui.Config{}}
	cfg.Security = &security.Config{
		Basic: &security.BasicConfig{
			Username:                        "john.doe",
			PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
		},
	}
	health.ResetAndSetProviders(nil)
	health.SetEndpointWarnings(nil)
	api := New(cfg)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/health", http.NoBody)
	response, err := api.Router().Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %d", response.StatusCode)
	}
}

func TestAlertingHealth_AuthorizedWithBasicAuth(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{UI: &ui.Config{}}
	cfg.Security = &security.Config{
		Basic: &security.BasicConfig{
			Username:                        "john.doe",
			PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
		},
	}
	health.ResetAndSetProviders(nil)
	health.SetEndpointWarnings(nil)
	api := New(cfg)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/health", http.NoBody)
	request.SetBasicAuth("john.doe", "hunter2")
	response, err := api.Router().Test(request, 10000)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Errorf("expected 200, got %d body=%s", response.StatusCode, string(body))
		return
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"healthy":true,"providers":null,"delivery":null}` && string(body) != `{"healthy":true,"providers":[],"delivery":[]}` {
		t.Errorf("unexpected body: %s", string(body))
	}
}