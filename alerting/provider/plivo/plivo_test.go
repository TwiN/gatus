package plivo

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestPlivoAlertProvider_IsValid(t *testing.T) {
	scenarios := []struct {
		Name          string
		Provider      AlertProvider
		ExpectedError error
	}{
		{
			Name:          "invalid-provider-missing-config",
			Provider:      AlertProvider{},
			ExpectedError: ErrAuthIDNotSet,
		},
		{
			Name: "valid-provider",
			Provider: AlertProvider{
				DefaultConfig: Config{
					AuthID:    "1",
					AuthToken: "1",
					From:      "1234567890",
					To:        []string{"0987654321"},
				},
			},
			ExpectedError: nil,
		},
		{
			Name: "valid-provider-with-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					AuthID:    "1",
					AuthToken: "1",
					From:      "1234567890",
					To:        []string{"0987654321"},
				},
				Overrides: []Override{
					{
						Group:  "group1",
						Config: Config{AuthID: "2", AuthToken: "2", From: "2222222222", To: []string{"3333333333"}},
					},
				},
			},
			ExpectedError: nil,
		},
		{
			Name: "invalid-provider-duplicate-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					AuthID:    "1",
					AuthToken: "1",
					From:      "1234567890",
					To:        []string{"0987654321"},
				},
				Overrides: []Override{
					{
						Group:  "group1",
						Config: Config{AuthID: "2", AuthToken: "2", From: "2222222222", To: []string{"3333333333"}},
					},
					{
						Group:  "group1",
						Config: Config{AuthID: "3", AuthToken: "3", From: "4444444444", To: []string{"5555555555"}},
					},
				},
			},
			ExpectedError: ErrDuplicateGroupOverride,
		},
		{
			Name: "invalid-provider-empty-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					AuthID:    "1",
					AuthToken: "1",
					From:      "1234567890",
					To:        []string{"0987654321"},
				},
				Overrides: []Override{
					{
						Group:  "",
						Config: Config{AuthID: "2", AuthToken: "2", From: "2222222222", To: []string{"3333333333"}},
					},
				},
			},
			ExpectedError: ErrDuplicateGroupOverride,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Provider.Validate()
			if scenario.ExpectedError == nil && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if scenario.ExpectedError != nil && err == nil {
				t.Errorf("expected error %v, got none", scenario.ExpectedError)
			}
			if scenario.ExpectedError != nil && err != nil && err.Error() != scenario.ExpectedError.Error() {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
		})
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:     "triggered",
			Provider: AlertProvider{DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "multiple-recipients",
			Provider: AlertProvider{DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321", "1122334455"}}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			ExpectedError: false,
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

func TestAlertProvider_buildMessage(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name            string
		Provider        AlertProvider
		Alert           alert.Alert
		Resolved        bool
		ExpectedMessage string
	}{
		{
			Name:            "triggered",
			Provider:        AlertProvider{DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}}},
			Alert:           alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			ExpectedMessage: "TRIGGERED: endpoint-name - description-1",
		},
		{
			Name:            "resolved",
			Provider:        AlertProvider{DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}}},
			Alert:           alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        true,
			ExpectedMessage: "RESOLVED: endpoint-name - description-2",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			message := scenario.Provider.buildMessage(
				&scenario.Provider.DefaultConfig,
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
			if message != scenario.ExpectedMessage {
				t.Errorf("expected %s, got %s", scenario.ExpectedMessage, message)
			}
		})
	}
}

