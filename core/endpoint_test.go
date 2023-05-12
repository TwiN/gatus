package core

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core/ui"
	"github.com/TwiN/gatus/v5/test"
)

func TestEndpoint(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	scenarios := []struct {
		Name             string
		Endpoint         Endpoint
		ExpectedResult   *Result
		MockRoundTripper test.MockRoundTripper
	}{
		{
			Name: "success",
			Endpoint: Endpoint{
				Name:       "website-health",
				URL:        "https://twin.sh/health",
				Conditions: []Condition{"[STATUS] == 200", "[BODY].status == UP", "[CERTIFICATE_EXPIRATION] > 24h"},
			},
			ExpectedResult: &Result{
				Success:   true,
				Connected: true,
				Hostname:  "twin.sh",
				ConditionResults: []*ConditionResult{
					{Condition: "[STATUS] == 200", Success: true},
					{Condition: "[BODY].status == UP", Success: true},
					{Condition: "[CERTIFICATE_EXPIRATION] > 24h", Success: true},
				},
				DomainExpiration: 0, // Because there's no [DOMAIN_EXPIRATION] condition, this is not resolved, so it should be 0.
			},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"status": "UP"}`)),
					TLS:        &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{NotAfter: time.Now().Add(9999 * time.Hour)}}},
				}
			}),
		},
		{
			Name: "failed-body-condition",
			Endpoint: Endpoint{
				Name:       "website-health",
				URL:        "https://twin.sh/health",
				Conditions: []Condition{"[STATUS] == 200", "[BODY].status == UP"},
			},
			ExpectedResult: &Result{
				Success:   false,
				Connected: true,
				Hostname:  "twin.sh",
				ConditionResults: []*ConditionResult{
					{Condition: "[STATUS] == 200", Success: true},
					{Condition: "[BODY].status (DOWN) == UP", Success: false},
				},
				DomainExpiration: 0, // Because there's no [DOMAIN_EXPIRATION] condition, this is not resolved, so it should be 0.
			},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"status": "DOWN"}`))}
			}),
		},
		{
			Name: "failed-status-condition",
			Endpoint: Endpoint{
				Name:       "website-health",
				URL:        "https://twin.sh/health",
				Conditions: []Condition{"[STATUS] == 200"},
			},
			ExpectedResult: &Result{
				Success:   false,
				Connected: true,
				Hostname:  "twin.sh",
				ConditionResults: []*ConditionResult{
					{Condition: "[STATUS] (502) == 200", Success: false},
				},
				DomainExpiration: 0, // Because there's no [DOMAIN_EXPIRATION] condition, this is not resolved, so it should be 0.
			},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody}
			}),
		},
		{
			Name: "condition-with-failed-certificate-expiration",
			Endpoint: Endpoint{
				Name:       "website-health",
				URL:        "https://twin.sh/health",
				Conditions: []Condition{"[CERTIFICATE_EXPIRATION] > 100h"},
				UIConfig:   &ui.Config{DontResolveFailedConditions: true},
			},
			ExpectedResult: &Result{
				Success:   false,
				Connected: true,
				Hostname:  "twin.sh",
				ConditionResults: []*ConditionResult{
					// Because UIConfig.DontResolveFailedConditions is true, the values in the condition should not be resolved
					{Condition: "[CERTIFICATE_EXPIRATION] > 100h", Success: false},
				},
				DomainExpiration: 0, // Because there's no [DOMAIN_EXPIRATION] condition, this is not resolved, so it should be 0.
			},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       http.NoBody,
					TLS:        &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{NotAfter: time.Now().Add(5 * time.Hour)}}},
				}
			}),
		},
		{
			Name: "domain-expiration",
			Endpoint: Endpoint{
				Name:       "website-health",
				URL:        "https://twin.sh/health",
				Conditions: []Condition{"[DOMAIN_EXPIRATION] > 100h"},
			},
			ExpectedResult: &Result{
				Success:   true,
				Connected: true,
				Hostname:  "twin.sh",
				ConditionResults: []*ConditionResult{
					{Condition: "[DOMAIN_EXPIRATION] > 100h", Success: true},
				},
				DomainExpiration: 999999 * time.Hour, // Note that this test only checks if it's non-zero.
			},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
		},
		{
			Name: "endpoint-that-will-time-out-and-hidden-hostname",
			Endpoint: Endpoint{
				Name:         "endpoint-that-will-time-out",
				URL:          "https://twin.sh/health",
				Conditions:   []Condition{"[CONNECTED] == true"},
				UIConfig:     &ui.Config{HideHostname: true},
				ClientConfig: &client.Config{Timeout: time.Millisecond},
			},
			ExpectedResult: &Result{
				Success:   false,
				Connected: false,
				Hostname:  "", // Because Endpoint.UIConfig.HideHostname is true, this should be empty.
				ConditionResults: []*ConditionResult{
					{Condition: "[CONNECTED] (false) == true", Success: false},
				},
				// Because there's no [DOMAIN_EXPIRATION] condition, this is not resolved, so it should be 0.
				DomainExpiration: 0,
				// Because Endpoint.UIConfig.HideHostname is true, the hostname should be replaced by <redacted>.
				Errors: []string{`Get "https://<redacted>/health": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`},
			},
			MockRoundTripper: nil,
		},
		{
			Name: "endpoint-that-will-time-out-and-hidden-url",
			Endpoint: Endpoint{
				Name:         "endpoint-that-will-time-out",
				URL:          "https://twin.sh/health",
				Conditions:   []Condition{"[CONNECTED] == true"},
				UIConfig:     &ui.Config{HideURL: true},
				ClientConfig: &client.Config{Timeout: time.Millisecond},
			},
			ExpectedResult: &Result{
				Success:   false,
				Connected: false,
				Hostname:  "twin.sh",
				ConditionResults: []*ConditionResult{
					{Condition: "[CONNECTED] (false) == true", Success: false},
				},
				// Because there's no [DOMAIN_EXPIRATION] condition, this is not resolved, so it should be 0.
				DomainExpiration: 0,
				// Because Endpoint.UIConfig.HideURL is true, the URL should be replaced by <redacted>.
				Errors: []string{`Get "<redacted>": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`},
			},
			MockRoundTripper: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if scenario.MockRoundTripper != nil {
				mockClient := &http.Client{Transport: scenario.MockRoundTripper}
				if scenario.Endpoint.ClientConfig != nil && scenario.Endpoint.ClientConfig.Timeout > 0 {
					mockClient.Timeout = scenario.Endpoint.ClientConfig.Timeout
				}
				client.InjectHTTPClient(mockClient)
			} else {
				client.InjectHTTPClient(nil)
			}
			scenario.Endpoint.ValidateAndSetDefaults()
			result := scenario.Endpoint.EvaluateHealth()
			if result.Success != scenario.ExpectedResult.Success {
				t.Errorf("Expected success to be %v, got %v", scenario.ExpectedResult.Success, result.Success)
			}
			if result.Connected != scenario.ExpectedResult.Connected {
				t.Errorf("Expected connected to be %v, got %v", scenario.ExpectedResult.Connected, result.Connected)
			}
			if result.Hostname != scenario.ExpectedResult.Hostname {
				t.Errorf("Expected hostname to be %v, got %v", scenario.ExpectedResult.Hostname, result.Hostname)
			}
			if len(result.ConditionResults) != len(scenario.ExpectedResult.ConditionResults) {
				t.Errorf("Expected %v condition results, got %v", len(scenario.ExpectedResult.ConditionResults), len(result.ConditionResults))
			} else {
				for i, conditionResult := range result.ConditionResults {
					if conditionResult.Condition != scenario.ExpectedResult.ConditionResults[i].Condition {
						t.Errorf("Expected condition to be %v, got %v", scenario.ExpectedResult.ConditionResults[i].Condition, conditionResult.Condition)
					}
					if conditionResult.Success != scenario.ExpectedResult.ConditionResults[i].Success {
						t.Errorf("Expected success of condition '%s' to be %v, got %v", conditionResult.Condition, scenario.ExpectedResult.ConditionResults[i].Success, conditionResult.Success)
					}
				}
			}
			if len(result.Errors) != len(scenario.ExpectedResult.Errors) {
				t.Errorf("Expected %v errors, got %v", len(scenario.ExpectedResult.Errors), len(result.Errors))
			} else {
				for i, err := range result.Errors {
					if err != scenario.ExpectedResult.Errors[i] {
						t.Errorf("Expected error to be %v, got %v", scenario.ExpectedResult.Errors[i], err)
					}
				}
			}
			if result.DomainExpiration != scenario.ExpectedResult.DomainExpiration {
				// Note that DomainExpiration is only resolved if there's a condition with the DomainExpirationPlaceholder in it.
				// In other words, if there's no condition with [DOMAIN_EXPIRATION] in it, the DomainExpiration field will be 0.
				// Because this is a live call, mocking it would be too much of a pain, so we're just going to check if
				// the actual value is non-zero when the expected result is non-zero.
				if scenario.ExpectedResult.DomainExpiration.Hours() > 0 && !(result.DomainExpiration.Hours() > 0) {
					t.Errorf("Expected domain expiration to be non-zero, got %v", result.DomainExpiration)
				}
			}
		})
	}
}

