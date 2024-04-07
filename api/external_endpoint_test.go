package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/discord"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

func TestCreateExternalEndpointResult(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Alerting: &alerting.Config{
			Discord: &discord.AlertProvider{},
		},
		ExternalEndpoints: []*core.ExternalEndpoint{
			{
				Name:  "n",
				Group: "g",
				Token: "token",
				Alerts: []*alert.Alert{
					{
						Type:             alert.TypeDiscord,
						FailureThreshold: 2,
						SuccessThreshold: 2,
					},
				},
			},
		},
		Maintenance: &maintenance.Config{},
	}
	api := New(cfg)
	router := api.Router()
	scenarios := []struct {
		Name                           string
		Path                           string
		AuthorizationHeaderBearerToken string
		ExpectedCode                   int
	}{
		{
			Name:                           "no-token",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			AuthorizationHeaderBearerToken: "",
			ExpectedCode:                   401,
		},
		{
			Name:                           "bad-token",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			AuthorizationHeaderBearerToken: "Bearer bad-token",
			ExpectedCode:                   401,
		},
		{
			Name:                           "bad-key",
			Path:                           "/api/v1/endpoints/bad_key/external?success=true",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   404,
		},
		{
			Name:                           "good-token-success-true",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   200,
		},
		{
			Name:                           "good-token-success-false",
			Path:                           "/api/v1/endpoints/g_n/external?success=false",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   200,
		},
		{
			Name:                           "good-token-success-false-again",
			Path:                           "/api/v1/endpoints/g_n/external?success=false",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   200,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request := httptest.NewRequest("POST", scenario.Path, http.NoBody)
			if len(scenario.AuthorizationHeaderBearerToken) > 0 {
				request.Header.Set("Authorization", scenario.AuthorizationHeaderBearerToken)
			}
			response, err := router.Test(request)
			if err != nil {
				return
			}
			defer response.Body.Close()
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}
		})
	}
	t.Run("verify-end-results", func(t *testing.T) {
		endpointStatus, err := store.Get().GetEndpointStatus("g", "n", paging.NewEndpointStatusParams().WithResults(1, 10))
		if err != nil {
			t.Errorf("failed to get endpoint status: %s", err.Error())
			return
		}
		if endpointStatus.Key != "g_n" {
			t.Errorf("expected key to be g_n but got %s", endpointStatus.Key)
		}
		if len(endpointStatus.Results) != 3 {
			t.Errorf("expected 3 results but got %d", len(endpointStatus.Results))
		}
		if !endpointStatus.Results[0].Success {
			t.Errorf("expected first result to be successful")
		}
		if endpointStatus.Results[1].Success {
			t.Errorf("expected second result to be unsuccessful")
		}
		if endpointStatus.Results[2].Success {
			t.Errorf("expected third result to be unsuccessful")
		}
		externalEndpointFromConfig := cfg.GetExternalEndpointByKey("g_n")
		if externalEndpointFromConfig.NumberOfFailuresInARow != 2 {
			t.Errorf("expected 2 failures in a row but got %d", externalEndpointFromConfig.NumberOfFailuresInARow)
		}
		if externalEndpointFromConfig.NumberOfSuccessesInARow != 0 {
			t.Errorf("expected 0 successes in a row but got %d", externalEndpointFromConfig.NumberOfSuccessesInARow)
		}
	})
}
