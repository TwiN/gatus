package sql

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

var (
	firstCondition  = core.Condition("[STATUS] == 200")
	secondCondition = core.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = core.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	now = time.Now()

	testEndpoint = core.Endpoint{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []core.Condition{firstCondition, secondCondition, thirdCondition},
		Alerts:                  nil,
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
	if _, err := NewStore("", "TestNewStore.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents); err != ErrDatabaseDriverNotSpecified {
		t.Error("expected error due to blank driver parameter")
	}
	if _, err := NewStore("sqlite", "", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents); err != ErrPathNotSpecified {
		t.Error("expected error due to blank path parameter")
	}
	if store, err := NewStore("sqlite", t.TempDir()+"/TestNewStore.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents); err != nil {
		t.Error("shouldn't have returned any error, got", err.Error())
	} else {
		_ = store.db.Close()
	}
}

func TestStore_InsertCleansUpOldUptimeEntriesProperly(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InsertCleansUpOldUptimeEntriesProperly.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	now := time.Now().Truncate(time.Hour)
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	store.Insert(&testEndpoint, &core.Result{Timestamp: now.Add(-5 * time.Hour), Success: true})

	tx, _ := store.db.Begin()
	oldest, _ := store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 5*time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~5 hours old, was %s", oldest)
	}

	// The oldest cache entry should remain at ~5 hours old, because this entry is more recent
	store.Insert(&testEndpoint, &core.Result{Timestamp: now.Add(-3 * time.Hour), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 5*time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~5 hours old, was %s", oldest)
	}

	// The oldest cache entry should now become at ~8 hours old, because this entry is older
	store.Insert(&testEndpoint, &core.Result{Timestamp: now.Add(-8 * time.Hour), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 8*time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~8 hours old, was %s", oldest)
	}

	// Since this is one hour before reaching the clean up threshold, the oldest entry should now be this one
	store.Insert(&testEndpoint, &core.Result{Timestamp: now.Add(-(uptimeCleanUpThreshold - time.Hour)), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != uptimeCleanUpThreshold-time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~%s hours old, was %s", uptimeCleanUpThreshold-time.Hour, oldest)
	}

	// Since this entry is after the uptimeCleanUpThreshold, both this entry as well as the previous
	// one should be deleted since they both surpass uptimeRetention
	store.Insert(&testEndpoint, &core.Result{Timestamp: now.Add(-(uptimeCleanUpThreshold + time.Hour)), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 8*time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~8 hours old, was %s", oldest)
	}
}

func TestStore_InsertCleansUpEventsAndResultsProperly(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InsertCleansUpEventsAndResultsProperly.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	resultsCleanUpThreshold := store.maximumNumberOfResults + resultsAboveMaximumCleanUpThreshold
	eventsCleanUpThreshold := store.maximumNumberOfEvents + eventsAboveMaximumCleanUpThreshold
	for i := 0; i < resultsCleanUpThreshold+eventsCleanUpThreshold; i++ {
		store.Insert(&testEndpoint, &testSuccessfulResult)
		store.Insert(&testEndpoint, &testUnsuccessfulResult)
		ss, _ := store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults*5).WithEvents(1, storage.DefaultMaximumNumberOfEvents*5))

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
	path := t.TempDir() + "/TestStore_Persistence.db"
	store, _ := NewStore("sqlite", path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	store.Insert(&testEndpoint, &testSuccessfulResult)
	store.Insert(&testEndpoint, &testUnsuccessfulResult)
	if uptime, _ := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); uptime != 0.5 {
		t.Errorf("the uptime over the past 1h should've been 0.5, got %f", uptime)
	}
	if uptime, _ := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour*24), time.Now()); uptime != 0.5 {
		t.Errorf("the uptime over the past 24h should've been 0.5, got %f", uptime)
	}
	if uptime, _ := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour*24*7), time.Now()); uptime != 0.5 {
		t.Errorf("the uptime over the past 7d should've been 0.5, got %f", uptime)
	}
	ssFromOldStore, _ := store.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults).WithEvents(1, storage.DefaultMaximumNumberOfEvents))
	if ssFromOldStore == nil || ssFromOldStore.Group != "group" || ssFromOldStore.Name != "name" || len(ssFromOldStore.Events) != 3 || len(ssFromOldStore.Results) != 2 {
		store.Close()
		t.Fatal("sanity check failed")
	}
	store.Close()
	store, _ = NewStore("sqlite", path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	ssFromNewStore, _ := store.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults).WithEvents(1, storage.DefaultMaximumNumberOfEvents))
	if ssFromNewStore == nil || ssFromNewStore.Group != "group" || ssFromNewStore.Name != "name" || len(ssFromNewStore.Events) != 3 || len(ssFromNewStore.Results) != 2 {
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
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_Save.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	if store.Save() != nil {
		t.Error("Save shouldn't do anything for this store")
	}
}

