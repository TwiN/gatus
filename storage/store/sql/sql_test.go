package sql

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

var (
	firstCondition  = endpoint.Condition("[STATUS] == 200")
	secondCondition = endpoint.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = endpoint.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	now = time.Now()

	testEndpoint = endpoint.Endpoint{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []endpoint.Condition{firstCondition, secondCondition, thirdCondition},
		Alerts:                  nil,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuccessfulResult = endpoint.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                nil,
		Connected:             true,
		Success:               true,
		Timestamp:             now,
		Duration:              150 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*endpoint.ConditionResult{
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
	testUnsuccessfulResult = endpoint.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                []string{"error-1", "error-2"},
		Connected:             true,
		Success:               false,
		Timestamp:             now,
		Duration:              750 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*endpoint.ConditionResult{
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
	if _, err := NewStore("", t.TempDir()+"/TestNewStore.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents); !errors.Is(err, ErrDatabaseDriverNotSpecified) {
		t.Error("expected error due to blank driver parameter")
	}
	if _, err := NewStore("sqlite", "", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents); !errors.Is(err, ErrPathNotSpecified) {
		t.Error("expected error due to blank path parameter")
	}
	if store, err := NewStore("sqlite", t.TempDir()+"/TestNewStore.db", true, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents); err != nil {
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

	store.InsertEndpointResult(&testEndpoint, &endpoint.Result{Timestamp: now.Add(-5 * time.Hour), Success: true})

	tx, _ := store.db.Begin()
	oldest, _ := store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 5*time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~5 hours old, was %s", oldest)
	}

	// The oldest cache entry should remain at ~5 hours old, because this entry is more recent
	store.InsertEndpointResult(&testEndpoint, &endpoint.Result{Timestamp: now.Add(-3 * time.Hour), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 5*time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~5 hours old, was %s", oldest)
	}

	// The oldest cache entry should now become at ~8 hours old, because this entry is older
	store.InsertEndpointResult(&testEndpoint, &endpoint.Result{Timestamp: now.Add(-8 * time.Hour), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 8*time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~8 hours old, was %s", oldest)
	}

	// Since this is one hour before reaching the clean up threshold, the oldest entry should now be this one
	store.InsertEndpointResult(&testEndpoint, &endpoint.Result{Timestamp: now.Add(-(uptimeAgeCleanUpThreshold - time.Hour)), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != uptimeAgeCleanUpThreshold-time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~%s hours old, was %s", uptimeAgeCleanUpThreshold-time.Hour, oldest)
	}

	// Since this entry is after the uptimeAgeCleanUpThreshold, both this entry as well as the previous
	// one should be deleted since they both surpass uptimeRetention
	store.InsertEndpointResult(&testEndpoint, &endpoint.Result{Timestamp: now.Add(-(uptimeAgeCleanUpThreshold + time.Hour)), Success: true})

	tx, _ = store.db.Begin()
	oldest, _ = store.getAgeOfOldestEndpointUptimeEntry(tx, 1)
	_ = tx.Commit()
	if oldest.Truncate(time.Hour) != 8*time.Hour {
		t.Errorf("oldest endpoint uptime entry should've been ~8 hours old, was %s", oldest)
	}
}

func TestStore_HourlyUptimeEntriesAreMergedIntoDailyUptimeEntriesProperly(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_HourlyUptimeEntriesAreMergedIntoDailyUptimeEntriesProperly.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	now := time.Now().Truncate(time.Hour)
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	scenarios := []struct {
		numberOfHours            int
		expectedMaxUptimeEntries int64
	}{
		{numberOfHours: 1, expectedMaxUptimeEntries: 1},
		{numberOfHours: 10, expectedMaxUptimeEntries: 10},
		{numberOfHours: 50, expectedMaxUptimeEntries: 50},
		{numberOfHours: 75, expectedMaxUptimeEntries: 75},
		{numberOfHours: 99, expectedMaxUptimeEntries: 99},
		{numberOfHours: 150, expectedMaxUptimeEntries: 100},
		{numberOfHours: 300, expectedMaxUptimeEntries: 100},
		{numberOfHours: 768, expectedMaxUptimeEntries: 100}, // 32 days (in hours), which means anything beyond that won't be persisted anyway
		{numberOfHours: 1000, expectedMaxUptimeEntries: 100},
	}
	// Note that is not technically an accurate real world representation, because uptime entries are always added in
	// the present, while this test is inserting results from the past to simulate long term uptime entries.
	// Since we want to test the behavior and not the test itself, this is a "best effort" approach.
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("num-hours-%d-expected-max-entries-%d", scenario.numberOfHours, scenario.expectedMaxUptimeEntries), func(t *testing.T) {
			for i := scenario.numberOfHours; i > 0; i-- {
				//fmt.Printf("i: %d (%s)\n", i, now.Add(-time.Duration(i)*time.Hour))
				// Create an uptime entry
				err := store.InsertEndpointResult(&testEndpoint, &endpoint.Result{Timestamp: now.Add(-time.Duration(i) * time.Hour), Success: true})
				if err != nil {
					t.Log(err)
				}
				//// DEBUGGING: check number of uptime entries for endpoint
				//tx, _ := store.db.Begin()
				//numberOfUptimeEntriesForEndpoint, err := store.getNumberOfUptimeEntriesByEndpointID(tx, 1)
				//if err != nil {
				//	t.Log(err)
				//}
				//_ = tx.Commit()
				//t.Logf("i=%d; numberOfHours=%d; There are currently %d uptime entries for endpointID=%d", i, scenario.numberOfHours, numberOfUptimeEntriesForEndpoint, 1)
			}
			// check number of uptime entries for endpoint
			tx, _ := store.db.Begin()
			numberOfUptimeEntriesForEndpoint, err := store.getNumberOfUptimeEntriesByEndpointID(tx, 1)
			if err != nil {
				t.Log(err)
			}
			_ = tx.Commit()
			//t.Logf("numberOfHours=%d; There are currently %d uptime entries for endpointID=%d", scenario.numberOfHours, numberOfUptimeEntriesForEndpoint, 1)
			if scenario.expectedMaxUptimeEntries < numberOfUptimeEntriesForEndpoint {
				t.Errorf("expected %d (uptime entries) to be smaller than %d", numberOfUptimeEntriesForEndpoint, scenario.expectedMaxUptimeEntries)
			}
			store.Clear()
		})
	}
}

func TestStore_getEndpointUptime(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InsertCleansUpEventsAndResultsProperly.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Clear()
	defer store.Close()
	// Add 768 hourly entries (32 days)
	// Daily entries should be merged from hourly entries automatically
	for i := 768; i > 0; i-- {
		err := store.InsertEndpointResult(&testEndpoint, &endpoint.Result{Timestamp: time.Now().Add(-time.Duration(i) * time.Hour), Duration: time.Second, Success: true})
		if err != nil {
			t.Log(err)
		}
	}
	// Check the number of uptime entries
	tx, _ := store.db.Begin()
	numberOfUptimeEntriesForEndpoint, err := store.getNumberOfUptimeEntriesByEndpointID(tx, 1)
	if err != nil {
		t.Log(err)
	}
	if numberOfUptimeEntriesForEndpoint < 20 || numberOfUptimeEntriesForEndpoint > 200 {
		t.Errorf("expected number of uptime entries to be between 20 and 200, got %d", numberOfUptimeEntriesForEndpoint)
	}
	// Retrieve uptime for the past 30d
	uptime, avgResponseTime, err := store.getEndpointUptime(tx, 1, time.Now().Add(-(30 * 24 * time.Hour)), time.Now())
	if err != nil {
		t.Log(err)
	}
	_ = tx.Commit()
	if avgResponseTime != time.Second {
		t.Errorf("expected average response time to be %s, got %s", time.Second, avgResponseTime)
	}
	if uptime != 1 {
		t.Errorf("expected uptime to be 1, got %f", uptime)
	}
	// Add a new unsuccessful result, which should impact the uptime
	err = store.InsertEndpointResult(&testEndpoint, &endpoint.Result{Timestamp: time.Now(), Duration: time.Second, Success: false})
	if err != nil {
		t.Log(err)
	}
	// Retrieve uptime for the past 30d
	tx, _ = store.db.Begin()
	uptime, _, err = store.getEndpointUptime(tx, 1, time.Now().Add(-(30 * 24 * time.Hour)), time.Now())
	if err != nil {
		t.Log(err)
	}
	_ = tx.Commit()
	if uptime == 1 {
		t.Errorf("expected uptime to be less than 1, got %f", uptime)
	}
	// Retrieve uptime for the past 30d, but excluding the last 24h
	// This is not a real use case as there is no way for users to exclude the last 24h, but this is a great way
	// to ensure that hourly merging works as intended
	tx, _ = store.db.Begin()
	uptimeExcludingLast24h, _, err := store.getEndpointUptime(tx, 1, time.Now().Add(-(30 * 24 * time.Hour)), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Log(err)
	}
	_ = tx.Commit()
	if uptimeExcludingLast24h == uptime {
		t.Error("expected uptimeExcludingLast24h to to be different from uptime, got")
	}
}

func TestStore_InsertCleansUpEventsAndResultsProperly(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InsertCleansUpEventsAndResultsProperly.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Clear()
	defer store.Close()
	resultsCleanUpThreshold := store.maximumNumberOfResults + resultsAboveMaximumCleanUpThreshold
	eventsCleanUpThreshold := store.maximumNumberOfEvents + eventsAboveMaximumCleanUpThreshold
	for i := 0; i < resultsCleanUpThreshold+eventsCleanUpThreshold; i++ {
		store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
		store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult)
		ss, _ := store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithResults(1, storage.DefaultMaximumNumberOfResults*5).WithEvents(1, storage.DefaultMaximumNumberOfEvents*5))
		if len(ss.Results) > resultsCleanUpThreshold+1 {
			t.Errorf("number of results shouldn't have exceeded %d, reached %d", resultsCleanUpThreshold, len(ss.Results))
		}
		if len(ss.Events) > eventsCleanUpThreshold+1 {
			t.Errorf("number of events shouldn't have exceeded %d, reached %d", eventsCleanUpThreshold, len(ss.Events))
		}
	}
}

func TestStore_InsertWithCaching(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_InsertWithCaching.db", true, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	// Add 2 results
	store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
	store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
	// Verify that they exist
	endpointStatuses, _ := store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20))
	if numberOfEndpointStatuses := len(endpointStatuses); numberOfEndpointStatuses != 1 {
		t.Fatalf("expected 1 EndpointStatus, got %d", numberOfEndpointStatuses)
	}
	if len(endpointStatuses[0].Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(endpointStatuses[0].Results))
	}
	// Add 2 more results
	store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult)
	store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult)
	// Verify that they exist
	endpointStatuses, _ = store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20))
	if numberOfEndpointStatuses := len(endpointStatuses); numberOfEndpointStatuses != 1 {
		t.Fatalf("expected 1 EndpointStatus, got %d", numberOfEndpointStatuses)
	}
	if len(endpointStatuses[0].Results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(endpointStatuses[0].Results))
	}
	// Clear the store, which should also clear the cache
	store.Clear()
	// Verify that they no longer exist
	endpointStatuses, _ = store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20))
	if numberOfEndpointStatuses := len(endpointStatuses); numberOfEndpointStatuses != 0 {
		t.Fatalf("expected 0 EndpointStatus, got %d", numberOfEndpointStatuses)
	}
}

