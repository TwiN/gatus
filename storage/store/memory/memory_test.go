package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/util"
)

var (
	firstCondition  = core.Condition("[STATUS] == 200")
	secondCondition = core.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = core.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	timestamp = time.Now()

	testService = core.Service{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []*core.Condition{&firstCondition, &secondCondition, &thirdCondition},
		Alerts:                  nil,
		Insecure:                false,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuccessfulResult = core.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Body:                  []byte("body"),
		Errors:                nil,
		Connected:             true,
		Success:               true,
		Timestamp:             timestamp,
		Duration:              150 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*core.ConditionResult{
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
			{
				Condition: "[RESPONSE_TIME] < 500",
				Success:   true,
			},
			{
				Condition: "[CERTIFICATE_EXPIRATION] < 72h",
				Success:   true,
			},
		},
	}
	testUnsuccessfulResult = core.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Body:                  []byte("body"),
		Errors:                []string{"error-1", "error-2"},
		Connected:             true,
		Success:               false,
		Timestamp:             timestamp,
		Duration:              750 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*core.ConditionResult{
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
			{
				Condition: "[RESPONSE_TIME] < 500",
				Success:   false,
			},
			{
				Condition: "[CERTIFICATE_EXPIRATION] < 72h",
				Success:   false,
			},
		},
	}
)

func TestStore_Insert(t *testing.T) {
	store, _ := NewStore("")
	store.Insert(&testService, &testSuccessfulResult)
	store.Insert(&testService, &testUnsuccessfulResult)

	if store.cache.Count() != 1 {
		t.Fatalf("expected 1 ServiceStatus, got %d", store.cache.Count())
	}
	key := fmt.Sprintf("%s_%s", testService.Group, testService.Name)
	serviceStatus := store.GetServiceStatusByKey(key)
	if serviceStatus == nil {
		t.Fatalf("Store should've had key '%s', but didn't", key)
	}
	if len(serviceStatus.Results) != 2 {
		t.Fatalf("Service '%s' should've had 2 results, but actually returned %d", serviceStatus.Name, len(serviceStatus.Results))
	}
	for i, r := range serviceStatus.Results {
		expectedResult := store.GetServiceStatus(testService.Group, testService.Name).Results[i]
		if r.HTTPStatus != expectedResult.HTTPStatus {
			t.Errorf("Result at index %d should've had a HTTPStatus of %d, but was actually %d", i, expectedResult.HTTPStatus, r.HTTPStatus)
		}
		if r.DNSRCode != expectedResult.DNSRCode {
			t.Errorf("Result at index %d should've had a DNSRCode of %s, but was actually %s", i, expectedResult.DNSRCode, r.DNSRCode)
		}
		if len(r.Body) != len(expectedResult.Body) {
			t.Errorf("Result at index %d should've had a body of length %d, but was actually %d", i, len(expectedResult.Body), len(r.Body))
		}
		if r.Hostname != expectedResult.Hostname {
			t.Errorf("Result at index %d should've had a Hostname of %s, but was actually %s", i, expectedResult.Hostname, r.Hostname)
		}
		if r.IP != expectedResult.IP {
			t.Errorf("Result at index %d should've had a IP of %s, but was actually %s", i, expectedResult.IP, r.IP)
		}
		if r.Connected != expectedResult.Connected {
			t.Errorf("Result at index %d should've had a Connected value of %t, but was actually %t", i, expectedResult.Connected, r.Connected)
		}
		if r.Duration != expectedResult.Duration {
			t.Errorf("Result at index %d should've had a Duration of %s, but was actually %s", i, expectedResult.Duration.String(), r.Duration.String())
		}
		if len(r.Errors) != len(expectedResult.Errors) {
			t.Errorf("Result at index %d should've had %d errors, but actually had %d errors", i, len(expectedResult.Errors), len(r.Errors))
		}
		if len(r.ConditionResults) != len(expectedResult.ConditionResults) {
			t.Errorf("Result at index %d should've had %d ConditionResults, but actually had %d ConditionResults", i, len(expectedResult.ConditionResults), len(r.ConditionResults))
		}
		if r.Success != expectedResult.Success {
			t.Errorf("Result at index %d should've had a Success of %t, but was actually %t", i, expectedResult.Success, r.Success)
		}
		if r.Timestamp != expectedResult.Timestamp {
			t.Errorf("Result at index %d should've had a Timestamp of %s, but was actually %s", i, expectedResult.Timestamp.String(), r.Timestamp.String())
		}
		if r.CertificateExpiration != expectedResult.CertificateExpiration {
			t.Errorf("Result at index %d should've had a CertificateExpiration of %s, but was actually %s", i, expectedResult.CertificateExpiration.String(), r.CertificateExpiration.String())
		}
	}
}