// Note that are much more extensive tests in /storage/store/store_test.go.
// This test is simply an extra sanity check
func TestStore_SanityCheck(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_SanityCheck.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	store.Insert(&testEndpoint, &testSuccessfulResult)
	endpointStatuses, _ := store.GetAllEndpointStatuses(paging.NewEndpointStatusParams())
	if numberOfEndpointStatuses := len(endpointStatuses); numberOfEndpointStatuses != 1 {
		t.Fatalf("expected 1 EndpointStatus, got %d", numberOfEndpointStatuses)
	}
	store.Insert(&testEndpoint, &testUnsuccessfulResult)
	// Both results inserted are for the same endpoint, therefore, the count shouldn't have increased
	endpointStatuses, _ = store.GetAllEndpointStatuses(paging.NewEndpointStatusParams())
	if numberOfEndpointStatuses := len(endpointStatuses); numberOfEndpointStatuses != 1 {
		t.Fatalf("expected 1 EndpointStatus, got %d", numberOfEndpointStatuses)
	}
	if hourlyAverageResponseTime, err := store.GetHourlyAverageResponseTimeByKey(testEndpoint.Key(), time.Now().Add(-24*time.Hour), time.Now()); err != nil {
		t.Errorf("expected no error, got %v", err)
	} else if len(hourlyAverageResponseTime) != 1 {
		t.Errorf("expected 1 hour to have had a result in the past 24 hours, got %d", len(hourlyAverageResponseTime))
	}
	if uptime, _ := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-24*time.Hour), time.Now()); uptime != 0.5 {
		t.Errorf("expected uptime of last 24h to be 0.5, got %f", uptime)
	}
	if averageResponseTime, _ := store.GetAverageResponseTimeByKey(testEndpoint.Key(), time.Now().Add(-24*time.Hour), time.Now()); averageResponseTime != 450 {
		t.Errorf("expected average response time of last 24h to be 450, got %d", averageResponseTime)
	}
	ss, _ := store.GetEndpointStatus(testEndpoint.Group, testEndpoint.Name, paging.NewEndpointStatusParams().WithResults(1, 20).WithEvents(1, 20))
	if ss == nil {
		t.Fatalf("Store should've had key '%s', but didn't", testEndpoint.Key())
	}
	if len(ss.Events) != 3 {
		t.Errorf("Endpoint '%s' should've had 3 events, got %d", ss.Name, len(ss.Events))
	}
	if len(ss.Results) != 2 {
		t.Errorf("Endpoint '%s' should've had 2 results, got %d", ss.Name, len(ss.Results))
	}
	if deleted := store.DeleteAllEndpointStatusesNotInKeys([]string{"invalid-key-which-means-everything-should-get-deleted"}); deleted != 1 {
		t.Errorf("%d entries should've been deleted, got %d", 1, deleted)
	}
	if deleted := store.DeleteAllEndpointStatusesNotInKeys([]string{}); deleted != 0 {
		t.Errorf("There should've been no entries left to delete, got %d", deleted)
	}
}