func TestStore_Persistence(t *testing.T) {
	path := t.TempDir() + "/TestStore_Persistence.db"
	store, _ := NewStore("sqlite", path, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
	store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult)
	if uptime, _ := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour), time.Now()); uptime != 0.5 {
		t.Errorf("the uptime over the past 1h should've been 0.5, got %f", uptime)
	}
	if uptime, _ := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour*24), time.Now()); uptime != 0.5 {
		t.Errorf("the uptime over the past 24h should've been 0.5, got %f", uptime)
	}
	if uptime, _ := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour*24*7), time.Now()); uptime != 0.5 {
		t.Errorf("the uptime over the past 7d should've been 0.5, got %f", uptime)
	}
	if uptime, _ := store.GetUptimeByKey(testEndpoint.Key(), time.Now().Add(-time.Hour*24*30), time.Now()); uptime != 0.5 {
		t.Errorf("the uptime over the past 30d should've been 0.5, got %f", uptime)
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
	store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
	endpointStatuses, _ := store.GetAllEndpointStatuses(paging.NewEndpointStatusParams())
	if numberOfEndpointStatuses := len(endpointStatuses); numberOfEndpointStatuses != 1 {
		t.Fatalf("expected 1 EndpointStatus, got %d", numberOfEndpointStatuses)
	}
	store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult)
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
	if err := store.insertEndpointEvent(tx, 1, endpoint.NewEventFromResult(&testSuccessfulResult)); err == nil {
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
	if _, err := store.getLastEndpointResultSuccessValue(tx, 1); !errors.Is(err, errNoRowsReturned) {
		t.Errorf("should've %v, got %v", errNoRowsReturned, err)
	}
	if _, err := store.getAgeOfOldestEndpointUptimeEntry(tx, 1); !errors.Is(err, errNoRowsReturned) {
		t.Errorf("should've %v, got %v", errNoRowsReturned, err)
	}
}

