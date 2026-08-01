package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/discord"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/maintenance"
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
		ExternalEndpoints: []*endpoint.ExternalEndpoint{
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
		Body                           string
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
			Name:                           "bad-success-value",
			Path:                           "/api/v1/endpoints/g_n/external?success=invalid",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   400,
		},
		{
			Name:                           "bad-duration-value",
			Path:                           "/api/v1/endpoints/g_n/external?success=true&duration=invalid",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   400,
		},
		{
			Name:                           "good-token-success-true",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   200,
		},
		{
			Name:                           "good-token-success-true-with-ignored-error-because-success-true",
			Path:                           "/api/v1/endpoints/g_n/external?success=true&error=failed",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   200,
		},
		{
			Name:                           "good-duration-success-true",
			Path:                           "/api/v1/endpoints/g_n/external?success=true&duration=10s",
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
		{
			Name:                           "good-token-success-false-with-error",
			Path:                           "/api/v1/endpoints/g_n/external?success=false&error=failed",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   200,
		},
		{
			Name:                           "malformed-body",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			Body:                           "{",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   400,
		},
		{
			Name:                           "body-is-not-an-object",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			Body:                           "[]",
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   400,
		},
		{
			Name:                           "null-condition-result",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			Body:                           `{"conditionResults":[null]}`,
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   400,
		},
		{
			Name:                           "blank-condition",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			Body:                           `{"conditionResults":[{"condition":"  ","success":true}]}`,
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   400,
		},
		{
			Name:                           "condition-too-long",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			Body:                           `{"conditionResults":[{"condition":"` + strings.Repeat("a", maximumConditionLengthPerPush+1) + `","success":true}]}`,
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   400,
		},
		{
			Name:                           "too-many-condition-results",
			Path:                           "/api/v1/endpoints/g_n/external?success=true",
			Body:                           `{"conditionResults":[` + strings.Repeat(`{"condition":"[STATUS] == 200","success":true},`, maximumNumberOfConditionResultsPerPush) + `{"condition":"[STATUS] == 200","success":true}]}`,
			AuthorizationHeaderBearerToken: "Bearer token",
			ExpectedCode:                   400,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var body io.Reader = http.NoBody
			if len(scenario.Body) > 0 {
				body = strings.NewReader(scenario.Body)
			}
			request := httptest.NewRequest("POST", scenario.Path, body)
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
		endpointStatus, err := store.Get().GetEndpointStatus("g", "n", paging.NewEndpointStatusParams().WithResults(1, 11))
		if err != nil {
			t.Errorf("failed to get endpoint status: %s", err.Error())
			return
		}
		if endpointStatus.Key != "g_n" {
			t.Errorf("expected key to be g_n but got %s", endpointStatus.Key)
		}
		if len(endpointStatus.Results) != 6 {
			t.Errorf("expected 6 results but got %d", len(endpointStatus.Results))
		}
		if !endpointStatus.Results[0].Success {
			t.Errorf("expected first result to be successful")
		}
		if !endpointStatus.Results[1].Success {
			t.Errorf("expected second result to be successful")
		}
		if len(endpointStatus.Results[1].Errors) > 0 {
			t.Errorf("expected second result to have no errors")
		}
		if endpointStatus.Results[2].Duration == 0 || endpointStatus.Results[2].Duration.Seconds() != 10 {
			t.Errorf("expected third result to have a duration of 10 seconds")
		}
		if endpointStatus.Results[3].Success {
			t.Errorf("expected fourth result to be unsuccessful")
		}
		if endpointStatus.Results[4].Success {
			t.Errorf("expected fifth result to be unsuccessful")
		}
		if endpointStatus.Results[5].Success {
			t.Errorf("expected sixth result to be unsuccessful")
		}
		if len(endpointStatus.Results[5].Errors) == 0 || endpointStatus.Results[5].Errors[0] != "failed" {
			t.Errorf("expected sixth result to have errors: failed")
		}
		externalEndpointFromConfig := cfg.GetExternalEndpointByKey("g_n")
		if externalEndpointFromConfig.NumberOfFailuresInARow != 3 {
			t.Errorf("expected 3 failures in a row but got %d", externalEndpointFromConfig.NumberOfFailuresInARow)
		}
		if externalEndpointFromConfig.NumberOfSuccessesInARow != 0 {
			t.Errorf("expected 0 successes in a row but got %d", externalEndpointFromConfig.NumberOfSuccessesInARow)
		}
	})
}

func TestCreateExternalEndpointResultWithConditionResults(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Alerting: &alerting.Config{
			Discord: &discord.AlertProvider{},
		},
		ExternalEndpoints: []*endpoint.ExternalEndpoint{
			{
				Name:  "n",
				Group: "g",
				Token: "token",
			},
		},
		Maintenance: &maintenance.Config{},
	}
	api := New(cfg)
	router := api.Router()
	push := func(t *testing.T, path, body string) {
		t.Helper()
		var requestBody io.Reader = http.NoBody
		if len(body) > 0 {
			requestBody = strings.NewReader(body)
		}
		request := httptest.NewRequest("POST", path, requestBody)
		request.Header.Set("Authorization", "Bearer token")
		response, err := router.Test(request)
		if err != nil {
			t.Fatalf("failed to push result: %s", err.Error())
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatalf("%s %s should have returned 200, but returned %d instead", request.Method, request.URL, response.StatusCode)
		}
	}
	// Even though one of the conditions is successful, the result must remain unsuccessful,
	// because success is determined by the query parameter and not by the condition results.
	push(t, "/api/v1/endpoints/g_n/external?success=false", `{"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] < 300","success":false}],"somethingElse":1}`)
	push(t, "/api/v1/endpoints/g_n/external?success=true", "")
	push(t, "/api/v1/endpoints/g_n/external?success=true", "{}")
	endpointStatus, err := store.Get().GetEndpointStatusByKey("g_n", paging.NewEndpointStatusParams().WithResults(1, 10))
	if err != nil {
		t.Fatalf("failed to get endpoint status: %s", err.Error())
	}
	if len(endpointStatus.Results) != 3 {
		t.Fatalf("expected 3 results but got %d", len(endpointStatus.Results))
	}
	if endpointStatus.Results[0].Success {
		t.Error("expected first result to be unsuccessful, because success comes from the query parameter")
	}
	if len(endpointStatus.Results[0].ConditionResults) != 2 {
		t.Fatalf("expected first result to have 2 condition results but got %d", len(endpointStatus.Results[0].ConditionResults))
	}
	if endpointStatus.Results[0].ConditionResults[0].Condition != "[STATUS] == 200" || !endpointStatus.Results[0].ConditionResults[0].Success {
		t.Errorf("expected first condition result to be '[STATUS] == 200' and successful, but got '%s' and %v", endpointStatus.Results[0].ConditionResults[0].Condition, endpointStatus.Results[0].ConditionResults[0].Success)
	}
	if endpointStatus.Results[0].ConditionResults[1].Condition != "[RESPONSE_TIME] < 300" || endpointStatus.Results[0].ConditionResults[1].Success {
		t.Errorf("expected second condition result to be '[RESPONSE_TIME] < 300' and unsuccessful, but got '%s' and %v", endpointStatus.Results[0].ConditionResults[1].Condition, endpointStatus.Results[0].ConditionResults[1].Success)
	}
	if len(endpointStatus.Results[1].ConditionResults) != 0 {
		t.Errorf("expected second result to have no condition results, because no body was sent")
	}
	if len(endpointStatus.Results[2].ConditionResults) != 0 {
		t.Errorf("expected third result to have no condition results, because the body was empty")
	}
}