// TestStore_InvalidTransaction tests what happens if an invalid transaction is passed as parameter
func TestStore_InvalidTransaction(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InvalidTransaction.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	tx, _ := store.db.Begin()
	tx.Commit()
	if _, err := store.insertEndpoint(tx, &testEndpoint); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.insertEndpointEvent(tx, 1, core.NewEventFromResult(&testSuccessfulResult)); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.insertEndpointResult(tx, 1, &testSuccessfulResult); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.insertConditionResults(tx, 1, testSuccessfulResult.ConditionResults); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.updateEndpointUptime(tx, 1, &testSuccessfulResult); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getAllEndpointKeys(tx); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getEndpointStatusByKey(tx, testEndpoint.Key(), paging.NewEndpointStatusParams().WithResults(1, 20)); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getEndpointEventsByEndpointID(tx, 1, 1, 50); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getEndpointResultsByEndpointID(tx, 1, 1, 50); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.deleteOldEndpointEvents(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if err := store.deleteOldEndpointResults(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, _, err := store.getEndpointUptime(tx, 1, time.Now(), time.Now()); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getEndpointID(tx, &testEndpoint); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getNumberOfEventsByEndpointID(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getNumberOfResultsByEndpointID(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getAgeOfOldestEndpointUptimeEntry(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
	if _, err := store.getLastEndpointResultSuccessValue(tx, 1); err == nil {
		t.Error("should've returned an error, because the transaction was already committed")
	}
}

func TestStore_NoRows(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_NoRows.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	tx, _ := store.db.Begin()
	defer tx.Rollback()
	if _, err := store.getLastEndpointResultSuccessValue(tx, 1); err != errNoRowsReturned {
		t.Errorf("should've %v, got %v", errNoRowsReturned, err)
	}
	if _, err := store.getAgeOfOldestEndpointUptimeEntry(tx, 1); err != errNoRowsReturned {
		t.Errorf("should've %v, got %v", errNoRowsReturned, err)
	}
}

// This tests very unlikely cases where a table is deleted.
func TestStore_BrokenSchema(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_BrokenSchema.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if _, err := store.GetAverageResponseTimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if _, err := store.GetAllEndpointStatuses(paging.NewEndpointStatusParams()); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoints")
	// And now we'll try to insert something in our broken schema
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetAverageResponseTimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetHourlyAverageResponseTimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetAllEndpointStatuses(paging.NewEndpointStatusParams()); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams()); err == nil {
		t.Fatal("expected an error")
	}
	// Repair
	if err := store.createSchema(); err != nil {
		t.Fatal("schema should've been repaired")
	}
	store.Clear()
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoint_events")
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, because this should silently fails, got", err.Error())
	}
	if _, err := store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 1).WithEvents(1, 1)); err != nil {
		t.Fatal("expected no error, because this should silently fail, got", err.Error())
	}
	// Repair
	if err := store.createSchema(); err != nil {
		t.Fatal("schema should've been repaired")
	}
	store.Clear()
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoint_results")
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 1).WithEvents(1, 1)); err != nil {
		t.Fatal("expected no error, because this should silently fail, got", err.Error())
	}
	// Repair
	if err := store.createSchema(); err != nil {
		t.Fatal("schema should've been repaired")
	}
	store.Clear()
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoint_result_conditions")
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err == nil {
		t.Fatal("expected an error")
	}
	// Repair
	if err := store.createSchema(); err != nil {
		t.Fatal("schema should've been repaired")
	}
	store.Clear()
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoint_uptimes")
	if err := store.Insert(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, because this should silently fails, got", err.Error())
	}
	if _, err := store.GetAverageResponseTimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetHourlyAverageResponseTimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCacheKey(t *testing.T) {
	scenarios := []struct {
		endpointKey      string
		params           paging.EndpointStatusParams
		overrideCacheKey string
		expectedCacheKey string
		wantErr          bool
	}{
		{
			endpointKey:      "simple",
			params:           paging.EndpointStatusParams{EventsPage: 1, EventsPageSize: 2, ResultsPage: 3, ResultsPageSize: 4},
			expectedCacheKey: "simple-1-2-3-4",
			wantErr:          false,
		},
		{
			endpointKey:      "with-hyphen",
			params:           paging.EndpointStatusParams{EventsPage: 0, EventsPageSize: 0, ResultsPage: 1, ResultsPageSize: 20},
			expectedCacheKey: "with-hyphen-0-0-1-20",
			wantErr:          false,
		},
		{
			endpointKey:      "with-multiple-hyphens",
			params:           paging.EndpointStatusParams{EventsPage: 0, EventsPageSize: 0, ResultsPage: 2, ResultsPageSize: 20},
			expectedCacheKey: "with-multiple-hyphens-0-0-2-20",
			wantErr:          false,
		},
		{
			overrideCacheKey: "invalid-a-2-3-4",
			wantErr:          true,
		},
		{
			overrideCacheKey: "invalid-1-a-3-4",
			wantErr:          true,
		},
		{
			overrideCacheKey: "invalid-1-2-a-4",
			wantErr:          true,
		},
		{
			overrideCacheKey: "invalid-1-2-3-a",
			wantErr:          true,
		},
		{
			overrideCacheKey: "notenoughhyphen1-2-3-4",
			wantErr:          true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.expectedCacheKey+scenario.overrideCacheKey, func(t *testing.T) {
			var cacheKey string
			if len(scenario.overrideCacheKey) > 0 {
				cacheKey = scenario.overrideCacheKey
			} else {
				cacheKey = generateCacheKey(scenario.endpointKey, &scenario.params)
				if cacheKey != scenario.expectedCacheKey {
					t.Errorf("expected %s, got %s", scenario.expectedCacheKey, cacheKey)
				}
			}
			extractedEndpointKey, extractedParams, err := extractKeyAndParamsFromCacheKey(cacheKey)
			if (err != nil) != scenario.wantErr {
				t.Errorf("expected error %v, got %v", scenario.wantErr, err)
				return
			}
			if err != nil {
				// If there's an error, we don't need to check the extracted values
				return
			}
			if extractedEndpointKey != scenario.endpointKey {
				t.Errorf("expected endpointKey %s, got %s", scenario.endpointKey, extractedEndpointKey)
			}
			if extractedParams.EventsPage != scenario.params.EventsPage {
				t.Errorf("expected EventsPage %d, got %d", scenario.params.EventsPage, extractedParams.EventsPage)
			}
			if extractedParams.EventsPageSize != scenario.params.EventsPageSize {
				t.Errorf("expected EventsPageSize %d, got %d", scenario.params.EventsPageSize, extractedParams.EventsPageSize)
			}
			if extractedParams.ResultsPage != scenario.params.ResultsPage {
				t.Errorf("expected ResultsPage %d, got %d", scenario.params.ResultsPage, extractedParams.ResultsPage)
			}
			if extractedParams.ResultsPageSize != scenario.params.ResultsPageSize {
				t.Errorf("expected ResultsPageSize %d, got %d", scenario.params.ResultsPageSize, extractedParams.ResultsPageSize)
			}
		})
	}
}
