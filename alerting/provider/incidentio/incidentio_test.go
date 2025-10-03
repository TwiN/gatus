package incidentio

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name: "valid",
			provider: AlertProvider{
				DefaultConfig: Config{
					URL:       "https://api.incident.io/v2/alert_events/http/some-id",
					AuthToken: "some-token",
				},
			},
			expected: true,
		},
		{
			name: "invalid-url",
			provider: AlertProvider{
				DefaultConfig: Config{
					URL:       "id-without-rest-api-url-as-prefix",
					AuthToken: "some-token",
				},
			},
			expected: false,
		},
		{
			name: "invalid-missing-auth-token",
			provider: AlertProvider{
				DefaultConfig: Config{
					URL: "some-id",
				},
			},
			expected: false,
		},
		{
			name: "invalid-missing-alert-source-config-id",
			provider: AlertProvider{
				DefaultConfig: Config{
					AuthToken: "some-token",
				},
			},
			expected: false,
		},
		{
			name: "valid-override",
			provider: AlertProvider{
				DefaultConfig: Config{
					AuthToken: "some-token",
					URL:       "https://api.incident.io/v2/alert_events/http/some-id",
				},
				Overrides: []Override{{Group: "core", Config: Config{URL: "https://api.incident.io/v2/alert_events/http/another-id"}}},
			},
			expected: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.provider.Validate()
			if scenario.expected && err != nil {
				t.Error("expected no error, got", err.Error())
			}
			if !scenario.expected && err == nil {
				t.Error("expected error, got none")
			}
		})
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	firstDescription := "description-1"
	secondDescription := "description-2"
	restAPIUrl := "https://api.incident.io/v2/alert_events/http/"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name: "triggered",
			Provider: AlertProvider{DefaultConfig: Config{
				URL:       restAPIUrl + "some-id",
				AuthToken: "some-token",
			}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				var b bytes.Buffer

				response := Response{DeduplicationKey: "some-key"}
				json.NewEncoder(&b).Encode(response)
				reader := io.NopCloser(&b)
				return &http.Response{StatusCode: http.StatusAccepted, Body: reader}
			}),
			ExpectedError: false,
		},
		{
			Name: "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{
				URL:       restAPIUrl + "some-id",
				AuthToken: "some-token",
			}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name: "resolved",
			Provider: AlertProvider{DefaultConfig: Config{
				URL:       restAPIUrl + "some-id",
				AuthToken: "some-token",
			}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				var b bytes.Buffer
				response := Response{DeduplicationKey: "some-key"}
				json.NewEncoder(&b).Encode(response)
				reader := io.NopCloser(&b)
				return &http.Response{StatusCode: http.StatusAccepted, Body: reader}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err := scenario.Provider.Send(
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if scenario.ExpectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectedError && err != nil {
				t.Error("expected no error, got", err.Error())
			}
		})
	}
}