func TestStore_GetServiceStatus(t *testing.T) {
	store, _ := NewStore("")
	store.Insert(&testService, &testSuccessfulResult)
	store.Insert(&testService, &testUnsuccessfulResult)

	serviceStatus := store.GetServiceStatus(testService.Group, testService.Name)
	if serviceStatus == nil {
		t.Fatalf("serviceStatus shouldn't have been nil")
	}
	if serviceStatus.Uptime == nil {
		t.Fatalf("serviceStatus.Uptime shouldn't have been nil")
	}
	if serviceStatus.Uptime.LastHour != 0.5 {
		t.Errorf("serviceStatus.Uptime.LastHour should've been 0.5")
	}
	if serviceStatus.Uptime.LastTwentyFourHours != 0.5 {
		t.Errorf("serviceStatus.Uptime.LastTwentyFourHours should've been 0.5")
	}
	if serviceStatus.Uptime.LastSevenDays != 0.5 {
		t.Errorf("serviceStatus.Uptime.LastSevenDays should've been 0.5")
	}
}

func TestStore_GetServiceStatusForMissingStatusReturnsNil(t *testing.T) {
	store, _ := NewStore("")
	store.Insert(&testService, &testSuccessfulResult)

	serviceStatus := store.GetServiceStatus("nonexistantgroup", "nonexistantname")
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, testService.Name)
	}
	serviceStatus = store.GetServiceStatus(testService.Group, "nonexistantname")
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, "nonexistantname")
	}
	serviceStatus = store.GetServiceStatus("nonexistantgroup", testService.Name)
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", "nonexistantgroup", testService.Name)
	}
}

func TestStore_GetServiceStatusByKey(t *testing.T) {
	store, _ := NewStore("")
	store.Insert(&testService, &testSuccessfulResult)
	store.Insert(&testService, &testUnsuccessfulResult)

	serviceStatus := store.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(testService.Group, testService.Name))
	if serviceStatus == nil {
		t.Fatalf("serviceStatus shouldn't have been nil")
	}
	if serviceStatus.Uptime == nil {
		t.Fatalf("serviceStatus.Uptime shouldn't have been nil")
	}
	if serviceStatus.Uptime.LastHour != 0.5 {
		t.Errorf("serviceStatus.Uptime.LastHour should've been 0.5")
	}
	if serviceStatus.Uptime.LastTwentyFourHours != 0.5 {
		t.Errorf("serviceStatus.Uptime.LastTwentyFourHours should've been 0.5")
	}
	if serviceStatus.Uptime.LastSevenDays != 0.5 {
		t.Errorf("serviceStatus.Uptime.LastSevenDays should've been 0.5")
	}
}

func TestStore_GetAllAsJSON(t *testing.T) {
	store, _ := NewStore("")
	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult
	store.Insert(&testService, firstResult)
	store.Insert(&testService, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}
	output, err := store.GetAllAsJSON()
	if err != nil {
		t.Fatal("shouldn't have returned an error, got", err.Error())
	}
	expectedOutput := `{"group_name":{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"errors":null,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}}`
	if string(output) != expectedOutput {
		t.Errorf("expected:\n %s\n\ngot:\n %s", expectedOutput, string(output))
	}
}

func TestStore_DeleteAllServiceStatusesNotInKeys(t *testing.T) {
	store, _ := NewStore("")
	firstService := core.Service{Name: "service-1", Group: "group"}
	secondService := core.Service{Name: "service-2", Group: "group"}
	result := &testSuccessfulResult
	store.Insert(&firstService, result)
	store.Insert(&secondService, result)
	if store.cache.Count() != 2 {
		t.Errorf("expected cache to have 2 keys, got %d", store.cache.Count())
	}
	if store.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(firstService.Group, firstService.Name)) == nil {
		t.Fatal("firstService should exist")
	}
	if store.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(secondService.Group, secondService.Name)) == nil {
		t.Fatal("secondService should exist")
	}
	store.DeleteAllServiceStatusesNotInKeys([]string{util.ConvertGroupAndServiceToKey(firstService.Group, firstService.Name)})
	if store.cache.Count() != 1 {
		t.Fatalf("expected cache to have 1 keys, got %d", store.cache.Count())
	}
	if store.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(firstService.Group, firstService.Name)) == nil {
		t.Error("secondService should've been deleted")
	}
	if store.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(secondService.Group, secondService.Name)) != nil {
		t.Error("firstService should still exist")
	}
}

func TestStore_Save(t *testing.T) {
	files := []string{
		"",
		t.TempDir() + "/test.db",
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			store, err := NewStore(file)
			if err != nil {
				t.Fatal("expected no error, got", err.Error())
			}
			err = store.Save()
			if err != nil {
				t.Fatal("expected no error, got", err.Error())
			}
		})
	}
}
