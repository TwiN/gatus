package database

import (
	"fmt"
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

func TestNewStore(t *testing.T) {
	if _, err := NewStore("", "TestNewStore.db"); err != ErrDatabaseDriverNotSpecified {
		t.Error("expected error due to blank driver parameter")
	}
	if _, err := NewStore("sqlite", ""); err != ErrFilePathNotSpecified {
		t.Error("expected error due to blank path parameter")
	}
	if store, err := NewStore("sqlite", t.TempDir()+"/TestNewStore.db"); err != nil {
		t.Error("shouldn't have returned any error, got", err.Error())
	} else {
		_ = store.db.Close()
	}
}

func TestStore_Insert(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_Insert.db")
	defer store.db.Close()
	store.Insert(&testService, &testSuccessfulResult)
	store.Insert(&testService, &testUnsuccessfulResult)

	key := fmt.Sprintf("%s_%s", testService.Group, testService.Name)
	serviceStatus := store.GetServiceStatusByKey(key, paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
	if serviceStatus == nil {
		t.Fatalf("Store should've had key '%s', but didn't", key)
	}
	if len(serviceStatus.Results) != 2 {
		t.Fatalf("Service '%s' should've had 2 results, but actually returned %d", serviceStatus.Name, len(serviceStatus.Results))
	}
	for i, r := range serviceStatus.Results {
		expectedResult := store.GetServiceStatus(testService.Group, testService.Name, paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime()).Results[i]
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

func TestStore_GetServiceStatus(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_GetServiceStatus.db")
	defer store.db.Close()
	store.Insert(&testService, &testSuccessfulResult)
	store.Insert(&testService, &testUnsuccessfulResult)

	serviceStatus := store.GetServiceStatus(testService.Group, testService.Name, paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
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
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_GetServiceStatusForMissingStatusReturnsNil.db")
	defer store.db.Close()
	store.Insert(&testService, &testSuccessfulResult)

	serviceStatus := store.GetServiceStatus("nonexistantgroup", "nonexistantname", paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, testService.Name)
	}
	serviceStatus = store.GetServiceStatus(testService.Group, "nonexistantname", paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", testService.Group, "nonexistantname")
	}
	serviceStatus = store.GetServiceStatus("nonexistantgroup", testService.Name, paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
	if serviceStatus != nil {
		t.Errorf("Returned service status for group '%s' and name '%s' not nil after inserting the service into the store", "nonexistantgroup", testService.Name)
	}
}

func TestStore_GetServiceStatusByKey(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_GetServiceStatusByKey.db")
	defer store.db.Close()
	store.Insert(&testService, &testSuccessfulResult)
	store.Insert(&testService, &testUnsuccessfulResult)

	serviceStatus := store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithEvents(1, core.MaximumNumberOfEvents).WithResults(1, core.MaximumNumberOfResults).WithUptime())
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

func TestStore_GetAllServiceStatuses(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_GetAllServiceStatuses.db")
	defer store.db.Close()
	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult
	store.Insert(&testService, firstResult)
	store.Insert(&testService, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}
	serviceStatuses := store.GetAllServiceStatuses(paging.NewServiceStatusParams().WithResults(1, 20))
	if len(serviceStatuses) != 1 {
		t.Fatal("expected 1 service status")
	}
	actual, exists := serviceStatuses[testService.Key()]
	if !exists {
		t.Fatal("expected service status to exist")
	}
	if len(actual.Results) != 2 {
		t.Error("expected 2 results, got", len(actual.Results))
	}
	if len(actual.Events) != 0 {
		t.Error("expected 0 events, got", len(actual.Events))
	}
}

func TestStore_InsertCleansUpOldUptimeEntriesProperly(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InsertCleansUpOldUptimeEntriesProperly.db")
	defer store.db.Close()
	now := time.Now().Round(time.Minute)
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	store.Insert(&testService, &core.Result{Timestamp: now.Add(-5 * time.Hour), Success: true})

	tx, _ := store.db.Begin()
	oldest, _ := store.getAgeOfOldestServiceUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 5*time.Hour {
		t.Errorf("oldest service uptime entry should've been ~5 hours old, was %s", oldest)
	}

	// The oldest cache entry should remain at ~5 hours old, because this entry is more recent
	store.Insert(&testService, &core.Result{Timestamp: now.Add(-3 * time.Hour), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestServiceUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 5*time.Hour {
		t.Errorf("oldest service uptime entry should've been ~5 hours old, was %s", oldest)
	}

	// The oldest cache entry should now become at ~8 hours old, because this entry is older
	store.Insert(&testService, &core.Result{Timestamp: now.Add(-8 * time.Hour), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestServiceUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 8*time.Hour {
		t.Errorf("oldest service uptime entry should've been ~8 hours old, was %s", oldest)
	}

	// Since this is one hour before reaching the clean up threshold, the oldest entry should now be this one
	store.Insert(&testService, &core.Result{Timestamp: now.Add(-(uptimeCleanUpThreshold - time.Hour)), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestServiceUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != uptimeCleanUpThreshold-time.Hour {
		t.Errorf("oldest service uptime entry should've been ~%s hours old, was %s", uptimeCleanUpThreshold-time.Hour, oldest)
	}

	// Since this entry is after the uptimeCleanUpThreshold, both this entry as well as the previous
	// one should be deleted since they both surpass uptimeRetention
	store.Insert(&testService, &core.Result{Timestamp: now.Add(-(uptimeCleanUpThreshold + time.Hour)), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestServiceUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 8*time.Hour {
		t.Errorf("oldest service uptime entry should've been ~8 hours old, was %s", oldest)
	}
}

func TestStore_InsertCleansUpProperly(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_deleteOldServiceResults.db")
	defer store.db.Close()
	for i := 0; i < resultsCleanUpThreshold+eventsCleanUpThreshold; i++ {
		store.Insert(&testService, &testSuccessfulResult)
		store.Insert(&testService, &testUnsuccessfulResult)
		ss := store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithResults(1, core.MaximumNumberOfResults*5).WithEvents(1, core.MaximumNumberOfEvents*5))
		if len(ss.Results) > resultsCleanUpThreshold+1 {
			t.Errorf("number of results shouldn't have exceeded %d, reached %d", resultsCleanUpThreshold, len(ss.Results))
		}
		if len(ss.Events) > eventsCleanUpThreshold+1 {
			t.Errorf("number of events shouldn't have exceeded %d, reached %d", eventsCleanUpThreshold, len(ss.Events))
		}
	}
}