// This tests very unlikely cases where a table is deleted.
func TestStore_BrokenSchema(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_BrokenSchema.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
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
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err == nil {
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
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoint_events")
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
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
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoint_results")
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 1).WithEvents(1, 1)); err == nil {
		t.Fatal("expected an error")
	}
	// Repair
	if err := store.createSchema(); err != nil {
		t.Fatal("schema should've been repaired")
	}
	store.Clear()
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoint_result_conditions")
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err == nil {
		t.Fatal("expected an error")
	}
	// Repair
	if err := store.createSchema(); err != nil {
		t.Fatal("schema should've been repaired")
	}
	store.Clear()
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Break
	_, _ = store.db.Exec("DROP TABLE endpoint_uptimes")
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
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

func TestTriggeredEndpointAlertsPersistence(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestTriggeredEndpointAlertsPersistence.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	yes, desc := false, "description"
	ep := testEndpoint
	ep.NumberOfSuccessesInARow = 0
	alrt := &alert.Alert{
		Type:             alert.TypePagerDuty,
		Enabled:          &yes,
		FailureThreshold: 4,
		SuccessThreshold: 2,
		Description:      &desc,
		SendOnResolved:   &yes,
		Triggered:        true,
		ResolveKey:       "1234567",
	}
	// Alert just triggered, so NumberOfSuccessesInARow is 0
	if err := store.UpsertTriggeredEndpointAlert(&ep, alrt); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	exists, resolveKey, numberOfSuccessesInARow, err := store.GetTriggeredEndpointAlert(&ep, alrt)
	if err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if !exists {
		t.Error("expected triggered alert to exist")
	}
	if resolveKey != alrt.ResolveKey {
		t.Errorf("expected resolveKey %s, got %s", alrt.ResolveKey, resolveKey)
	}
	if numberOfSuccessesInARow != ep.NumberOfSuccessesInARow {
		t.Errorf("expected persisted NumberOfSuccessesInARow to be %d, got %d", ep.NumberOfSuccessesInARow, numberOfSuccessesInARow)
	}
	// Endpoint just had a successful evaluation, so NumberOfSuccessesInARow is now 1
	ep.NumberOfSuccessesInARow++
	if err := store.UpsertTriggeredEndpointAlert(&ep, alrt); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	exists, resolveKey, numberOfSuccessesInARow, err = store.GetTriggeredEndpointAlert(&ep, alrt)
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if !exists {
		t.Error("expected triggered alert to exist")
	}
	if resolveKey != alrt.ResolveKey {
		t.Errorf("expected resolveKey %s, got %s", alrt.ResolveKey, resolveKey)
	}
	if numberOfSuccessesInARow != ep.NumberOfSuccessesInARow {
		t.Errorf("expected persisted NumberOfSuccessesInARow to be %d, got %d", ep.NumberOfSuccessesInARow, numberOfSuccessesInARow)
	}
	// Simulate the endpoint having another successful evaluation, which means the alert is now resolved,
	// and we should delete the triggered alert from the store
	ep.NumberOfSuccessesInARow++
	if err := store.DeleteTriggeredEndpointAlert(&ep, alrt); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	exists, _, _, err = store.GetTriggeredEndpointAlert(&ep, alrt)
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if exists {
		t.Error("expected triggered alert to no longer exist as it has been deleted")
	}
}