func TestAlertProvider_BuildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	restAPIUrl := "https://api.incident.io/v2/alert_events/http/"
	scenarios := []struct {
		Name                     string
		Provider                 AlertProvider
		Alert                    alert.Alert
		Resolved                 bool
		ExpectedAlertSourceID    string
		ExpectedStatus           string
		ExpectedTitle            string
		ExpectedDescription      string
		ExpectedSourceURL        string
		ExpectedMetadata         map[string]interface{}
		ShouldHaveDeduplicationKey bool
	}{
		{
			Name:                     "triggered",
			Provider:                 AlertProvider{DefaultConfig: Config{URL: restAPIUrl + "some-id", AuthToken: "some-token"}},
			Alert:                    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:                 false,
			ExpectedAlertSourceID:    "some-id",
			ExpectedStatus:           "firing",
			ExpectedTitle:            "Gatus: endpoint-name",
			ExpectedDescription:      "An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1 and the following conditions:  🔴 [CONNECTED] == true  🔴 [STATUS] == 200  ",
			ShouldHaveDeduplicationKey: true,
		},
		{
			Name:                     "resolved",
			Provider:                 AlertProvider{DefaultConfig: Config{URL: restAPIUrl + "some-id", AuthToken: "some-token"}},
			Alert:                    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:                 true,
			ExpectedAlertSourceID:    "some-id",
			ExpectedStatus:           "resolved",
			ExpectedTitle:            "Gatus: endpoint-name",
			ExpectedDescription:      "An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-2 and the following conditions:  🟢 [CONNECTED] == true  🟢 [STATUS] == 200  ",
			ShouldHaveDeduplicationKey: true,
		},
		{
			Name:                     "resolved-with-metadata-source-url",
			Provider:                 AlertProvider{DefaultConfig: Config{URL: restAPIUrl + "some-id", AuthToken: "some-token", Metadata: map[string]interface{}{"service": "some-service", "team": "very-core"}, SourceURL: "some-source-url"}},
			Alert:                    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:                 true,
			ExpectedAlertSourceID:    "some-id",
			ExpectedStatus:           "resolved",
			ExpectedTitle:            "Gatus: endpoint-name",
			ExpectedDescription:      "An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-2 and the following conditions:  🟢 [CONNECTED] == true  🟢 [STATUS] == 200  ",
			ExpectedSourceURL:        "some-source-url",
			ExpectedMetadata:         map[string]interface{}{"service": "some-service", "team": "very-core"},
			ShouldHaveDeduplicationKey: true,
		},
		{
			Name:                     "group-override",
			Provider:                 AlertProvider{DefaultConfig: Config{URL: restAPIUrl + "some-id", AuthToken: "some-token"}, Overrides: []Override{{Group: "g", Config: Config{URL: restAPIUrl + "different-id", AuthToken: "some-token"}}}},
			Alert:                    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:                 false,
			ExpectedAlertSourceID:    "different-id",
			ExpectedStatus:           "firing",
			ExpectedTitle:            "Gatus: endpoint-name",
			ExpectedDescription:      "An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1 and the following conditions:  🔴 [CONNECTED] == true  🔴 [STATUS] == 200  ",
			ShouldHaveDeduplicationKey: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg, err := scenario.Provider.GetConfig("g", &scenario.Alert)
			if err != nil {
				t.Error("expected no error, got", err.Error())
			}
			body := scenario.Provider.buildRequestBody(
				cfg,
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			
			// Parse the JSON body
			var parsedBody Body
			if err := json.Unmarshal(body, &parsedBody); err != nil {
				t.Error("expected body to be valid JSON, got error:", err.Error())
			}
			
			// Validate individual fields
			if parsedBody.AlertSourceConfigID != scenario.ExpectedAlertSourceID {
				t.Errorf("expected alert_source_config_id to be %s, got %s", scenario.ExpectedAlertSourceID, parsedBody.AlertSourceConfigID)
			}
			if parsedBody.Status != scenario.ExpectedStatus {
				t.Errorf("expected status to be %s, got %s", scenario.ExpectedStatus, parsedBody.Status)
			}
			if parsedBody.Title != scenario.ExpectedTitle {
				t.Errorf("expected title to be %s, got %s", scenario.ExpectedTitle, parsedBody.Title)
			}
			if parsedBody.Description != scenario.ExpectedDescription {
				t.Errorf("expected description to be %s, got %s", scenario.ExpectedDescription, parsedBody.Description)
			}
			if scenario.ExpectedSourceURL != "" && parsedBody.SourceURL != scenario.ExpectedSourceURL {
				t.Errorf("expected source_url to be %s, got %s", scenario.ExpectedSourceURL, parsedBody.SourceURL)
			}
			if scenario.ExpectedMetadata != nil {
				metadataJSON, _ := json.Marshal(parsedBody.Metadata)
				expectedMetadataJSON, _ := json.Marshal(scenario.ExpectedMetadata)
				if string(metadataJSON) != string(expectedMetadataJSON) {
					t.Errorf("expected metadata to be %s, got %s", string(expectedMetadataJSON), string(metadataJSON))
				}
			}
			// Validate that deduplication_key exists and is not empty
			if scenario.ShouldHaveDeduplicationKey {
				if parsedBody.DeduplicationKey == "" {
					t.Error("expected deduplication_key to be present and non-empty")
				}
			}
		})
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

func TestAlertProvider_GetConfig(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "https://api.incident.io/v2/alert_events/http/diff-id"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "https://api.incident.io/v2/alert_events/http/diff-id", AuthToken: "some-token"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://api.incident.io/v2/alert_events/http/diff-id", AuthToken: "some-token"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "https://api.incident.io/v2/alert_events/http/diff-id", AuthToken: "some-token"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"url": "https://api.incident.io/v2/alert_events/http/another-id"}},
			ExpectedOutput: Config{URL: "https://api.incident.io/v2/alert_events/http/another-id", AuthToken: "some-token"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.URL != scenario.ExpectedOutput.URL {
				t.Errorf("expected alert source config to be %s, got %s", scenario.ExpectedOutput.URL, got.URL)
			}
			if got.AuthToken != scenario.ExpectedOutput.AuthToken {
				t.Errorf("expected alert auth token to be %s, got %s", scenario.ExpectedOutput.AuthToken, got.AuthToken)
			}

			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{URL: "https://api.incident.io/v2/alert_events/http/some-id", AuthToken: "some-token"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{URL: "", AuthToken: "some-token"},
				Group:  "group",
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{URL: "https://api.incident.io/v2/alert_events/http/nice-id", AuthToken: "some-token"},
		Overrides: []Override{
			{
				Config: Config{URL: "https://api.incident.io/v2/alert_events/http/very-good-id", AuthToken: "some-token"},
				Group:  "group",
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}
