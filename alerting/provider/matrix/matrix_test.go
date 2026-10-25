package matrix

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/ui"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{
		DefaultConfig: Config{
			AccessToken:    "",
			InternalRoomID: "",
		},
	}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{
		DefaultConfig: Config{
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
	}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
	validProviderWithHomeserver := AlertProvider{
		DefaultConfig: Config{
			ServerURL:      "https://example.com",
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
	}
	if err := validProviderWithHomeserver.Validate(); err != nil {
		t.Error("provider with homeserver should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Group: "",
				Config: Config{
					AccessToken:    "",
					InternalRoomID: "",
				},
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				Group: "group",
				Config: Config{
					AccessToken:    "",
					InternalRoomID: "",
				},
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
		Overrides: []Override{
			{
				Group: "group",
				Config: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
		t.Error("provider should've been valid")
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
			Name:     "triggered-with-bad-config",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "triggered",
			Provider: AlertProvider{DefaultConfig: Config{AccessToken: "1", InternalRoomID: "!a:example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{AccessToken: "1", InternalRoomID: "!a:example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{AccessToken: "1", InternalRoomID: "!a:example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{AccessToken: "1", InternalRoomID: "!a:example.com"}},
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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	escapedDescription := "fails & shows <error>"
	scenarios := []struct {
		Name                  string
		Provider              AlertProvider
		Endpoint              endpoint.Endpoint
		Alert                 alert.Alert
		Result                endpoint.Result
		Resolved              bool
		ExpectedBody          string
		ExpectedFormattedBody string
	}{
		{
			Name:     "triggered",
			Provider: AlertProvider{},
			Endpoint: endpoint.Endpoint{Name: "endpoint-name"},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Result: endpoint.Result{ConditionResults: []*endpoint.ConditionResult{
				{Condition: "[CONNECTED] == true", Success: false},
				{Condition: "[STATUS] == 200", Success: false},
			}},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: endpoint-name (failed 3 time(s) in a row)\n\ndescription-1\n\n✕ - [CONNECTED] == true\n✕ - [STATUS] == 200",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: endpoint-name (failed 3 time(s) in a row)</p>\n<blockquote>description-1</blockquote>\n<h5>Condition results</h5><ul><li>❌ - <code>[CONNECTED] == true</code></li><li>❌ - <code>[STATUS] == 200</code></li></ul>",
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{},
			Endpoint: endpoint.Endpoint{Name: "endpoint-name"},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Result: endpoint.Result{ConditionResults: []*endpoint.ConditionResult{
				{Condition: "[CONNECTED] == true", Success: true},
				{Condition: "[STATUS] == 200", Success: true},
			}},
			Resolved:              true,
			ExpectedBody:          "💚 RESOLVED\nEndpoint: endpoint-name (passed 5 time(s) in a row)\n\ndescription-2\n\n✓ - [CONNECTED] == true\n✓ - [STATUS] == 200",
			ExpectedFormattedBody: "<p>💚 <b>RESOLVED</b><br>\nEndpoint: endpoint-name (passed 5 time(s) in a row)</p>\n<blockquote>description-2</blockquote>\n<h5>Condition results</h5><ul><li>✅ - <code>[CONNECTED] == true</code></li><li>✅ - <code>[STATUS] == 200</code></li></ul>",
		},
		{
			Name:     "triggered-with-errors-escaped-and-link",
			Provider: AlertProvider{},
			Endpoint: endpoint.Endpoint{Name: "api", Group: "core", URL: "https://example.org/health"},
			Alert:    alert.Alert{Description: &escapedDescription, FailureThreshold: 1},
			Result: endpoint.Result{
				ConditionResults: []*endpoint.ConditionResult{
					{Condition: "[STATUS] (0) == 200", Success: false},
					{Condition: "[RESPONSE_TIME] < 300", Success: true},
				},
				Errors: []string{"dial tcp: lookup example.org: no such host"},
			},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: core/api (failed 1 time(s) in a row)\n\nfails & shows <error>\n\n✕ - [STATUS] (0) == 200\n✓ - [RESPONSE_TIME] < 300\n\ndial tcp: lookup example.org: no such host",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: <a href=\"https://example.org/health\">core/api</a> (failed 1 time(s) in a row)</p>\n<blockquote>fails &amp; shows &lt;error&gt;</blockquote>\n<h5>Condition results</h5><ul><li>❌ - <code>[STATUS] (0) == 200</code></li><li>✅ - <code>[RESPONSE_TIME] &lt; 300</code></li></ul>\n<pre><code>dial tcp: lookup example.org: no such host</code></pre>",
		},
		{
			Name:                  "errors-only",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "external"},
			Alert:                 alert.Alert{FailureThreshold: 2},
			Result:                endpoint.Result{Errors: []string{"timeout"}},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: external (failed 2 time(s) in a row)\n\ntimeout",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: external (failed 2 time(s) in a row)</p>\n<pre><code>timeout</code></pre>",
		},
		{
			Name:                  "external-endpoint-heartbeat-resolved-without-conditions",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "job", Group: "cron"},
			Alert:                 alert.Alert{SuccessThreshold: 1},
			Result:                endpoint.Result{Success: true},
			Resolved:              true,
			ExpectedBody:          "💚 RESOLVED\nEndpoint: cron/job (passed 1 time(s) in a row)",
			ExpectedFormattedBody: "<p>💚 <b>RESOLVED</b><br>\nEndpoint: cron/job (passed 1 time(s) in a row)</p>",
		},
		{
			Name:                  "multiple-errors",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "job"},
			Alert:                 alert.Alert{FailureThreshold: 1},
			Result:                endpoint.Result{Errors: []string{"heartbeat: no update received within 5m0s", "second <error>"}},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: job (failed 1 time(s) in a row)\n\nheartbeat: no update received within 5m0s\nsecond <error>",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: job (failed 1 time(s) in a row)</p>\n<pre><code>heartbeat: no update received within 5m0s\nsecond &lt;error&gt;</code></pre>",
		},
		{
			Name:                  "hidden-url-is-not-linked",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "secret", URL: "https://example.org/hook/abc", UIConfig: &ui.Config{HideURL: true}},
			Alert:                 alert.Alert{FailureThreshold: 1},
			Result:                endpoint.Result{Errors: []string{`Get "<redacted>": timeout`}},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: secret (failed 1 time(s) in a row)\n\nGet \"<redacted>\": timeout",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: secret (failed 1 time(s) in a row)</p>\n<pre><code>Get &#34;&lt;redacted&gt;&#34;: timeout</code></pre>",
		},
		{
			Name:                  "hidden-hostname-is-not-linked",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "internal", URL: "https://internal.example.org/health", UIConfig: &ui.Config{HideHostname: true}},
			Alert:                 alert.Alert{FailureThreshold: 1},
			Result:                endpoint.Result{},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: internal (failed 1 time(s) in a row)",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: internal (failed 1 time(s) in a row)</p>",
		},
		{
			Name:                  "url-with-credentials-is-not-linked",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "with-secrets", URL: "https://user:secret@example.org/health"},
			Alert:                 alert.Alert{FailureThreshold: 1},
			Result:                endpoint.Result{},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: with-secrets (failed 1 time(s) in a row)",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: with-secrets (failed 1 time(s) in a row)</p>",
		},
		{
			Name:                  "url-with-query-is-not-linked",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "dashboard", URL: "https://grafana.example.org/d/abc?orgId=2"},
			Alert:                 alert.Alert{FailureThreshold: 1},
			Result:                endpoint.Result{},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: dashboard (failed 1 time(s) in a row)",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: dashboard (failed 1 time(s) in a row)</p>",
		},
		{
			Name:                  "url-with-fragment-is-not-linked",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "spa", URL: "https://app.example.org/#/status"},
			Alert:                 alert.Alert{FailureThreshold: 1},
			Result:                endpoint.Result{},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: spa (failed 1 time(s) in a row)",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: spa (failed 1 time(s) in a row)</p>",
		},
		{
			Name:                  "non-http-endpoint-is-not-linked",
			Provider:              AlertProvider{},
			Endpoint:              endpoint.Endpoint{Name: "db", URL: "tcp://10.0.0.1:5432"},
			Alert:                 alert.Alert{FailureThreshold: 1},
			Result:                endpoint.Result{ConditionResults: []*endpoint.ConditionResult{{Condition: "[CONNECTED] (false) == true"}}},
			Resolved:              false,
			ExpectedBody:          "💔 ALERT\nEndpoint: db (failed 1 time(s) in a row)\n\n✕ - [CONNECTED] (false) == true",
			ExpectedFormattedBody: "<p>💔 <b>ALERT</b><br>\nEndpoint: db (failed 1 time(s) in a row)</p>\n<h5>Condition results</h5><ul><li>❌ - <code>[CONNECTED] (false) == true</code></li></ul>",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(&scenario.Endpoint, &scenario.Alert, &scenario.Result, scenario.Resolved)
			var out Body
			if err := json.Unmarshal(body, &out); err != nil {
				t.Fatal("expected body to be valid JSON, got error:", err.Error())
			}
			if out.MsgType != "m.text" || out.Format != "org.matrix.custom.html" {
				t.Errorf("unexpected msgtype/format: %s/%s", out.MsgType, out.Format)
			}
			if out.Body != scenario.ExpectedBody {
				t.Errorf("expected body:\n%s\ngot:\n%s", scenario.ExpectedBody, out.Body)
			}
			if out.FormattedBody != scenario.ExpectedFormattedBody {
				t.Errorf("expected formatted_body:\n%s\ngot:\n%s", scenario.ExpectedFormattedBody, out.FormattedBody)
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
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: nil,
			},
			InputGroup: "",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: nil,
			},
			InputGroup: "group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: []Override{
					{
						Group: "group",
						Config: Config{
							ServerURL:      "https://group-example.com",
							AccessToken:    "12",
							InternalRoomID: "!a:group-example.com",
						},
					},
				},
			},
			InputGroup: "",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: []Override{
					{
						Group: "group",
						Config: Config{
							ServerURL:      "https://group-example.com",
							AccessToken:    "12",
							InternalRoomID: "!a:group-example.com",
						},
					},
				},
			},
			InputGroup: "group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				ServerURL:      "https://group-example.com",
				AccessToken:    "12",
				InternalRoomID: "!a:group-example.com",
			},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: []Override{
					{
						Group: "group",
						Config: Config{
							ServerURL:      "https://group-example.com",
							AccessToken:    "12",
							InternalRoomID: "!a:example01.com",
						},
					},
				},
			},
			InputGroup: "group",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{"server-url": "https://alert-example.com", "access-token": "123", "internal-room-id": "!a:alert-example.com"}},
			ExpectedOutput: Config{
				ServerURL:      "https://alert-example.com",
				AccessToken:    "123",
				InternalRoomID: "!a:alert-example.com",
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			outputConfig, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if outputConfig.ServerURL != scenario.ExpectedOutput.ServerURL {
				t.Errorf("expected ServerURL to be %s, got %s", scenario.ExpectedOutput.ServerURL, outputConfig.ServerURL)
			}
			if outputConfig.AccessToken != scenario.ExpectedOutput.AccessToken {
				t.Errorf("expected AccessToken to be %s, got %s", scenario.ExpectedOutput.AccessToken, outputConfig.AccessToken)
			}
			if outputConfig.InternalRoomID != scenario.ExpectedOutput.InternalRoomID {
				t.Errorf("expected InternalRoomID to be %s, got %s", scenario.ExpectedOutput.InternalRoomID, outputConfig.InternalRoomID)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}