func TestStore_DeleteAllTriggeredAlertsNotInChecksumsByEndpoint(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_DeleteAllTriggeredAlertsNotInChecksumsByEndpoint.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	yes, desc := false, "description"
	ep1 := testEndpoint
	ep1.Name = "ep1"
	ep2 := testEndpoint
	ep2.Name = "ep2"
	alert1 := alert.Alert{
		Type:             alert.TypePagerDuty,
		Enabled:          &yes,
		FailureThreshold: 4,
		SuccessThreshold: 2,
		Description:      &desc,
		SendOnResolved:   &yes,
		Triggered:        true,
		ResolveKey:       "1234567",
	}
	alert2 := alert1
	alert2.Type, alert2.ResolveKey = alert.TypeSlack, ""
	alert3 := alert2
	if err := store.UpsertTriggeredEndpointAlert(&ep1, &alert1); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if err := store.UpsertTriggeredEndpointAlert(&ep1, &alert2); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if err := store.UpsertTriggeredEndpointAlert(&ep2, &alert3); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if exists, _, _, _ := store.GetTriggeredEndpointAlert(&ep1, &alert1); !exists {
		t.Error("expected alert1 to have been deleted")
	}
	if exists, _, _, _ := store.GetTriggeredEndpointAlert(&ep1, &alert2); !exists {
		t.Error("expected alert2 to exist for ep1")
	}
	if exists, _, _, _ := store.GetTriggeredEndpointAlert(&ep2, &alert3); !exists {
		t.Error("expected alert3 to exist for ep2")
	}
	// Now we simulate the alert configuration being updated, and the alert being resolved
	if deleted := store.DeleteAllTriggeredAlertsNotInChecksumsByEndpoint(&ep1, []string{alert2.Checksum()}); deleted != 1 {
		t.Errorf("expected 1 triggered alert to be deleted, got %d", deleted)
	}
	if exists, _, _, _ := store.GetTriggeredEndpointAlert(&ep1, &alert1); exists {
		t.Error("expected alert1 to have been deleted")
	}
	if exists, _, _, _ := store.GetTriggeredEndpointAlert(&ep1, &alert2); !exists {
		t.Error("expected alert2 to exist for ep1")
	}
	if exists, _, _, _ := store.GetTriggeredEndpointAlert(&ep2, &alert3); !exists {
		t.Error("expected alert3 to exist for ep2")
	}
	// Now let's just assume all alerts for ep1 were removed
	if deleted := store.DeleteAllTriggeredAlertsNotInChecksumsByEndpoint(&ep1, []string{}); deleted != 1 {
		t.Errorf("expected 1 triggered alert to be deleted, got %d", deleted)
	}
	// Make sure the alert for ep2 still exists
	if exists, _, _, _ := store.GetTriggeredEndpointAlert(&ep2, &alert3); !exists {
		t.Error("expected alert3 to exist for ep2")
	}
}

