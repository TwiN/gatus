package opsgenie

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{APIKey: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	description := "my bad alert description"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:          "triggered",
			Provider:      AlertProvider{DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"}},
			Alert:         alert.Alert{Description: &description, SuccessThreshold: 1, FailureThreshold: 1},
			Resolved:      false,
			ExpectedError: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
		},
		{
			Name:          "triggered-error",
			Provider:      AlertProvider{DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"}},
			Alert:         alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      false,
			ExpectedError: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
		},
		{
			Name:          "resolved",
			Provider:      AlertProvider{DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"}},
			Alert:         alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      true,
			ExpectedError: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
		},
		{
			Name:          "resolved-error",
			Provider:      AlertProvider{DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"}},
			Alert:         alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      true,
			ExpectedError: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
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

func TestAlertProvider_buildCreateRequestBody(t *testing.T) {
	t.Parallel()
	description := "alert description"
	scenarios := []struct {
		Name     string
		Provider *AlertProvider
		Alert    *alert.Alert
		Endpoint *endpoint.Endpoint
		Result   *endpoint.Result
		Resolved bool
		want     alertCreateRequest
	}{
		{
			Name:     "missing all params (unresolved)",
			Provider: &AlertProvider{DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"}},
			Alert:    &alert.Alert{},
			Endpoint: &endpoint.Endpoint{},
			Result:   &endpoint.Result{},
			Resolved: false,
			want: alertCreateRequest{
				Message:     " - ",
				Priority:    "P1",
				Source:      "gatus",
				Entity:      "gatus-",
				Alias:       "gatus-healthcheck-",
				Description: "An alert for ** has been triggered due to having failed 0 time(s) in a row\n",
				Tags:        nil,
				Details:     map[string]string{},
			},
		},
		{
			Name:     "missing all params (resolved)",
			Provider: &AlertProvider{DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"}},
			Alert:    &alert.Alert{},
			Endpoint: &endpoint.Endpoint{},
			Result:   &endpoint.Result{},
			Resolved: true,
			want: alertCreateRequest{
				Message:     "RESOLVED:  - ",
				Priority:    "P1",
				Source:      "gatus",
				Entity:      "gatus-",
				Alias:       "gatus-healthcheck-",
				Description: "An alert for ** has been resolved after passing successfully 0 time(s) in a row\n",
				Tags:        nil,
				Details:     map[string]string{},
			},
		},
		{
			Name:     "with default options (unresolved)",
			Provider: &AlertProvider{DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"}},
			Alert: &alert.Alert{
				Description:      &description,
				FailureThreshold: 3,
			},
			Endpoint: &endpoint.Endpoint{
				Name: "my super app",
			},
			Result: &endpoint.Result{
				ConditionResults: []*endpoint.ConditionResult{
					{
						Condition: "[STATUS] == 200",
						Success:   true,
					},
					{
						Condition: "[BODY] == OK",
						Success:   false,
					},
				},
			},
			Resolved: false,
			want: alertCreateRequest{
				Message:     "my super app - " + description,
				Priority:    "P1",
				Source:      "gatus",
				Entity:      "gatus-my-super-app",
				Alias:       "gatus-healthcheck-my-super-app",
				Description: "An alert for *my super app* has been triggered due to having failed 3 time(s) in a row\n▣ - `[STATUS] == 200`\n▢ - `[BODY] == OK`\n",
				Tags:        nil,
				Details:     map[string]string{},
			},
		},
		{
			Name: "with custom options (resolved)",
			Provider: &AlertProvider{
				DefaultConfig: Config{
					Priority:     "P5",
					EntityPrefix: "oompa-",
					AliasPrefix:  "loompa-",
					Source:       "gatus-hc",
					Tags:         []string{"do-ba-dee-doo"},
				},
			},
			Alert: &alert.Alert{
				Description:      &description,
				SuccessThreshold: 4,
			},
			Endpoint: &endpoint.Endpoint{
				Name: "my mega app",
			},
			Result: &endpoint.Result{
				ConditionResults: []*endpoint.ConditionResult{
					{
						Condition: "[STATUS] == 200",
						Success:   true,
					},
				},
			},
			Resolved: true,
			want: alertCreateRequest{
				Message:     "RESOLVED: my mega app - " + description,
				Priority:    "P5",
				Source:      "gatus-hc",
				Entity:      "oompa-my-mega-app",
				Alias:       "loompa-my-mega-app",
				Description: "An alert for *my mega app* has been resolved after passing successfully 4 time(s) in a row\n▣ - `[STATUS] == 200`\n",
				Tags:        []string{"do-ba-dee-doo"},
				Details:     map[string]string{},
			},
		},
		{
			Name: "with default options and details (unresolved)",
			Provider: &AlertProvider{
				DefaultConfig: Config{Tags: []string{"foo"}, APIKey: "00000000-0000-0000-0000-000000000000"},
			},
			Alert: &alert.Alert{
				Description:      &description,
				FailureThreshold: 6,
			},
			Endpoint: &endpoint.Endpoint{
				Name:  "my app",
				Group: "end game",
				URL:   "https://my.go/app",
			},
			Result: &endpoint.Result{
				HTTPStatus: 400,
				Hostname:   "my.go",
				Errors:     []string{"error 01", "error 02"},
				Success:    false,
				ConditionResults: []*endpoint.ConditionResult{
					{
						Condition: "[STATUS] == 200",
						Success:   false,
					},
				},
			},
			Resolved: false,
			want: alertCreateRequest{
				Message:     "[end game] my app - " + description,
				Priority:    "P1",
				Source:      "gatus",
				Entity:      "gatus-end-game-my-app",
				Alias:       "gatus-healthcheck-end-game-my-app",
				Description: "An alert for *end game/my app* has been triggered due to having failed 6 time(s) in a row\n▢ - `[STATUS] == 200`\n",
				Tags:        []string{"foo"},
				Details: map[string]string{
					"endpoint:url":       "https://my.go/app",
					"endpoint:group":     "end game",
					"result:hostname":    "my.go",
					"result:errors":      "error 01,error 02",
					"result:http_status": "400",
				},
			},
		},
	}
	for _, scenario := range scenarios {
		actual := scenario
		t.Run(actual.Name, func(t *testing.T) {
			_ = scenario.Provider.Validate()
			if got := actual.Provider.buildCreateRequestBody(&scenario.Provider.DefaultConfig, actual.Endpoint, actual.Alert, actual.Result, actual.Resolved); !reflect.DeepEqual(got, actual.want) {
				t.Errorf("got:\n%v\nwant:\n%v", got, actual.want)
			}
		})
	}
}

func TestAlertProvider_buildCloseRequestBody(t *testing.T) {
	t.Parallel()
	description := "alert description"
	scenarios := []struct {
		Name     string
		Provider *AlertProvider
		Alert    *alert.Alert
		Endpoint *endpoint.Endpoint
		want     alertCloseRequest
	}{
		{
			Name:     "Missing all values",
			Provider: &AlertProvider{},
			Alert:    &alert.Alert{},
			Endpoint: &endpoint.Endpoint{},
			want: alertCloseRequest{
				Source: "",
				Note:   "RESOLVED:  - ",
			},
		},
		{
			Name:     "Basic values",
			Provider: &AlertProvider{},
			Alert: &alert.Alert{
				Description: &description,
			},
			Endpoint: &endpoint.Endpoint{
				Name: "endpoint name",
			},
			want: alertCloseRequest{
				Source: "endpoint-name",
				Note:   "RESOLVED: endpoint name - alert description",
			},
		},
	}
	for _, scenario := range scenarios {
		actual := scenario
		t.Run(actual.Name, func(t *testing.T) {
			if got := actual.Provider.buildCloseRequestBody(actual.Endpoint, actual.Alert); !reflect.DeepEqual(got, actual.want) {
				t.Errorf("buildCloseRequestBody() = %v, want %v", got, actual.want)
			}
		})
	}
}

func TestAlertProvider_GetConfig(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name: "provider-no-override-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"},
			},
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "00000000-0000-0000-0000-000000000000"},
		},
		{
			Name: "provider-with-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "00000000-0000-0000-0000-000000000000"},
			},
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"api-key": "00000000-0000-0000-0000-000000000001"}},
			ExpectedOutput: Config{APIKey: "00000000-0000-0000-0000-000000000001"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig("", &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.APIKey != scenario.ExpectedOutput.APIKey {
				t.Errorf("expected APIKey to be %s, got %s", scenario.ExpectedOutput.APIKey, got.APIKey)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides("", &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}
