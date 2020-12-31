package core

import (
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestService_ValidateAndSetDefaults(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		URL:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
		Alerts:     []*Alert{{Type: PagerDutyAlert}},
	}
	service.ValidateAndSetDefaults()
	if service.Method != "GET" {
		t.Error("Service method should've defaulted to GET")
	}
	if service.Interval != time.Minute {
		t.Error("Service interval should've defaulted to 1 minute")
	}
	if service.Headers == nil {
		t.Error("Service headers should've defaulted to an empty map")
	}
	if len(service.Alerts) != 1 {
		t.Error("Service should've had 1 alert")
	}
	if service.Alerts[0].Enabled {
		t.Error("Service alert should've defaulted to disabled")
	}
	if service.Alerts[0].SuccessThreshold != 2 {
		t.Error("Service alert should've defaulted to a success threshold of 2")
	}
	if service.Alerts[0].FailureThreshold != 3 {
		t.Error("Service alert should've defaulted to a failure threshold of 3")
	}
}

func TestService_ValidateAndSetDefaultsWithNoName(t *testing.T) {
	defer func() { recover() }()
	condition := Condition("[STATUS] == 200")
	service := &Service{
		Name:       "",
		URL:        "http://example.com",
		Conditions: []*Condition{&condition},
	}
	service.ValidateAndSetDefaults()
	t.Fatal("Should've panicked because service didn't have a name, which is a mandatory field")
}

func TestService_ValidateAndSetDefaultsWithNoUrl(t *testing.T) {
	defer func() { recover() }()
	condition := Condition("[STATUS] == 200")
	service := &Service{
		Name:       "example",
		URL:        "",
		Conditions: []*Condition{&condition},
	}
	service.ValidateAndSetDefaults()
	t.Fatal("Should've panicked because service didn't have an url, which is a mandatory field")
}

func TestService_ValidateAndSetDefaultsWithNoConditions(t *testing.T) {
	defer func() { recover() }()
	service := &Service{
		Name:       "example",
		URL:        "http://example.com",
		Conditions: nil,
	}
	service.ValidateAndSetDefaults()
	t.Fatal("Should've panicked because service didn't have at least 1 condition")
}

func TestService_ValidateAndSetDefaultsWithDNS(t *testing.T) {
	conditionSuccess := Condition("[DNS_RCODE] == NOERROR")
	service := &Service{
		Name: "dns-test",
		URL:  "http://example.com",
		DNS: &DNS{
			QueryType: "A",
			QueryName: "example.com",
		},
		Conditions: []*Condition{&conditionSuccess},
	}
	service.ValidateAndSetDefaults()
	if service.DNS.QueryName != "example.com." {
		t.Error("Service.dns.query-name should be formatted with . suffix")
	}
}

func TestService_GetAlertsTriggered(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		URL:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
		Alerts:     []*Alert{{Type: PagerDutyAlert, Enabled: true}},
	}
	service.ValidateAndSetDefaults()
	if service.NumberOfFailuresInARow != 0 {
		t.Error("Service.NumberOfFailuresInARow should start with 0")
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Error("Service.NumberOfSuccessesInARow should start with 0")
	}
	if len(service.GetAlertsTriggered()) > 0 {
		t.Error("No alerts should've been triggered, because service.NumberOfFailuresInARow is 0, which is below the failure threshold")
	}
	service.NumberOfFailuresInARow = service.Alerts[0].FailureThreshold
	if len(service.GetAlertsTriggered()) != 1 {
		t.Error("Alert should've been triggered")
	}
}

func TestService_buildHTTPRequest(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		URL:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
	}
	service.ValidateAndSetDefaults()
	request := service.buildHTTPRequest()
	if request.Method != "GET" {
		t.Error("request.Method should've been GET, but was", request.Method)
	}
	if request.Host != "twinnation.org" {
		t.Error("request.Host should've been twinnation.org, but was", request.Host)
	}
	if userAgent := request.Header.Get("User-Agent"); userAgent != GatusUserAgent {
		t.Errorf("request.Header.Get(User-Agent) should've been %s, but was %s", GatusUserAgent, userAgent)
	}
}