func TestStore_HasEndpointStatusNewerThan(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_HasEndpointStatusNewerThan.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	// InsertEndpointResult an endpoint status
	if err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	// Check if it has a status newer than 1 hour ago
	hasNewerStatus, err := store.HasEndpointStatusNewerThan(testEndpoint.Key(), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if !hasNewerStatus {
		t.Error("expected to have a newer status")
	}
	// Check if it has a status newer than 2 days ago
	hasNewerStatus, err = store.HasEndpointStatusNewerThan(testEndpoint.Key(), time.Now().Add(-48*time.Hour))
	if err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if !hasNewerStatus {
		t.Error("expected to have a newer status")
	}
	// Check if there's a status newer than 1 hour in the future (silly test, but it should work)
	hasNewerStatus, err = store.HasEndpointStatusNewerThan(testEndpoint.Key(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if hasNewerStatus {
		t.Error("expected not to have a newer status in the future")
	}
}

// TestEventOrderingFix specifically tests the SQL ordering fix for issue #1040
// This test verifies that getEndpointEventsByEndpointID returns the most recent events
// in chronological order (oldest to newest)
func TestEventOrderingFix(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/test.db", false, 100, 100)
	defer store.Close()
	ep := &endpoint.Endpoint{
		Name:  "ordering-test",
		Group: "test",
		URL:   "https://example.com",
	}
	// Create many events over time
	baseTime := time.Now().Add(-100 * time.Hour) // Start 100 hours ago
	for i := 0; i < 50; i++ {
		result := &endpoint.Result{
			Success:   i%2 == 0, // Alternate between true/false to create events
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
		}
		err := store.InsertEndpointResult(ep, result)
		if err != nil {
			t.Fatalf("Failed to insert result %d: %v", i, err)
		}
	}
	// Now retrieve events with pagination to test the ordering
	tx, _ := store.db.Begin()
	endpointID, _, _, _ := store.getEndpointIDGroupAndNameByKey(tx, ep.Key())
	// Get the first page (should get the MOST RECENT events, but in chronological order)
	events, err := store.getEndpointEventsByEndpointID(tx, endpointID, 1, 10)
	tx.Commit()
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}
	if len(events) != 10 {
		t.Errorf("Expected 10 events, got %d", len(events))
	}
	// Verify the events are in chronological order (oldest to newest)
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Before(events[i-1].Timestamp) {
			t.Errorf("Events not in chronological order: event %d timestamp %v is before event %d timestamp %v",
				i, events[i].Timestamp, i-1, events[i-1].Timestamp)
		}
	}
	// Verify these are the most recent events
	// The last event in the returned list should be close to "now" (within the last few events we created)
	lastEventTime := events[len(events)-1].Timestamp
	expectedRecentTime := baseTime.Add(49 * time.Hour) // The most recent event we created
	timeDiff := expectedRecentTime.Sub(lastEventTime)
	if timeDiff > 10*time.Hour { // Allow some margin for events
		t.Errorf("Events are not the most recent ones. Last event time: %v, expected around: %v (diff: %v)",
			lastEventTime, expectedRecentTime, timeDiff)
	}
	t.Logf("Successfully retrieved %d most recent events in chronological order", len(events))
	t.Logf("First event: %s at %v", events[0].Type, events[0].Timestamp)
	t.Logf("Last event: %s at %v", events[len(events)-1].Type, events[len(events)-1].Timestamp)
}
