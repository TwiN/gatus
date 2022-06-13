package core

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core/ui"
)

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
	type fields struct {
		URL string
		DNS *DNS
	}
	tests := []struct {
		fields fields
		want   EndpointType
	}{{
		fields: fields{
			URL: "8.8.8.8",
			DNS: &DNS{
				QueryType: "A",
				QueryName: "example.com",
			},
		},
		want: EndpointTypeDNS,
	}, {
		fields: fields{
			URL: "tcp://127.0.0.1:6379",
		},
		want: EndpointTypeTCP,
	}, {
		fields: fields{
			URL: "icmp://example.com",
		},
		want: EndpointTypeICMP,
	}, {
		fields: fields{
			URL: "starttls://smtp.gmail.com:587",
		},
		want: EndpointTypeSTARTTLS,
	}, {
		fields: fields{
			URL: "tls://example.com:443",
		},
		want: EndpointTypeTLS,
	}, {
		fields: fields{
			URL: "https://twin.sh/health",
		},
		want: EndpointTypeHTTP,
	}}
	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			endpoint := Endpoint{
				URL: tt.fields.URL,
				DNS: tt.fields.DNS,
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
	endpoint.ValidateAndSetDefaults()
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
	if endpoint.Alerts[0].IsEnabled() {
		t.Error("Endpoint alert should've defaulted to disabled")
	}
	if endpoint.Alerts[0].SuccessThreshold != 2 {
		t.Error("Endpoint alert should've defaulted to a success threshold of 2")
	}
	if endpoint.Alerts[0].FailureThreshold != 3 {
		t.Error("Endpoint alert should've defaulted to a failure threshold of 3")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithClientConfig(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	endpoint := Endpoint{
		Name:       "website-health",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{condition},
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

func TestEndpoint_ValidateAndSetDefaultsWithNoName(t *testing.T) {
	defer func() { recover() }()
	condition := Condition("[STATUS] == 200")
	endpoint := &Endpoint{
		Name:       "",
		URL:        "http://example.com",
		Conditions: []Condition{condition},
	}
	err := endpoint.ValidateAndSetDefaults()
	if err == nil {
		t.Fatal("Should've returned an error because endpoint didn't have a name, which is a mandatory field")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithNoUrl(t *testing.T) {
	defer func() { recover() }()
	condition := Condition("[STATUS] == 200")
	endpoint := &Endpoint{
		Name:       "example",
		URL:        "",
		Conditions: []Condition{condition},
	}
	err := endpoint.ValidateAndSetDefaults()
	if err == nil {
		t.Fatal("Should've returned an error because endpoint didn't have an url, which is a mandatory field")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithNoConditions(t *testing.T) {
	defer func() { recover() }()
	endpoint := &Endpoint{
		Name:       "example",
		URL:        "http://example.com",
		Conditions: nil,
	}
	err := endpoint.ValidateAndSetDefaults()
	if err == nil {
		t.Fatal("Should've returned an error because endpoint didn't have at least 1 condition")
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithDNS(t *testing.T) {
	endpoint := &Endpoint{
		Name: "dns-test",
		URL:  "http://example.com",
		DNS: &DNS{
			QueryType: "A",
			QueryName: "example.com",
		},
		Conditions: []Condition{Condition("[DNS_RCODE] == NOERROR")},
	}
	err := endpoint.ValidateAndSetDefaults()
	if err != nil {

	}
	if endpoint.DNS.QueryName != "example.com." {
		t.Error("Endpoint.dns.query-name should be formatted with . suffix")
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

func TestIntegrationEvaluateHealthWithFailure(t *testing.T) {
	condition := Condition("[STATUS] == 500")
	endpoint := Endpoint{
		Name:       "website-health",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{condition},
	}
	endpoint.ValidateAndSetDefaults()
	result := endpoint.EvaluateHealth()
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	if !result.Connected {
		t.Error("Because the connection has been established, result.Connected should've been true")
	}
	if result.Success {
		t.Error("Because one of the conditions failed, result.Success should have been false")
	}
}

func TestIntegrationEvaluateHealthWithInvalidCondition(t *testing.T) {
	condition := Condition("[STATUS] invalid 200")
	endpoint := Endpoint{
		Name:       "invalid-condition",
		URL:        "https://twin.sh/health",
		Conditions: []Condition{condition},
	}
	if err := endpoint.ValidateAndSetDefaults(); err != nil {
		// XXX: Should this really not return an error? After all, the condition is not valid and conditions are part of the endpoint...
		t.Error("endpoint validation should've been successful, but wasn't")
	}
	result := endpoint.EvaluateHealth()
	if result.Success {
		t.Error("Because one of the conditions was invalid, result.Success should have been false")
	}
	if len(result.Errors) == 0 {
		t.Error("There should've been an error")
	}
}

func TestIntegrationEvaluateHealthWithError(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	endpoint := Endpoint{
		Name:       "invalid-host",
		URL:        "http://invalid/health",
		Conditions: []Condition{condition},
		UIConfig: &ui.Config{
			HideHostname: true,
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
	if !strings.Contains(result.Errors[0], "<redacted>") {
		t.Error("result.Errors[0] should've had the hostname redacted because ui.hide-hostname is set to true")
	}
	if result.Hostname != "" {
		t.Error("result.Hostname should've been empty because ui.hide-hostname is set to true")
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
		t.Errorf("Conditions '%s' and %s should have been a success", conditionSuccess, conditionBody)
	}
	if !result.Connected {
		t.Error("Because the connection has been established, result.Connected should've been true")
	}
	if !result.Success {
		t.Error("Because all conditions passed, this should have been a success")
	}
}

func TestIntegrationEvaluateHealthForICMP(t *testing.T) {
	conditionSuccess := Condition("[CONNECTED] == true")
	endpoint := Endpoint{
		Name:       "icmp-test",
		URL:        "icmp://127.0.0.1",
		Conditions: []Condition{conditionSuccess},
	}
	endpoint.ValidateAndSetDefaults()
	result := endpoint.EvaluateHealth()
	if !result.ConditionResults[0].Success {
		t.Errorf("Conditions '%s' should have been a success", conditionSuccess)
	}
	if !result.Connected {
		t.Error("Because the connection has been established, result.Connected should've been true")
	}
	if !result.Success {
		t.Error("Because all conditions passed, this should have been a success")
	}
}

func TestEndpoint_getIP(t *testing.T) {
	conditionSuccess := Condition("[CONNECTED] == true")
	endpoint := Endpoint{
		Name:       "invalid-url-test",
		URL:        "",
		Conditions: []Condition{conditionSuccess},
	}
	result := &Result{}
	endpoint.getIP(result)
	if len(result.Errors) == 0 {
		t.Error("endpoint.getIP(result) should've thrown an error because the URL is invalid, thus cannot be parsed")
	}
}

func TestEndpoint_NeedsToReadBody(t *testing.T) {
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
