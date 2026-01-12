package endpoint

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint/dns"
	"github.com/TwiN/gatus/v5/config/endpoint/ssh"
	"github.com/TwiN/gatus/v5/config/endpoint/ui"
	"github.com/TwiN/gatus/v5/config/gontext"
	"github.com/TwiN/gatus/v5/config/maintenance"
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
			Name: "failed-status-condition-with-hidden-conditions",
			Endpoint: Endpoint{
				Name:       "website-health",
				URL:        "https://twin.sh/health",
				Conditions: []Condition{"[STATUS] == 200"},
				UIConfig:   &ui.Config{HideConditions: true},
			},
			ExpectedResult: &Result{
				Success:          false,
				Connected:        true,
				Hostname:         "twin.sh",
				ConditionResults: []*ConditionResult{}, // Because UIConfig.HideConditions is true, the condition results should not be shown.
				DomainExpiration: 0,                    // Because there's no [DOMAIN_EXPIRATION] condition, this is not resolved, so it should be 0.
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
				Interval:   5 * time.Minute,
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
				URL:          "https://twin.sh:9999/health",
				Conditions:   []Condition{"[CONNECTED] == true"},
				UIConfig:     &ui.Config{HideHostname: true, HidePort: true},
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
				Errors: []string{`Get "https://<redacted>:<redacted>/health": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`},
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
			err := scenario.Endpoint.ValidateAndSetDefaults()
			if err != nil {
				t.Error("did not expect an error, got", err)
			}
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

func TestEndpoint_ResolveSuccessfulConditions(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	endpoint := Endpoint{
		Name:       "test-endpoint",
		URL:        "https://example.com/health",
		Conditions: []Condition{"[BODY].status == UP"},
		UIConfig:   &ui.Config{ResolveSuccessfulConditions: true},
	}
	mockResponse := test.MockRoundTripper(func(r *http.Request) *http.Response {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"status":"UP"}`))}
	})
	client.InjectHTTPClient(&http.Client{Transport: mockResponse})
	if err := endpoint.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("ValidateAndSetDefaults failed: %v", err)
	}
	result := endpoint.EvaluateHealth()
	if len(result.ConditionResults) != 1 {
		t.Fatalf("expected 1 condition result, got %d", len(result.ConditionResults))
	}
	expectedCondition := "[BODY].status (UP) == UP"
	if result.ConditionResults[0].Condition != expectedCondition {
		t.Errorf("expected condition to be '%s', got '%s'", expectedCondition, result.ConditionResults[0].Condition)
	}
}

func TestEndpoint_IsEnabled(t *testing.T) {
	if !(&Endpoint{Enabled: nil}).IsEnabled() {
		t.Error("endpoint.IsEnabled() should've returned true, because Enabled was set to nil")
	}
	if value := false; (&Endpoint{Enabled: &value}).IsEnabled() {
		t.Error("endpoint.IsEnabled() should've returned false, because Enabled was set to false")
	}
	if value := true; !(&Endpoint{Enabled: &value}).IsEnabled() {
		t.Error("Endpoint.IsEnabled() should've returned true, because Enabled was set to true")
	}
}

func TestEndpoint_Type(t *testing.T) {
	type args struct {
		URL string
		DNS *dns.Config
		SSH *ssh.Config
	}
	tests := []struct {
		args args
		want Type
	}{
		{
			args: args{
				URL: "8.8.8.8",
				DNS: &dns.Config{
					QueryType: "A",
					QueryName: "example.com",
				},
			},
			want: TypeDNS,
		},
		{
			args: args{
				URL: "tcp://127.0.0.1:6379",
			},
			want: TypeTCP,
		},
		{
			args: args{
				URL: "icmp://example.com",
			},
			want: TypeICMP,
		},
		{
			args: args{
				URL: "sctp://example.com",
			},
			want: TypeSCTP,
		},
		{
			args: args{
				URL: "udp://example.com",
			},
			want: TypeUDP,
		},
		{
			args: args{
				URL: "starttls://smtp.gmail.com:587",
			},
			want: TypeSTARTTLS,
		},
		{
			args: args{
				URL: "tls://example.com:443",
			},
			want: TypeTLS,
		},
		{
			args: args{
				URL: "https://twin.sh/health",
			},
			want: TypeHTTP,
		},
		{
			args: args{
				URL: "wss://example.com/",
			},
			want: TypeWS,
		},
		{
			args: args{
				URL: "ws://example.com/",
			},
			want: TypeWS,
		},
		{
			args: args{
				URL: "ssh://example.com:22",
				SSH: &ssh.Config{
					Username: "root",
					Password: "password",
				},
			},
			want: TypeSSH,
		},
		{
			args: args{
				URL: "invalid://example.org",
			},
			want: TypeUNKNOWN,
		},
		{
			args: args{
				URL: "no-scheme",
			},
			want: TypeUNKNOWN,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			endpoint := Endpoint{
				URL:       tt.args.URL,
				DNSConfig: tt.args.DNS,
			}
			if got := endpoint.Type(); got != tt.want {
				t.Errorf("Endpoint.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndpoint_ValidateAndSetDefaults(t *testing.T) {
	endpoint := Endpoint{
		Name:               "website-health",
		URL:                "https://twin.sh/health",
		Conditions:         []Condition{Condition("[STATUS] == 200")},
		Alerts:             []*alert.Alert{{Type: alert.TypePagerDuty}},
		MaintenanceWindows: []*maintenance.Config{{Start: "03:50", Duration: 4 * time.Hour}},
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
	if len(endpoint.MaintenanceWindows) != 1 {
		t.Error("Endpoint should've had 1 maintenance window")
	}
	if !endpoint.MaintenanceWindows[0].IsEnabled() {
		t.Error("Endpoint maintenance should've defaulted to true")
	}
	if endpoint.MaintenanceWindows[0].Timezone != "UTC" {
		t.Error("Endpoint maintenance should've defaulted to UTC")
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
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
		DNSConfig: &dns.Config{
			QueryType: "A",
			QueryName: "example.com",
		},
		Conditions: []Condition{Condition("[DNS_RCODE] == NOERROR")},
	}
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Error("did not expect an error, got", err)
	}
	if endpoint.DNSConfig.QueryName != "example.com." {
		t.Error("Endpoint.dns.query-name should be formatted with . suffix")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithSSH(t *testing.T) {
	scenarios := []struct {
		name        string
		username    string
		password    string
		privateKey  string
		expectedErr error
	}{
		{
			name:        "fail when has no user but has password",
			username:    "",
			password:    "password",
			expectedErr: ssh.ErrEndpointWithoutSSHUsername,
		},
		{
			name:        "fail when has no user but has private key",
			username:    "",
			privateKey:  "-----BEGIN",
			expectedErr: ssh.ErrEndpointWithoutSSHUsername,
		},
		{
			name:        "fail when has no password or private key",
			username:    "username",
			password:    "",
			privateKey:  "",
			expectedErr: ssh.ErrEndpointWithoutSSHAuth,
		},
		{
			name:        "success when username and password are set",
			username:    "username",
			password:    "password",
			expectedErr: nil,
		},
		{
			name:        "success when username and private key are set",
			username:    "username",
			privateKey:  "-----BEGIN",
			expectedErr: nil,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			endpoint := &Endpoint{
				Name: "ssh-test",
				URL:  "https://example.com",
				SSHConfig: &ssh.Config{
					Username:   scenario.username,
					Password:   scenario.password,
					PrivateKey: scenario.privateKey,
				},
				Conditions: []Condition{Condition("[STATUS] == 0")},
			}
			err := endpoint.ValidateAndSetDefaults()
			if !errors.Is(err, scenario.expectedErr) {
				t.Errorf("expected error %v, got %v", scenario.expectedErr, err)
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
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
	conditionBody := Condition("[BODY] == pat(*.*.*.*)")
	endpoint := Endpoint{
		Name: "example",
		URL:  "8.8.8.8",
		DNSConfig: &dns.Config{
			QueryType: "A",
			QueryName: "example.com.",
		},
		Conditions: []Condition{conditionSuccess, conditionBody},
	}
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
	scenarios := []struct {
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
				SSHConfig: &ssh.Config{
					Username: "scenario",
					Password: "scenario",
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
				SSHConfig: &ssh.Config{
					Username: "scenario",
					Password: "scenario",
				},
				Body: "{ \"command\": \"uptime\" }",
			},
			conditions: []Condition{Condition("[STATUS] == 1")},
			success:    false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			scenario.endpoint.ValidateAndSetDefaults()
			scenario.endpoint.Conditions = scenario.conditions
			result := scenario.endpoint.EvaluateHealth()
			if result.Success != scenario.success {
				t.Errorf("Expected success to be %v, but was %v", scenario.success, result.Success)
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
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {
		t.Fatal("did not expect an error, got", err)
	}
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
	// Test store configuration with body placeholder
	storeWithBodyPlaceholder := map[string]string{
		"token": "[BODY].accessToken",
	}
	if !(&Endpoint{
		Conditions: []Condition{statusCondition},
		Store:      storeWithBodyPlaceholder,
	}).needsToReadBody() {
		t.Error("expected true when store has body placeholder, got false")
	}
	// Test store configuration without body placeholder
	storeWithoutBodyPlaceholder := map[string]string{
		"status": "[STATUS]",
	}
	if (&Endpoint{
		Conditions: []Condition{statusCondition},
		Store:      storeWithoutBodyPlaceholder,
	}).needsToReadBody() {
		t.Error("expected false when store has no body placeholder, got true")
	}
	// Test empty store
	if (&Endpoint{
		Conditions: []Condition{statusCondition},
		Store:      map[string]string{},
	}).needsToReadBody() {
		t.Error("expected false when store is empty, got true")
	}
	// Test nil store
	if (&Endpoint{
		Conditions: []Condition{statusCondition},
		Store:      nil,
	}).needsToReadBody() {
		t.Error("expected false when store is nil, got true")
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

func TestEndpoint_preprocessWithContext(t *testing.T) {
	// Import the gontext package for creating test contexts
	// This test thoroughly exercises the replaceContextPlaceholders function
	tests := []struct {
		name                  string
		endpoint              *Endpoint
		context               map[string]interface{}
		expectedURL           string
		expectedBody          string
		expectedHeaders       map[string]string
		expectedErrorCount    int
		expectedErrorContains []string
	}{
		{
			name: "successful_url_replacement",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].userId",
				Body: "",
			},
			context: map[string]interface{}{
				"userId": "12345",
			},
			expectedURL:        "https://api.example.com/users/12345",
			expectedBody:       "",
			expectedErrorCount: 0,
		},
		{
			name: "successful_body_replacement",
			endpoint: &Endpoint{
				URL:  "https://api.example.com",
				Body: `{"userId": "[CONTEXT].userId", "action": "update"}`,
			},
			context: map[string]interface{}{
				"userId": "67890",
			},
			expectedURL:        "https://api.example.com",
			expectedBody:       `{"userId": "67890", "action": "update"}`,
			expectedErrorCount: 0,
		},
		{
			name: "successful_header_replacement",
			endpoint: &Endpoint{
				URL:  "https://api.example.com",
				Body: "",
				Headers: map[string]string{
					"Authorization": "Bearer [CONTEXT].token",
					"X-User-ID":     "[CONTEXT].userId",
				},
			},
			context: map[string]interface{}{
				"token":  "abc123token",
				"userId": "user123",
			},
			expectedURL:  "https://api.example.com",
			expectedBody: "",
			expectedHeaders: map[string]string{
				"Authorization": "Bearer abc123token",
				"X-User-ID":     "user123",
			},
			expectedErrorCount: 0,
		},
		{
			name: "multiple_placeholders_in_url",
			endpoint: &Endpoint{
				URL:  "https://[CONTEXT].host/api/v[CONTEXT].version/users/[CONTEXT].userId",
				Body: "",
			},
			context: map[string]interface{}{
				"host":    "api.example.com",
				"version": "2",
				"userId":  "12345",
			},
			expectedURL:        "https://api.example.com/api/v2/users/12345",
			expectedBody:       "",
			expectedErrorCount: 0,
		},
		{
			name: "nested_context_path",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].user.id",
				Body: `{"name": "[CONTEXT].user.name"}`,
			},
			context: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   "nested123",
					"name": "John Doe",
				},
			},
			expectedURL:        "https://api.example.com/users/nested123",
			expectedBody:       `{"name": "John Doe"}`,
			expectedErrorCount: 0,
		},
		{
			name: "url_context_not_found",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].missingUserId",
				Body: "",
			},
			context: map[string]interface{}{
				"userId": "12345", // different key
			},
			expectedURL:           "https://api.example.com/users/[CONTEXT].missingUserId",
			expectedBody:          "",
			expectedErrorCount:    1,
			expectedErrorContains: []string{"path 'missingUserId' not found"},
		},
		{
			name: "body_context_not_found",
			endpoint: &Endpoint{
				URL:  "https://api.example.com",
				Body: `{"userId": "[CONTEXT].missingUserId"}`,
			},
			context: map[string]interface{}{
				"userId": "12345", // different key
			},
			expectedURL:           "https://api.example.com",
			expectedBody:          `{"userId": "[CONTEXT].missingUserId"}`,
			expectedErrorCount:    1,
			expectedErrorContains: []string{"path 'missingUserId' not found"},
		},
		{
			name: "header_context_not_found",
			endpoint: &Endpoint{
				URL:  "https://api.example.com",
				Body: "",
				Headers: map[string]string{
					"Authorization": "Bearer [CONTEXT].missingToken",
				},
			},
			context: map[string]interface{}{
				"token": "validtoken", // different key
			},
			expectedURL:  "https://api.example.com",
			expectedBody: "",
			expectedHeaders: map[string]string{
				"Authorization": "Bearer [CONTEXT].missingToken",
			},
			expectedErrorCount:    1,
			expectedErrorContains: []string{"path 'missingToken' not found"},
		},
		{
			name: "multiple_missing_context_paths",
			endpoint: &Endpoint{
				URL:  "https://[CONTEXT].missingHost/users/[CONTEXT].missingUserId",
				Body: `{"token": "[CONTEXT].missingToken"}`,
			},
			context: map[string]interface{}{
				"validKey": "validValue",
			},
			expectedURL:        "https://[CONTEXT].missingHost/users/[CONTEXT].missingUserId",
			expectedBody:       `{"token": "[CONTEXT].missingToken"}`,
			expectedErrorCount: 2, // 1 for URL (both placeholders), 1 for Body
			expectedErrorContains: []string{
				"path 'missingHost' not found",
				"path 'missingUserId' not found",
				"path 'missingToken' not found",
			},
		},
		{
			name: "mixed_valid_and_invalid_placeholders",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].userId/posts/[CONTEXT].missingPostId",
				Body: `{"userId": "[CONTEXT].userId", "action": "[CONTEXT].missingAction"}`,
			},
			context: map[string]interface{}{
				"userId": "12345",
			},
			expectedURL:        "https://api.example.com/users/12345/posts/[CONTEXT].missingPostId",
			expectedBody:       `{"userId": "12345", "action": "[CONTEXT].missingAction"}`,
			expectedErrorCount: 2,
			expectedErrorContains: []string{
				"path 'missingPostId' not found",
				"path 'missingAction' not found",
			},
		},
		{
			name: "nil_context",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].userId",
				Body: "",
			},
			context:            nil,
			expectedURL:        "https://api.example.com/users/[CONTEXT].userId",
			expectedBody:       "",
			expectedErrorCount: 0,
		},
		{
			name: "empty_context",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].userId",
				Body: "",
			},
			context:               map[string]interface{}{},
			expectedURL:           "https://api.example.com/users/[CONTEXT].userId",
			expectedBody:          "",
			expectedErrorCount:    1,
			expectedErrorContains: []string{"path 'userId' not found"},
		},
		{
			name: "special_characters_in_context_values",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/search?q=[CONTEXT].query",
				Body: "",
			},
			context: map[string]interface{}{
				"query": "hello world & special chars!",
			},
			expectedURL:        "https://api.example.com/search?q=hello world & special chars!",
			expectedBody:       "",
			expectedErrorCount: 0,
		},
		{
			name: "numeric_context_values",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].userId/limit/[CONTEXT].limit",
				Body: "",
			},
			context: map[string]interface{}{
				"userId": 12345,
				"limit":  100,
			},
			expectedURL:        "https://api.example.com/users/12345/limit/100",
			expectedBody:       "",
			expectedErrorCount: 0,
		},
		{
			name: "boolean_context_values",
			endpoint: &Endpoint{
				URL:  "https://api.example.com",
				Body: `{"enabled": [CONTEXT].enabled, "active": [CONTEXT].active}`,
			},
			context: map[string]interface{}{
				"enabled": true,
				"active":  false,
			},
			expectedURL:        "https://api.example.com",
			expectedBody:       `{"enabled": true, "active": false}`,
			expectedErrorCount: 0,
		},
		{
			name: "no_context_placeholders",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/health",
				Body: `{"status": "check"}`,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
			context: map[string]interface{}{
				"userId": "12345",
			},
			expectedURL:  "https://api.example.com/health",
			expectedBody: `{"status": "check"}`,
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectedErrorCount: 0,
		},
		{
			name: "deeply_nested_context_path",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].response.data.user.id",
				Body: "",
			},
			context: map[string]interface{}{
				"response": map[string]interface{}{
					"data": map[string]interface{}{
						"user": map[string]interface{}{
							"id": "deep123",
						},
					},
				},
			},
			expectedURL:        "https://api.example.com/users/deep123",
			expectedBody:       "",
			expectedErrorCount: 0,
		},
		{
			name: "invalid_nested_context_path",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].response.missing.path",
				Body: "",
			},
			context: map[string]interface{}{
				"response": map[string]interface{}{
					"data": "value",
				},
			},
			expectedURL:           "https://api.example.com/users/[CONTEXT].response.missing.path",
			expectedBody:          "",
			expectedErrorCount:    1,
			expectedErrorContains: []string{"path 'response.missing.path' not found"},
		},
		{
			name: "hyphen_support_in_simple_keys",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].user-id",
				Body: `{"api-key": "[CONTEXT].api-key", "user-name": "[CONTEXT].user-name"}`,
			},
			context: map[string]interface{}{
				"user-id":   "user-12345",
				"api-key":   "key-abcdef",
				"user-name": "john-doe",
			},
			expectedURL:        "https://api.example.com/users/user-12345",
			expectedBody:       `{"api-key": "key-abcdef", "user-name": "john-doe"}`,
			expectedErrorCount: 0,
		},
		{
			name: "hyphen_support_in_headers",
			endpoint: &Endpoint{
				URL:  "https://api.example.com",
				Body: "",
				Headers: map[string]string{
					"X-API-Key":    "[CONTEXT].api-key",
					"X-User-ID":    "[CONTEXT].user-id",
					"Content-Type": "[CONTEXT].content-type",
				},
			},
			context: map[string]interface{}{
				"api-key":      "secret-key-123",
				"user-id":      "user-456",
				"content-type": "application-json",
			},
			expectedURL:  "https://api.example.com",
			expectedBody: "",
			expectedHeaders: map[string]string{
				"X-API-Key":    "secret-key-123",
				"X-User-ID":    "user-456",
				"Content-Type": "application-json",
			},
			expectedErrorCount: 0,
		},
		{
			name: "mixed_hyphens_underscores_and_dots",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/[CONTEXT].service-name/[CONTEXT].user_data.user-id",
				Body: `{"tenant-id": "[CONTEXT].tenant_config.tenant-id"}`,
			},
			context: map[string]interface{}{
				"service-name": "auth-service",
				"user_data": map[string]interface{}{
					"user-id": "user-789",
				},
				"tenant_config": map[string]interface{}{
					"tenant-id": "tenant-abc-123",
				},
			},
			expectedURL:        "https://api.example.com/auth-service/user-789",
			expectedBody:       `{"tenant-id": "tenant-abc-123"}`,
			expectedErrorCount: 0,
		},
		{
			name: "hyphen_in_nested_paths",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].auth-response.user-data.profile-id",
				Body: "",
			},
			context: map[string]interface{}{
				"auth-response": map[string]interface{}{
					"user-data": map[string]interface{}{
						"profile-id": "profile-xyz-789",
					},
				},
			},
			expectedURL:        "https://api.example.com/users/profile-xyz-789",
			expectedBody:       "",
			expectedErrorCount: 0,
		},
		{
			name: "missing_hyphenated_context_key",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/users/[CONTEXT].missing-user-id",
				Body: `{"api-key": "[CONTEXT].missing-api-key"}`,
			},
			context: map[string]interface{}{
				"user-id": "valid-user", // different key
			},
			expectedURL:           "https://api.example.com/users/[CONTEXT].missing-user-id",
			expectedBody:          `{"api-key": "[CONTEXT].missing-api-key"}`,
			expectedErrorCount:    2,
			expectedErrorContains: []string{"path 'missing-user-id' not found", "path 'missing-api-key' not found"},
		},
		{
			name: "multiple_hyphens_in_single_key",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/[CONTEXT].multi-hyphen-key-name",
				Body: "",
			},
			context: map[string]interface{}{
				"multi-hyphen-key-name": "value-with-multiple-hyphens",
			},
			expectedURL:        "https://api.example.com/value-with-multiple-hyphens",
			expectedBody:       "",
			expectedErrorCount: 0,
		},
		{
			name: "hyphens_with_numeric_values",
			endpoint: &Endpoint{
				URL:  "https://api.example.com/limit/[CONTEXT].max-items",
				Body: `{"timeout-ms": [CONTEXT].timeout-ms, "retry-count": [CONTEXT].retry-count}`,
			},
			context: map[string]interface{}{
				"max-items":   100,
				"timeout-ms":  5000,
				"retry-count": 3,
			},
			expectedURL:        "https://api.example.com/limit/100",
			expectedBody:       `{"timeout-ms": 5000, "retry-count": 3}`,
			expectedErrorCount: 0,
		},
		{
			name: "hyphens_with_boolean_values",
			endpoint: &Endpoint{
				URL:  "https://api.example.com",
				Body: `{"enable-feature": [CONTEXT].enable-feature, "disable-cache": [CONTEXT].disable-cache}`,
			},
			context: map[string]interface{}{
				"enable-feature": true,
				"disable-cache":  false,
			},
			expectedURL:        "https://api.example.com",
			expectedBody:       `{"enable-feature": true, "disable-cache": false}`,
			expectedErrorCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Import gontext package for creating context
			var ctx *gontext.Gontext
			if tt.context != nil {
				ctx = gontext.New(tt.context)
			}
			// Create a new Result to capture errors
			result := &Result{}
			// Call preprocessWithContext
			processed := tt.endpoint.preprocessWithContext(result, ctx)
			// Verify URL
			if processed.URL != tt.expectedURL {
				t.Errorf("URL mismatch:\nexpected: %s\nactual:   %s", tt.expectedURL, processed.URL)
			}
			// Verify Body
			if processed.Body != tt.expectedBody {
				t.Errorf("Body mismatch:\nexpected: %s\nactual:   %s", tt.expectedBody, processed.Body)
			}
			// Verify Headers
			if tt.expectedHeaders != nil {
				if processed.Headers == nil {
					t.Error("Expected headers but got nil")
				} else {
					for key, expectedValue := range tt.expectedHeaders {
						if actualValue, exists := processed.Headers[key]; !exists {
							t.Errorf("Expected header %s not found", key)
						} else if actualValue != expectedValue {
							t.Errorf("Header %s mismatch:\nexpected: %s\nactual:   %s", key, expectedValue, actualValue)
						}
					}
				}
			}
			// Verify error count
			if len(result.Errors) != tt.expectedErrorCount {
				t.Errorf("Error count mismatch:\nexpected: %d\nactual:   %d\nerrors: %v", tt.expectedErrorCount, len(result.Errors), result.Errors)
			}
			// Verify error messages contain expected strings
			if tt.expectedErrorContains != nil {
				actualErrors := strings.Join(result.Errors, " ")
				for _, expectedError := range tt.expectedErrorContains {
					if !strings.Contains(actualErrors, expectedError) {
						t.Errorf("Expected error containing '%s' not found in: %v", expectedError, result.Errors)
					}
				}
			}
			// Verify original endpoint is not modified
			if tt.endpoint.URL != ((&Endpoint{URL: tt.endpoint.URL, Body: tt.endpoint.Body, Headers: tt.endpoint.Headers}).URL) {
				t.Error("Original endpoint was modified")
			}
		})
	}
}

func TestEndpoint_HideUIFeatures(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	tests := []struct {
		name              string
		endpoint          Endpoint
		mockResponse      test.MockRoundTripper
		checkHostname     bool
		expectHostname    string
		checkErrors       bool
		expectErrors      bool
		checkConditions   bool
		expectConditions  bool
		checkErrorContent string
	}{
		{
			name: "hide-conditions",
			endpoint: Endpoint{
				Name:       "test-endpoint",
				URL:        "https://example.com/health",
				Conditions: []Condition{"[STATUS] == 200", "[BODY].status == UP"},
				UIConfig:   &ui.Config{HideConditions: true},
			},
			mockResponse: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"status": "UP"}`))}
			}),
			checkConditions:  true,
			expectConditions: false,
		},
		{
			name: "hide-hostname",
			endpoint: Endpoint{
				Name:       "test-endpoint",
				URL:        "https://example.com/health",
				Conditions: []Condition{"[STATUS] == 200"},
				UIConfig:   &ui.Config{HideHostname: true},
			},
			mockResponse: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			checkHostname:  true,
			expectHostname: "",
		},
		{
			name: "hide-url-in-errors",
			endpoint: Endpoint{
				Name:         "test-endpoint",
				URL:          "https://example.com/health",
				Conditions:   []Condition{"[CONNECTED] == true"},
				UIConfig:     &ui.Config{HideURL: true},
				ClientConfig: &client.Config{Timeout: time.Millisecond},
			},
			mockResponse:      nil,
			checkErrors:       true,
			expectErrors:      true,
			checkErrorContent: "<redacted>",
		},
		{
			name: "hide-port-in-errors",
			endpoint: Endpoint{
				Name:         "test-endpoint",
				URL:          "https://example.com:9999/health",
				Conditions:   []Condition{"[CONNECTED] == true"},
				UIConfig:     &ui.Config{HidePort: true},
				ClientConfig: &client.Config{Timeout: time.Millisecond},
			},
			mockResponse:      nil,
			checkErrors:       true,
			expectErrors:      true,
			checkErrorContent: "<redacted>",
		},
		{
			name: "hide-errors",
			endpoint: Endpoint{
				Name:         "test-endpoint",
				URL:          "https://example.com/health",
				Conditions:   []Condition{"[CONNECTED] == true"},
				UIConfig:     &ui.Config{HideErrors: true},
				ClientConfig: &client.Config{Timeout: time.Millisecond},
			},
			mockResponse: nil,
			checkErrors:  true,
			expectErrors: false,
		},
		{
			name: "dont-resolve-failed-conditions",
			endpoint: Endpoint{
				Name:       "test-endpoint",
				URL:        "https://example.com/health",
				Conditions: []Condition{"[STATUS] == 200"},
				UIConfig:   &ui.Config{DontResolveFailedConditions: true},
			},
			mockResponse: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody}
			}),
			checkConditions:  true,
			expectConditions: true,
		},
		{
			name: "multiple-hide-features",
			endpoint: Endpoint{
				Name:       "test-endpoint",
				URL:        "https://example.com/health",
				Conditions: []Condition{"[STATUS] == 200"},
				UIConfig:   &ui.Config{HideConditions: true, HideHostname: true, HideErrors: true},
			},
			mockResponse: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			checkConditions:  true,
			expectConditions: false,
			checkHostname:    true,
			expectHostname:   "",
			checkErrors:      true,
			expectErrors:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockResponse != nil {
				mockClient := &http.Client{Transport: tt.mockResponse}
				if tt.endpoint.ClientConfig != nil && tt.endpoint.ClientConfig.Timeout > 0 {
					mockClient.Timeout = tt.endpoint.ClientConfig.Timeout
				}
				client.InjectHTTPClient(mockClient)
			} else {
				client.InjectHTTPClient(nil)
			}
			err := tt.endpoint.ValidateAndSetDefaults()
			if err != nil {
				t.Fatalf("ValidateAndSetDefaults failed: %v", err)
			}
			result := tt.endpoint.EvaluateHealth()
			if tt.checkHostname {
				if result.Hostname != tt.expectHostname {
					t.Errorf("Expected hostname '%s', got '%s'", tt.expectHostname, result.Hostname)
				}
			}
			if tt.checkErrors {
				hasErrors := len(result.Errors) > 0
				if hasErrors != tt.expectErrors {
					t.Errorf("Expected errors=%v, got errors=%v (actual errors: %v)", tt.expectErrors, hasErrors, result.Errors)
				}
				if tt.checkErrorContent != "" && len(result.Errors) > 0 {
					found := false
					for _, err := range result.Errors {
						if strings.Contains(err, tt.checkErrorContent) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error to contain '%s', but got: %v", tt.checkErrorContent, result.Errors)
					}
				}
			}
			if tt.checkConditions {
				hasConditions := len(result.ConditionResults) > 0
				if hasConditions != tt.expectConditions {
					t.Errorf("Expected conditions=%v, got conditions=%v (actual: %v)", tt.expectConditions, hasConditions, result.ConditionResults)
				}
			}
		})
	}
}
