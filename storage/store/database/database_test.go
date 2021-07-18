package database

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

	now = time.Now()

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
		Timestamp:             now,
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
		Timestamp:             now,
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

func TestStore_InsertCleansUpOldUptimeEntriesProperly(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InsertCleansUpOldUptimeEntriesProperly.db")
	defer store.Close()
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

func TestStore_InsertCleansUpEventsAndResultsProperly(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InsertCleansUpEventsAndResultsProperly.db")
	defer store.Close()
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
	store.Clear()
}

func TestStore_Persistence(t *testing.T) {
	file := t.TempDir() + "/TestStore_Persistence.db"
	store, _ := NewStore("sqlite", file)
	store.Insert(&testService, &testSuccessfulResult)
	store.Insert(&testService, &testUnsuccessfulResult)
	ssFromOldStore := store.GetServiceStatus(testService.Group, testService.Name, paging.NewServiceStatusParams().WithResults(1, core.MaximumNumberOfResults).WithEvents(1, core.MaximumNumberOfEvents).WithUptime())
	if ssFromOldStore == nil || ssFromOldStore.Group != "group" || ssFromOldStore.Name != "name" || len(ssFromOldStore.Events) != 3 || len(ssFromOldStore.Results) != 2 || ssFromOldStore.Uptime.LastHour != 0.5 || ssFromOldStore.Uptime.LastTwentyFourHours != 0.5 || ssFromOldStore.Uptime.LastSevenDays != 0.5 {
		store.Close()
		t.Fatal("sanity check failed")
	}
	store.Close()
	store, _ = NewStore("sqlite", file)
	defer store.Close()
	ssFromNewStore := store.GetServiceStatus(testService.Group, testService.Name, paging.NewServiceStatusParams().WithResults(1, core.MaximumNumberOfResults).WithEvents(1, core.MaximumNumberOfEvents).WithUptime())
	if ssFromNewStore == nil || ssFromNewStore.Group != "group" || ssFromNewStore.Name != "name" || len(ssFromNewStore.Events) != 3 || len(ssFromNewStore.Results) != 2 || ssFromNewStore.Uptime.LastHour != 0.5 || ssFromNewStore.Uptime.LastTwentyFourHours != 0.5 || ssFromNewStore.Uptime.LastSevenDays != 0.5 {
		t.Fatal("failed sanity check")
	}
	if ssFromNewStore == ssFromOldStore {
		t.Fatal("ss from the old and new store should have a different memory address")
	}
	for i := range ssFromNewStore.Events {
		if ssFromNewStore.Events[i].Timestamp != ssFromOldStore.Events[i].Timestamp {
			t.Error("new and old should've been the same")
		}
		if ssFromNewStore.Events[i].Type != ssFromOldStore.Events[i].Type {
			t.Error("new and old should've been the same")
		}
	}
	for i := range ssFromOldStore.Results {
		if ssFromNewStore.Results[i].Timestamp != ssFromOldStore.Results[i].Timestamp {
			t.Error("new and old should've been the same")
		}
		if ssFromNewStore.Results[i].Success != ssFromOldStore.Results[i].Success {
			t.Error("new and old should've been the same")
		}
		if ssFromNewStore.Results[i].Connected != ssFromOldStore.Results[i].Connected {
			t.Error("new and old should've been the same")
		}
		if ssFromNewStore.Results[i].IP != ssFromOldStore.Results[i].IP {
			t.Error("new and old should've been the same")
		}
		if ssFromNewStore.Results[i].Hostname != ssFromOldStore.Results[i].Hostname {
			t.Error("new and old should've been the same")
		}
		if ssFromNewStore.Results[i].HTTPStatus != ssFromOldStore.Results[i].HTTPStatus {
			t.Error("new and old should've been the same")
		}
		if ssFromNewStore.Results[i].DNSRCode != ssFromOldStore.Results[i].DNSRCode {
			t.Error("new and old should've been the same")
		}
		if len(ssFromNewStore.Results[i].Errors) != len(ssFromOldStore.Results[i].Errors) {
			t.Error("new and old should've been the same")
		} else {
			for j := range ssFromOldStore.Results[i].Errors {
				if ssFromNewStore.Results[i].Errors[j] != ssFromOldStore.Results[i].Errors[j] {
					t.Error("new and old should've been the same")
				}
			}
		}
		if len(ssFromNewStore.Results[i].ConditionResults) != len(ssFromOldStore.Results[i].ConditionResults) {
			t.Error("new and old should've been the same")
		} else {
			for j := range ssFromOldStore.Results[i].ConditionResults {
				if ssFromNewStore.Results[i].ConditionResults[j].Condition != ssFromOldStore.Results[i].ConditionResults[j].Condition {
					t.Error("new and old should've been the same")
				}
				if ssFromNewStore.Results[i].ConditionResults[j].Success != ssFromOldStore.Results[i].ConditionResults[j].Success {
					t.Error("new and old should've been the same")
				}
			}
		}
	}
}

func TestStore_Save(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_Save.db")
	defer store.Close()
	if store.Save() != nil {
		t.Error("Save shouldn't do anything for this store")
	}
}

// TestStore_InvalidTransaction tests what happens if an invalid transaction is passed as parameter
func TestStore_InvalidTransaction(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InvalidTransaction.db")
	defer store.Close()
	tx, _ := store.db.Begin()
	tx.Commit()
	if _, err := store.insertService(tx, &testService); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.insertEvent(tx, 1, core.NewEventFromResult(&testSuccessfulResult)); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.insertResult(tx, 1, &testSuccessfulResult); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.insertConditionResults(tx, 1, testSuccessfulResult.ConditionResults); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.updateServiceUptime(tx, 1, &testSuccessfulResult); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getAllServiceKeys(tx); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getServiceStatusByKey(tx, testService.Key(), paging.NewServiceStatusParams().WithResults(1, 20)); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getEventsByServiceID(tx, 1, 1, 50); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getResultsByServiceID(tx, 1, 1, 50); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.deleteOldServiceEvents(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.deleteOldServiceResults(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, _, err := store.getServiceUptime(tx, 1, time.Now(), time.Now()); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getServiceID(tx, &testService); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getNumberOfEventsByServiceID(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getNumberOfResultsByServiceID(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getAgeOfOldestServiceUptimeEntry(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getLastServiceResultSuccessValue(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
}

func TestStore_NoRows(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_NoRows.db")
	defer store.Close()
	tx, _ := store.db.Begin()
	defer tx.Rollback()
	if _, err := store.getLastServiceResultSuccessValue(tx, 1); err != errNoRowsReturned {
		t.Errorf("should've %v, got %v", errNoRowsReturned, err)
	}
	if _, err := store.getAgeOfOldestServiceUptimeEntry(tx, 1); err != errNoRowsReturned {
		t.Errorf("should've %v, got %v", errNoRowsReturned, err)
	}
}
