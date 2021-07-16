package memory

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/paging"
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
	serviceStatus := store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithResults(1, 20))
	if serviceStatus == nil {
		t.Fatalf("Store should've had key '%s', but didn't", testService.Key())
	}
	if len(serviceStatus.Results) != 2 {
		t.Fatalf("Service '%s' should've had 2 results, but actually returned %d", serviceStatus.Name, len(serviceStatus.Results))
	}
	for i, r := range serviceStatus.Results {
		expectedResult := store.GetServiceStatus(testService.Group, testService.Name, paging.NewServiceStatusParams().WithResults(1, 20)).Results[i]
		if r.HTTPStatus != expectedResult.HTTPStatus {
			t.Errorf("Result at index %d should've had a HTTPStatus of %d, but was actually %d", i, expectedResult.HTTPStatus, r.HTTPStatus)
		}
		if r.DNSRCode != expectedResult.DNSRCode {
			t.Errorf("Result at index %d should've had a DNSRCode of %s, but was actually %s", i, expectedResult.DNSRCode, r.DNSRCode)
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