func TestService_buildHTTPRequestWithCustomUserAgent(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		URL:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
		Headers: map[string]string{
			"User-Agent": "Test/2.0",
		},
	}
	service.ValidateAndSetDefaults()
	request := service.buildHTTPRequest()
	if request.Method != "GET" {
		t.Error("request.Method should've been GET, but was", request.Method)
	}
	if request.Host != "twinnation.org" {
		t.Error("request.Host should've been twinnation.org, but was", request.Host)
	}
	if userAgent := request.Header.Get("User-Agent"); userAgent != "Test/2.0" {
		t.Errorf("request.Header.Get(User-Agent) should've been %s, but was %s", "Test/2.0", userAgent)
	}
}

func TestService_buildHTTPRequestWithHostHeader(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		URL:        "https://twinnation.org/health",
		Method:     "POST",
		Conditions: []*Condition{&condition},
		Headers: map[string]string{
			"Host": "example.com",
		},
	}
	service.ValidateAndSetDefaults()
	request := service.buildHTTPRequest()
	if request.Method != "POST" {
		t.Error("request.Method should've been POST, but was", request.Method)
	}
	if request.Host != "example.com" {
		t.Error("request.Host should've been example.com, but was", request.Host)
	}
}

func TestService_buildHTTPRequestWithGraphQLEnabled(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		URL:        "https://twinnation.org/graphql",
		Method:     "POST",
		Conditions: []*Condition{&condition},
		GraphQL:    true,
		Body: `{
  user(gender: "female") {
    id
    name
    gender
    avatar
  }
}`,
	}
	service.ValidateAndSetDefaults()
	request := service.buildHTTPRequest()
	if request.Method != "POST" {
		t.Error("request.Method should've been POST, but was", request.Method)
	}
	if contentType := request.Header.Get(ContentTypeHeader); contentType != "application/json" {
		t.Error("request.Header.Content-Type should've been application/json, but was", contentType)
	}
	body, _ := ioutil.ReadAll(request.Body)
	if !strings.HasPrefix(string(body), "{\"query\":") {
		t.Error("request.Body should've started with '{\"query\":', but it didn't:", string(body))
	}
}

func TestIntegrationEvaluateHealth(t *testing.T) {
	condition := Condition("[STATUS] == 200")
	service := Service{
		Name:       "TwiNNatioN",
		URL:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
	}
	result := service.EvaluateHealth()
	if !result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a success", condition)
	}
	if !result.Connected {
		t.Error("Because the connection has been established, result.Connected should've been true")
	}
	if !result.Success {
		t.Error("Because all conditions passed, this should have been a success")
	}
}

func TestIntegrationEvaluateHealthWithFailure(t *testing.T) {
	condition := Condition("[STATUS] == 500")
	service := Service{
		Name:       "TwiNNatioN",
		URL:        "https://twinnation.org/health",
		Conditions: []*Condition{&condition},
	}
	result := service.EvaluateHealth()
	if result.ConditionResults[0].Success {
		t.Errorf("Condition '%s' should have been a failure", condition)
	}
	if !result.Connected {
		t.Error("Because the connection has been established, result.Connected should've been true")
	}
	if result.Success {
		t.Error("Because one of the conditions failed, success should have been false")
	}
}

func TestIntegrationEvaluateHealthForDNS(t *testing.T) {
	conditionSuccess := Condition("[DNS_RCODE] == NOERROR")
	conditionBody := Condition("[BODY] == 93.184.216.34")
	service := Service{
		Name: "TwiNNatioN",
		URL:  "8.8.8.8",
		DNS: &DNS{
			QueryType: "A",
			QueryName: "example.com.",
		},
		Conditions: []*Condition{&conditionSuccess, &conditionBody},
	}
	result := service.EvaluateHealth()
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
	service := Service{
		Name:       "ICMP test",
		URL:        "icmp://127.0.0.1",
		Conditions: []*Condition{&conditionSuccess},
	}
	result := service.EvaluateHealth()
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

func TestService_getIP(t *testing.T) {
	conditionSuccess := Condition("[CONNECTED] == true")
	service := Service{
		Name:       "Invalid URL test",
		URL:        "",
		Conditions: []*Condition{&conditionSuccess},
	}
	result := &Result{}
	service.getIP(result)
	if len(result.Errors) == 0 {
		t.Error("service.getIP(result) should've thrown an error because the URL is invalid, thus cannot be parsed")
	}
}