func TestAlertProvider_sendSMS(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	cfg := &Config{
		AuthID:    "test-auth-id",
		AuthToken: "test-auth-token",
		From:      "1234567890",
	}
	scenarios := []struct {
		Name             string
		To               string
		Message          string
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:    "successful-sms",
			To:      "0987654321",
			Message: "Test message",
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				// Verify request structure
				body, _ := io.ReadAll(r.Body)
				var payload map[string]string
				json.Unmarshal(body, &payload)
				if payload["src"] != cfg.From {
					t.Errorf("expected src %s, got %s", cfg.From, payload["src"])
				}
				if payload["dst"] != "0987654321" {
					t.Errorf("expected dst %s, got %s", "0987654321", payload["dst"])
				}
				if payload["text"] != "Test message" {
					t.Errorf("expected text %s, got %s", "Test message", payload["text"])
				}
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:    "failed-sms",
			To:      "0987654321",
			Message: "Test message",
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusBadRequest, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			provider := AlertProvider{}
			err := provider.sendSMS(cfg, scenario.To, scenario.Message)
			if scenario.ExpectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectedError && err != nil {
				t.Error("expected no error, got", err.Error())
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
			Name: "provider-no-override-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}},
		},
		{
			Name: "provider-with-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}},
				Overrides: []Override{
					{
						Group:  "group1",
						Config: Config{AuthID: "3", AuthToken: "4", From: "3333333333", To: []string{"7777777777"}},
					},
				},
			},
			InputGroup:     "group1",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{AuthID: "3", AuthToken: "4", From: "3333333333", To: []string{"7777777777"}},
		},
		{
			Name: "provider-with-group-override-no-match",
			Provider: AlertProvider{
				DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}},
				Overrides: []Override{
					{
						Group:  "group1",
						Config: Config{AuthID: "3", AuthToken: "4", From: "3333333333", To: []string{"7777777777"}},
					},
				},
			},
			InputGroup:     "group2",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}},
		},
		{
			Name: "provider-with-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"auth-id": "5", "auth-token": "6", "from": "5555555555", "to": []string{"9999999999"}}},
			ExpectedOutput: Config{AuthID: "5", AuthToken: "6", From: "5555555555", To: []string{"9999999999"}},
		},
		{
			Name: "provider-with-group-and-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{AuthID: "1", AuthToken: "2", From: "1234567890", To: []string{"0987654321"}},
				Overrides: []Override{
					{
						Group:  "group1",
						Config: Config{AuthID: "3", AuthToken: "4", From: "3333333333", To: []string{"7777777777"}},
					},
				},
			},
			InputGroup:     "group1",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"auth-id": "5", "auth-token": "6"}},
			ExpectedOutput: Config{AuthID: "5", AuthToken: "6", From: "3333333333", To: []string{"7777777777"}},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Error("expected no error, got:", err.Error())
			}
			if got.AuthID != scenario.ExpectedOutput.AuthID {
				t.Errorf("expected AuthID to be %s, got %s", scenario.ExpectedOutput.AuthID, got.AuthID)
			}
			if got.AuthToken != scenario.ExpectedOutput.AuthToken {
				t.Errorf("expected AuthToken to be %s, got %s", scenario.ExpectedOutput.AuthToken, got.AuthToken)
			}
			if got.From != scenario.ExpectedOutput.From {
				t.Errorf("expected From to be %s, got %s", scenario.ExpectedOutput.From, got.From)
			}
			if len(got.To) != len(scenario.ExpectedOutput.To) {
				t.Errorf("expected To length to be %d, got %d", len(scenario.ExpectedOutput.To), len(got.To))
			}
			for i, to := range got.To {
				if to != scenario.ExpectedOutput.To[i] {
					t.Errorf("expected To[%d] to be %s, got %s", i, scenario.ExpectedOutput.To[i], to)
				}
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	scenarios := []struct {
		Name          string
		Config        Config
		ExpectedError error
	}{
		{
			Name: "valid-config",
			Config: Config{
				AuthID:    "test-auth-id",
				AuthToken: "test-auth-token",
				From:      "1234567890",
				To:        []string{"0987654321"},
			},
			ExpectedError: nil,
		},
		{
			Name: "missing-auth-id",
			Config: Config{
				AuthToken: "test-auth-token",
				From:      "1234567890",
				To:        []string{"0987654321"},
			},
			ExpectedError: ErrAuthIDNotSet,
		},
		{
			Name: "missing-auth-token",
			Config: Config{
				AuthID: "test-auth-id",
				From:   "1234567890",
				To:     []string{"0987654321"},
			},
			ExpectedError: ErrAuthTokenNotSet,
		},
		{
			Name: "missing-from",
			Config: Config{
				AuthID:    "test-auth-id",
				AuthToken: "test-auth-token",
				To:        []string{"0987654321"},
			},
			ExpectedError: ErrFromNotSet,
		},
		{
			Name: "missing-to",
			Config: Config{
				AuthID:    "test-auth-id",
				AuthToken: "test-auth-token",
				From:      "1234567890",
			},
			ExpectedError: ErrToNotSet,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Config.Validate()
			if scenario.ExpectedError == nil && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if scenario.ExpectedError != nil && err == nil {
				t.Errorf("expected error %v, got none", scenario.ExpectedError)
			}
			if scenario.ExpectedError != nil && err != nil && err.Error() != scenario.ExpectedError.Error() {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	cfg := Config{
		AuthID:    "original-auth-id",
		AuthToken: "original-auth-token",
		From:      "1111111111",
		To:        []string{"2222222222"},
	}
	override := Config{
		AuthID:    "override-auth-id",
		AuthToken: "override-auth-token",
		From:      "3333333333",
		To:        []string{"4444444444", "5555555555"},
	}
	cfg.Merge(&override)
	if cfg.AuthID != "override-auth-id" {
		t.Errorf("expected AuthID to be %s, got %s", "override-auth-id", cfg.AuthID)
	}
	if cfg.AuthToken != "override-auth-token" {
		t.Errorf("expected AuthToken to be %s, got %s", "override-auth-token", cfg.AuthToken)
	}
	if cfg.From != "3333333333" {
		t.Errorf("expected From to be %s, got %s", "3333333333", cfg.From)
	}
	if len(cfg.To) != 2 || cfg.To[0] != "4444444444" || cfg.To[1] != "5555555555" {
		t.Errorf("expected To to be [4444444444, 5555555555], got %v", cfg.To)
	}
}