func TestEndpoint_IsEnabled(t *testing.T) {
	if !(Endpoint{Enabled: nil}).IsEnabled() {
		t.Error("endpoint.IsEnabled() should've returned true, because Enabled was set to nil")
	}
	if value := false; (Endpoint{Enabled: &value}).IsEnabled() {
		t.Error("endpoint.IsEnabled() should've returned false, because Enabled was set to false")
	}
	if value := true; !(Endpoint{Enabled: &value}).IsEnabled() {
		t.Error("Endpoint.IsEnabled() should've returned true, because Enabled was set to true")
	}
}

func TestEndpoint_Type(t *testing.T) {
	type args struct {
		URL string
		DNS *DNS
		SSH *SSH
	}
	tests := []struct {
		args args
		want EndpointType
	}{
		{
			args: args{
				URL: "8.8.8.8",
				DNS: &DNS{
					QueryType: "A",
					QueryName: "example.com",
				},
			},
			want: EndpointTypeDNS,
		},
		{
			args: args{
				URL: "tcp://127.0.0.1:6379",
			},
			want: EndpointTypeTCP,
		},
		{
			args: args{
				URL: "icmp://example.com",
			},
			want: EndpointTypeICMP,
		},
		{
			args: args{
				URL: "sctp://example.com",
			},
			want: EndpointTypeSCTP,
		},
		{
			args: args{
				URL: "udp://example.com",
			},
			want: EndpointTypeUDP,
		},
		{
			args: args{
				URL: "starttls://smtp.gmail.com:587",
			},
			want: EndpointTypeSTARTTLS,
		},
		{
			args: args{
				URL: "tls://example.com:443",
			},
			want: EndpointTypeTLS,
		},
		{
			args: args{
				URL: "https://twin.sh/health",
			},
			want: EndpointTypeHTTP,
		},
		{
			args: args{
				URL: "wss://example.com/",
			},
			want: EndpointTypeWS,
		},
		{
			args: args{
				URL: "ws://example.com/",
			},
			want: EndpointTypeWS,
		},
		{
			args: args{
				URL: "ssh://example.com:22",
				SSH: &SSH{
					Username: "root",
					Password: "password",
				},
			},
			want: EndpointTypeSSH,
		},
		{
			args: args{
				URL: "invalid://example.org",
			},
			want: EndpointTypeUNKNOWN,
		},
		{
			args: args{
				URL: "no-scheme",
			},
			want: EndpointTypeUNKNOWN,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			endpoint := Endpoint{
				URL: tt.args.URL,
				DNS: tt.args.DNS,
			}
			if got := endpoint.Type(); got != tt.want {
				t.Errorf("Endpoint.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpoint_ValidateAndSetDefaults(t *testing.T) {
	endpoint := Endpoint{
		Name:       "website-health",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{Condition("[STATUS] == 200")},
		Alerts:     []*alert.Alert{{Type: alert.TypePagerDuty}},
	}
	if err := endpoint.ValidateAndSetDefaults(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if endpoint.ClientConfig == nil {
		t.Error("client configuration should've been set to the default configuration")
	} else {
		if endpoint.ClientConfig.Insecure != client.GetDefaultConfig().Insecure {
			t.Errorf("Default client configuration should've set Insecure to %v, got %v", client.GetDefaultConfig().Insecure, endpoint.ClientConfig.Insecure)
		}
		if endpoint.ClientConfig.IgnoreRedirect != client.GetDefaultConfig().IgnoreRedirect {
			t.Errorf("Default client configuration should've set IgnoreRedirect to %v, got %v", client.GetDefaultConfig().IgnoreRedirect, endpoint.ClientConfig.IgnoreRedirect)
		}
		if endpoint.ClientConfig.Timeout != client.GetDefaultConfig().Timeout {
			t.Errorf("Default client configuration should've set Timeout to %v, got %v", client.GetDefaultConfig().Timeout, endpoint.ClientConfig.Timeout)
		}
	}
	if endpoint.Method != "GET" {
		t.Error("Endpoint method should've defaulted to GET")
	}
	if endpoint.Interval != time.Minute {
		t.Error("Endpoint interval should've defaulted to 1 minute")
	}
	if endpoint.Headers == nil {
		t.Error("Endpoint headers should've defaulted to an empty map")
	}
	if len(endpoint.Alerts) != 1 {
		t.Error("Endpoint should've had 1 alert")
	}
	if !endpoint.Alerts[0].IsEnabled() {
		t.Error("Endpoint alert should've defaulted to true")
	}
	if endpoint.Alerts[0].SuccessThreshold != 2 {
		t.Error("Endpoint alert should've defaulted to a success threshold of 2")
	}
	if endpoint.Alerts[0].FailureThreshold != 3 {
		t.Error("Endpoint alert should've defaulted to a failure threshold of 3")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithInvalidCondition(t *testing.T) {
	endpoint := Endpoint{
		Name:       "invalid-condition",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{"[STATUS] invalid 200"},
	}
	if err := endpoint.ValidateAndSetDefaults(); err == nil {
		t.Error("endpoint validation should've returned an error, but didn't")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithClientConfig(t *testing.T) {
	endpoint := Endpoint{
		Name:       "website-health",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{Condition("[STATUS] == 200")},
		ClientConfig: &client.Config{
			Insecure:       true,
			IgnoreRedirect: true,
			Timeout:        0,
		},
	}
	endpoint.ValidateAndSetDefaults()
	if endpoint.ClientConfig == nil {
		t.Error("client configuration should've been set to the default configuration")
	} else {
		if !endpoint.ClientConfig.Insecure {
			t.Error("endpoint.ClientConfig.Insecure should've been set to true")
		}
		if !endpoint.ClientConfig.IgnoreRedirect {
			t.Error("endpoint.ClientConfig.IgnoreRedirect should've been set to true")
		}
		if endpoint.ClientConfig.Timeout != client.GetDefaultConfig().Timeout {
			t.Error("endpoint.ClientConfig.Timeout should've been set to 10s, because the timeout value entered is not set or invalid")
		}
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithDNS(t *testing.T) {
	endpoint := &Endpoint{
		Name: "dns-test",
		URL:  "https://example.com",
		DNS: &DNS{
			QueryType: "A",
			QueryName: "example.com",
		},
		Conditions: []Condition{Condition("[DNS_RCODE] == NOERROR")},
	}
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Error("did not expect an error, got", err)
	}
	if endpoint.DNS.QueryName != "example.com." {
		t.Error("Endpoint.dns.query-name should be formatted with . suffix")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithSSH(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		password    string
		expectedErr error
	}{
		{
			name:        "fail when has no user",
			username:    "",
			password:    "password",
			expectedErr: ErrEndpointWithoutSSHUsername,
		},
		{
			name:        "fail when has no password",
			username:    "username",
			password:    "",
			expectedErr: ErrEndpointWithoutSSHPassword,
		},
		{
			name:        "success when all fields are set",
			username:    "username",
			password:    "password",
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoint := &Endpoint{
				Name: "ssh-test",
				URL:  "https://example.com",
				SSH: &SSH{
					Username: test.username,
					Password: test.password,
				},
				Conditions: []Condition{Condition("[STATUS] == 0")},
			}
			err := endpoint.ValidateAndSetDefaults()
			if err != test.expectedErr {
				t.Errorf("expected error %v, got %v", test.expectedErr, err)
			}
		})
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithSimpleErrors(t *testing.T) {
	scenarios := []struct {
		endpoint    *Endpoint
		expectedErr error
	}{
		{
			endpoint: &Endpoint{
				Name:       "",
				URL:        "https://example.com",
				Conditions: []Condition{Condition("[STATUS] == 200")},
			},
			expectedErr: ErrEndpointWithNoName,
		},
		{
			endpoint: &Endpoint{
				Name:       "endpoint-with-no-url",
				URL:        "",
				Conditions: []Condition{Condition("[STATUS] == 200")},
			},
			expectedErr: ErrEndpointWithNoURL,
		},
		{
			endpoint: &Endpoint{
				Name:       "endpoint-with-no-conditions",
				URL:        "https://example.com",
				Conditions: nil,
			},
			expectedErr: ErrEndpointWithNoCondition,
		},
		{
			endpoint: &Endpoint{
				Name:       "domain-expiration-with-bad-interval",
				URL:        "https://example.com",
				Interval:   time.Minute,
				Conditions: []Condition{Condition("[DOMAIN_EXPIRATION] > 720h")},
			},
			expectedErr: ErrInvalidEndpointIntervalForDomainExpirationPlaceholder,
		},
		{
			endpoint: &Endpoint{
				Name:       "domain-expiration-with-good-interval",
				URL:        "https://example.com",
				Interval:   5 * time.Minute,
				Conditions: []Condition{Condition("[DOMAIN_EXPIRATION] > 720h")},
			},
			expectedErr: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.endpoint.Name, func(t *testing.T) {
			if err := scenario.endpoint.ValidateAndSetDefaults(); err != scenario.expectedErr {
				t.Errorf("Expected error %v, got %v", scenario.expectedErr, err)
			}
		})
	}
}

func TestEndpoint_buildHTTPRequest(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	endpoint := Endpoint{
		Name:       "website-health",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{condition},
	}
	endpoint.ValidateAndSetDefaults()
	request := endpoint.buildHTTPRequest()
	if request.Method != "GET" {
		t.Error("request.Method should've been GET, but was", request.Method)
	}
	if request.Host != "twin.sh" {
		t.Error("request.Host should've been twin.sh, but was", request.Host)
	}
	if userAgent := request.Header.Get("User-Agent"); userAgent != GatusUserAgent {
		t.Errorf("request.Header.Get(User-Agent) should've been %s, but was %s", GatusUserAgent, userAgent)
	}
}

func TestEndpoint_buildHTTPRequestWithCustomUserAgent(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	endpoint := Endpoint{
		Name:       "website-health",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{condition},
		Headers: map[string]string{
			"User-Agent": "Test/2.0",
		},
	}
	endpoint.ValidateAndSetDefaults()
	request := endpoint.buildHTTPRequest()
	if request.Method != "GET" {
		t.Error("request.Method should've been GET, but was", request.Method)
	}
	if request.Host != "twin.sh" {
		t.Error("request.Host should've been twin.sh, but was", request.Host)
	}
	if userAgent := request.Header.Get("User-Agent"); userAgent != "Test/2.0" {
		t.Errorf("request.Header.Get(User-Agent) should've been %s, but was %s", "Test/2.0", userAgent)
	}
}

func TestEndpoint_buildHTTPRequestWithHostHeader(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	endpoint := Endpoint{
		Name:       "website-health",
		URL:        "https://twin.sh/health",
		Method:     "POST",
		Conditions: []Condition{condition},
		Headers: map[string]string{
			"Host": "example.com",
		},
	}
	endpoint.ValidateAndSetDefaults()
	request := endpoint.buildHTTPRequest()
	if request.Method != "POST" {
		t.Error("request.Method should've been POST, but was", request.Method)
	}
	if request.Host != "example.com" {
		t.Error("request.Host should've been example.com, but was", request.Host)
	}
}

func TestEndpoint_buildHTTPRequestWithGraphQLEnabled(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	endpoint := Endpoint{
		Name:       "website-graphql",
		URL:        "https://twin.sh/graphql",
		Method:     "POST",
		Conditions: []Condition{condition},
		GraphQL:    true,
		Body: `{
  users(gender: "female") {
    id
    name
    gender
    avatar
  }
}`,
	}
	endpoint.ValidateAndSetDefaults()
	request := endpoint.buildHTTPRequest()
	if request.Method != "POST" {
		t.Error("request.Method should've been POST, but was", request.Method)
	}
	if contentType := request.Header.Get(ContentTypeHeader); contentType != "application/json" {
		t.Error("request.Header.Content-Type should've been application/json, but was", contentType)
	}
	body, _ := io.ReadAll(request.Body)
	if !strings.HasPrefix(string(body), "{\"query\":") {
		t.Error("request.body should've started with '{\"query\":', but it didn't:", string(body))
	}
}

func TestIntegrationEvaluateHealth(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	bodyCondition := Condition("[BODY].status == UP")
	endpoint := Endpoint{
		Name:       "website-health",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{condition, bodyCondition},
	}
	endpoint.ValidateAndSetDefaults()
	result := endpoint.EvaluateHealth()
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	if !result.Connected {
		t.Error("Because the connection has been established, result.Connected should've been true")
	}
	if !result.Success {
		t.Error("Because all conditions passed, this should have been a success")
	}
	if result.Hostname != "twin.sh" {
		t.Error("result.Hostname should've been twin.sh, but was", result.Hostname)
	}
}

func TestIntegrationEvaluateHealthWithErrorAndHideURL(t *testing.T) {
	endpoint := Endpoint{
		Name:       "invalid-url",
		URL:        "https://httpstat.us/200?sleep=100",
		Conditions: []Condition{Condition("[STATUS] == 200")},
		ClientConfig: &client.Config{
			Timeout: 1 * time.Millisecond,
		},
		UIConfig: &ui.Config{
			HideURL: true,
		},
	}
	endpoint.ValidateAndSetDefaults()
	result := endpoint.EvaluateHealth()
	if result.Success {
		t.Error("Because one of the conditions was invalid, result.Success should have been false")
	}
	if len(result.Errors) == 0 {
		t.Error("There should've been an error")
	}
	if !strings.Contains(result.Errors[0], "<redacted>") || strings.Contains(result.Errors[0], endpoint.URL) {
		t.Error("result.Errors[0] should've had the URL redacted because ui.hide-url is set to true")
	}
}

func TestIntegrationEvaluateHealthForDNS(t *testing.T) {
	conditionSuccess := Condition("[DNS_RCODE] == NOERROR")
	conditionBody := Condition("[BODY] == 93.184.216.34")
	endpoint := Endpoint{
		Name: "example",
		URL:  "8.8.8.8",
		DNS: &DNS{
			QueryType: "A",
			QueryName: "example.com.",
		},
		Conditions: []Condition{conditionSuccess, conditionBody},
	}
	endpoint.ValidateAndSetDefaults()
	result := endpoint.EvaluateHealth()
	if !result.ConditionResults[0].Success {
		t.Errorf("Conditions '%s' and '%s' should have been a success", conditionSuccess, conditionBody)
	}
	if !result.Connected {
		t.Error("Because the connection has been established, result.Connected should've been true")
	}
	if !result.Success {
		t.Error("Because all conditions passed, this should have been a success")
	}
}

func TestIntegrationEvaluateHealthForSSH(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   Endpoint
		conditions []Condition
		success    bool
	}{
		{
			name: "ssh-success",
			endpoint: Endpoint{
				Name: "ssh-success",
				URL:  "ssh://localhost",
				SSH: &SSH{
					Username: "test",
					Password: "test",
				},
				Body: "{ \"command\": \"uptime\" }",
			},
			conditions: []Condition{Condition("[STATUS] == 0")},
			success:    true,
		},
		{
			name: "ssh-failure",
			endpoint: Endpoint{
				Name: "ssh-failure",
				URL:  "ssh://localhost",
				SSH: &SSH{
					Username: "test",
					Password: "test",
				},
				Body: "{ \"command\": \"uptime\" }",
			},
			conditions: []Condition{Condition("[STATUS] == 1")},
			success:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.endpoint.ValidateAndSetDefaults()
			test.endpoint.Conditions = test.conditions
			result := test.endpoint.EvaluateHealth()
			if result.Success != test.success {
				t.Errorf("Expected success to be %v, but was %v", test.success, result.Success)
			}
		})
	}
}

func TestIntegrationEvaluateHealthForICMP(t *testing.T) {
	endpoint := Endpoint{
		Name:       "icmp-test",
		URL:        "icmp://127.0.0.1",
		Conditions: []Condition{"[CONNECTED] == true"},
	}
	endpoint.ValidateAndSetDefaults()
	result := endpoint.EvaluateHealth()
	if !result.ConditionResults[0].Success {
		t.Errorf("Conditions '%s' should have been a success", endpoint.Conditions[0])
	}
	if !result.Connected {
		t.Error("Because the connection has been established, result.Connected should've been true")
	}
	if !result.Success {
		t.Error("Because all conditions passed, this should have been a success")
	}
}

func TestEndpoint_DisplayName(t *testing.T) {
	if endpoint := (Endpoint{Name: "n"}); endpoint.DisplayName() != "n" {
		t.Error("endpoint.DisplayName() should've been 'n', but was", endpoint.DisplayName())
	}
	if endpoint := (Endpoint{Group: "g", Name: "n"}); endpoint.DisplayName() != "g/n" {
		t.Error("endpoint.DisplayName() should've been 'g/n', but was", endpoint.DisplayName())
	}
}

func TestEndpoint_getIP(t *testing.T) {
	endpoint := Endpoint{
		Name:       "invalid-url-test",
		URL:        "",
		Conditions: []Condition{"[CONNECTED] == true"},
	}
	result := &Result{}
	endpoint.getIP(result)
	if len(result.Errors) == 0 {
		t.Error("endpoint.getIP(result) should've thrown an error because the URL is invalid, thus cannot be parsed")
	}
}

func TestEndpoint_needsToReadBody(t *testing.T) {
	statusCondition := Condition("[STATUS] == 200")
	bodyCondition := Condition("[BODY].status == UP")
	bodyConditionWithLength := Condition("len([BODY].tags) > 0")
	if (&Endpoint{Conditions: []Condition{statusCondition}}).needsToReadBody() {
		t.Error("expected false, got true")
	}
	if !(&Endpoint{Conditions: []Condition{bodyCondition}}).needsToReadBody() {
		t.Error("expected true, got false")
	}
	if !(&Endpoint{Conditions: []Condition{bodyConditionWithLength}}).needsToReadBody() {
		t.Error("expected true, got false")
	}
	if !(&Endpoint{Conditions: []Condition{statusCondition, bodyCondition}}).needsToReadBody() {
		t.Error("expected true, got false")
	}
	if !(&Endpoint{Conditions: []Condition{bodyCondition, statusCondition}}).needsToReadBody() {
		t.Error("expected true, got false")
	}
	if !(&Endpoint{Conditions: []Condition{bodyConditionWithLength, statusCondition}}).needsToReadBody() {
		t.Error("expected true, got false")
	}
}

func TestEndpoint_needsToRetrieveDomainExpiration(t *testing.T) {
	if (&Endpoint{Conditions: []Condition{"[STATUS] == 200"}}).needsToRetrieveDomainExpiration() {
		t.Error("expected false, got true")
	}
	if !(&Endpoint{Conditions: []Condition{"[STATUS] == 200", "[DOMAIN_EXPIRATION] < 720h"}}).needsToRetrieveDomainExpiration() {
		t.Error("expected true, got false")
	}
}

func TestEndpoint_needsToRetrieveIP(t *testing.T) {
	if (&Endpoint{Conditions: []Condition{"[STATUS] == 200"}}).needsToRetrieveIP() {
		t.Error("expected false, got true")
	}
	if !(&Endpoint{Conditions: []Condition{"[STATUS] == 200", "[IP] == 127.0.0.1"}}).needsToRetrieveIP() {
		t.Error("expected true, got false")
	}
}
