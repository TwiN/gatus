package core

import (
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

func TestService_ValidateAndSetDefaultsWithNoDNSQueryName(t *testing.T) {
	defer func() { recover() }()
	service := &Service{
		Name: "",
		URL:  "http://example.com",
		DNS: &DNS{
			QueryType: "A",
			QueryName: "",
		},
	}
	service.ValidateAndSetDefaults()
	t.Fatal("Should've panicked because service`s dns didn't have a query name, which is a mandatory field for dns")
}

func TestService_ValidateAndSetDefaultsWithInvalidDNSQueryType(t *testing.T) {
	defer func() { recover() }()
	service := &Service{
		Name: "",
		URL:  "http://example.com",
		DNS: &DNS{
			QueryType: "B",
			QueryName: "example.com",
		},
	}
	service.ValidateAndSetDefaults()
	t.Fatal("Should've panicked because service`s dns query type is invalid, it needs to be a valid query name like A, AAAA, CNAME...")
}

func TestService_ValidateAndSetDefaultsWithDNS(t *testing.T) {
	service := &Service{
		Name: "",
		URL:  "http://example.com",
		DNS: &DNS{
			QueryType: "A",
			QueryName: "example.com",
		},
	}
	service.ValidateAndSetDefaults()
	if service.DNS.QueryName == "example.com." {
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
