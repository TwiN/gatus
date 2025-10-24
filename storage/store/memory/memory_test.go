package memory

import (
	"sync"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/suite"
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

// Note that are much more extensive tests in /storage/store/store_test.go.
// This test is simply an extra sanity check
func TestStore_SanityCheck(t *testing.T) {
	store, _ := NewStore(storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Clear()
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
	if deleted := store.DeleteAllEndpointStatusesNotInKeys([]string{}); deleted != 1 {
		t.Errorf("%d entries should've been deleted, got %d", 1, deleted)
	}
}

func TestStore_Save(t *testing.T) {
	store, err := NewStore(storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	err = store.Save()
	if err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	store.Clear()
	store.Close()
}

func TestStore_HasEndpointStatusNewerThan(t *testing.T) {
	store, _ := NewStore(storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Clear()
	defer store.Close()
	// InsertEndpointResult a result
	err := store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
	if err != nil {
		t.Fatalf("expected no error while inserting result, got %v", err)
	}
	// Check with a timestamp in the past
	hasNewerStatus, err := store.HasEndpointStatusNewerThan(testEndpoint.Key(), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hasNewerStatus {
		t.Fatal("expected to have a newer status, but didn't")
	}
	// Check with a timestamp in the future
	hasNewerStatus, err = store.HasEndpointStatusNewerThan(testEndpoint.Key(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hasNewerStatus {
		t.Fatal("expected not to have a newer status, but did")
	}
}

// TestStore_MixedEndpointsAndSuites tests that having both endpoints and suites in the cache
// doesn't cause issues with core operations
func TestStore_MixedEndpointsAndSuites(t *testing.T) {
	// Helper function to create and populate a store with test data
	setupStore := func(t *testing.T) (*Store, *endpoint.Endpoint, *endpoint.Endpoint, *endpoint.Endpoint, *endpoint.Endpoint, *suite.Suite) {
		store, err := NewStore(100, 50)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}

		// Create regular endpoints
		endpoint1 := &endpoint.Endpoint{
			Name:  "endpoint1",
			Group: "group1",
			URL:   "https://example.com/1",
		}
		endpoint2 := &endpoint.Endpoint{
			Name:  "endpoint2",
			Group: "group2",
			URL:   "https://example.com/2",
		}

		// Create suite endpoints (these would be part of a suite)
		suiteEndpoint1 := &endpoint.Endpoint{
			Name:  "suite-endpoint1",
			Group: "suite-group",
			URL:   "https://example.com/suite1",
		}
		suiteEndpoint2 := &endpoint.Endpoint{
			Name:  "suite-endpoint2",
			Group: "suite-group",
			URL:   "https://example.com/suite2",
		}

		// Create a suite
		testSuite := &suite.Suite{
			Name:  "test-suite",
			Group: "suite-group",
			Endpoints: []*endpoint.Endpoint{
				suiteEndpoint1,
				suiteEndpoint2,
			},
		}

		return store, endpoint1, endpoint2, suiteEndpoint1, suiteEndpoint2, testSuite
	}

	// Test 1: InsertEndpointResult endpoint results
	t.Run("InsertEndpointResults", func(t *testing.T) {
		store, endpoint1, endpoint2, suiteEndpoint1, suiteEndpoint2, _ := setupStore(t)
		// InsertEndpointResult regular endpoint results
		result1 := &endpoint.Result{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  100 * time.Millisecond,
		}
		if err := store.InsertEndpointResult(endpoint1, result1); err != nil {
			t.Fatalf("failed to insert endpoint1 result: %v", err)
		}

		result2 := &endpoint.Result{
			Success:   false,
			Timestamp: time.Now(),
			Duration:  200 * time.Millisecond,
			Errors:    []string{"error"},
		}
		if err := store.InsertEndpointResult(endpoint2, result2); err != nil {
			t.Fatalf("failed to insert endpoint2 result: %v", err)
		}

		// InsertEndpointResult suite endpoint results
		suiteResult1 := &endpoint.Result{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  50 * time.Millisecond,
		}
		if err := store.InsertEndpointResult(suiteEndpoint1, suiteResult1); err != nil {
			t.Fatalf("failed to insert suite endpoint1 result: %v", err)
		}

		suiteResult2 := &endpoint.Result{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  75 * time.Millisecond,
		}
		if err := store.InsertEndpointResult(suiteEndpoint2, suiteResult2); err != nil {
			t.Fatalf("failed to insert suite endpoint2 result: %v", err)
		}
	})

	// Test 2: InsertEndpointResult suite result
	t.Run("InsertSuiteResult", func(t *testing.T) {
		store, _, _, _, _, testSuite := setupStore(t)
		timestamp := time.Now()
		suiteResult := &suite.Result{
			Name:      testSuite.Name,
			Group:     testSuite.Group,
			Success:   true,
			Timestamp: timestamp,
			Duration:  125 * time.Millisecond,
			EndpointResults: []*endpoint.Result{
				{Success: true, Duration: 50 * time.Millisecond},
				{Success: true, Duration: 75 * time.Millisecond},
			},
		}
		if err := store.InsertSuiteResult(testSuite, suiteResult); err != nil {
			t.Fatalf("failed to insert suite result: %v", err)
		}

		// Verify the suite result was stored correctly
		status, err := store.GetSuiteStatusByKey(testSuite.Key(), nil)
		if err != nil {
			t.Fatalf("failed to get suite status: %v", err)
		}
		if len(status.Results) != 1 {
			t.Errorf("expected 1 suite result, got %d", len(status.Results))
		}

		stored := status.Results[0]
		if stored.Name != testSuite.Name {
			t.Errorf("expected result name %s, got %s", testSuite.Name, stored.Name)
		}
		if stored.Group != testSuite.Group {
			t.Errorf("expected result group %s, got %s", testSuite.Group, stored.Group)
		}
		if !stored.Success {
			t.Error("expected result to be successful")
		}
		if stored.Duration != 125*time.Millisecond {
			t.Errorf("expected duration 125ms, got %v", stored.Duration)
		}
		if len(stored.EndpointResults) != 2 {
			t.Errorf("expected 2 endpoint results, got %d", len(stored.EndpointResults))
		}
	})

	// Test 3: GetAllEndpointStatuses should only return endpoints, not suites
	t.Run("GetAllEndpointStatuses", func(t *testing.T) {
		store, endpoint1, endpoint2, _, _, testSuite := setupStore(t)

		// Insert standalone endpoint results only
		store.InsertEndpointResult(endpoint1, &endpoint.Result{Success: true, Timestamp: time.Now(), Duration: 100 * time.Millisecond})
		store.InsertEndpointResult(endpoint2, &endpoint.Result{Success: false, Timestamp: time.Now(), Duration: 200 * time.Millisecond})
		// Suite endpoints should only exist as part of suite results, not as individual endpoint results
		store.InsertSuiteResult(testSuite, &suite.Result{
			Name: testSuite.Name, Group: testSuite.Group, Success: true,
			Timestamp: time.Now(), Duration: 125 * time.Millisecond,
			EndpointResults: []*endpoint.Result{
				{Success: true, Duration: 50 * time.Millisecond, Name: "suite-endpoint1"},
				{Success: true, Duration: 75 * time.Millisecond, Name: "suite-endpoint2"},
			},
		})
		statuses, err := store.GetAllEndpointStatuses(&paging.EndpointStatusParams{})
		if err != nil {
			t.Fatalf("failed to get all endpoint statuses: %v", err)
		}

		// Should have 2 endpoints (only standalone endpoints, not suite endpoints)
		if len(statuses) != 2 {
			t.Errorf("expected 2 endpoint statuses, got %d", len(statuses))
		}

		// Verify all are standalone endpoint statuses with correct data, not suite endpoints
		expectedEndpoints := map[string]struct {
			success  bool
			duration time.Duration
		}{
			"endpoint1": {success: true, duration: 100 * time.Millisecond},
			"endpoint2": {success: false, duration: 200 * time.Millisecond},
		}

		for _, status := range statuses {
			if status.Name == "" {
				t.Error("endpoint status should have a name")
			}
			// Make sure none of them are the suite itself
			if status.Name == "test-suite" {
				t.Error("suite should not appear in endpoint statuses")
			}

			// Verify detailed endpoint data
			expected, exists := expectedEndpoints[status.Name]
			if !exists {
				t.Errorf("unexpected endpoint name: %s", status.Name)
				continue
			}

			// Check that endpoint has results and verify the data
			if len(status.Results) != 1 {
				t.Errorf("endpoint %s should have 1 result, got %d", status.Name, len(status.Results))
				continue
			}

			result := status.Results[0]
			if result.Success != expected.success {
				t.Errorf("endpoint %s result success should be %v, got %v", status.Name, expected.success, result.Success)
			}
			if result.Duration != expected.duration {
				t.Errorf("endpoint %s result duration should be %v, got %v", status.Name, expected.duration, result.Duration)
			}

			delete(expectedEndpoints, status.Name)
		}
		if len(expectedEndpoints) > 0 {
			t.Errorf("missing expected endpoints: %v", expectedEndpoints)
		}
	})

	// Test 4: GetAllSuiteStatuses should only return suites, not endpoints
	t.Run("GetAllSuiteStatuses", func(t *testing.T) {
		store, endpoint1, _, _, _, testSuite := setupStore(t)

		// InsertEndpointResult test data
		store.InsertEndpointResult(endpoint1, &endpoint.Result{Success: true, Timestamp: time.Now(), Duration: 100 * time.Millisecond})
		timestamp := time.Now()
		store.InsertSuiteResult(testSuite, &suite.Result{
			Name: testSuite.Name, Group: testSuite.Group, Success: true,
			Timestamp: timestamp, Duration: 125 * time.Millisecond,
		})
		statuses, err := store.GetAllSuiteStatuses(&paging.SuiteStatusParams{})
		if err != nil {
			t.Fatalf("failed to get all suite statuses: %v", err)
		}

		// Should have 1 suite
		if len(statuses) != 1 {
			t.Errorf("expected 1 suite status, got %d", len(statuses))
		}

		if len(statuses) > 0 {
			suiteStatus := statuses[0]
			if suiteStatus.Name != "test-suite" {
				t.Errorf("expected suite name 'test-suite', got '%s'", suiteStatus.Name)
			}
			if suiteStatus.Group != "suite-group" {
				t.Errorf("expected suite group 'suite-group', got '%s'", suiteStatus.Group)
			}
			if len(suiteStatus.Results) != 1 {
				t.Errorf("expected 1 suite result, got %d", len(suiteStatus.Results))
			}
			if len(suiteStatus.Results) > 0 {
				result := suiteStatus.Results[0]
				if !result.Success {
					t.Error("expected suite result to be successful")
				}
				if result.Duration != 125*time.Millisecond {
					t.Errorf("expected suite result duration 125ms, got %v", result.Duration)
				}
			}
		}
	})

	// Test 5: GetEndpointStatusByKey should work for all endpoints
	t.Run("GetEndpointStatusByKey", func(t *testing.T) {
		store, endpoint1, _, suiteEndpoint1, _, _ := setupStore(t)

		// InsertEndpointResult test data with specific timestamps and durations
		timestamp1 := time.Now()
		timestamp2 := time.Now().Add(1 * time.Hour)
		store.InsertEndpointResult(endpoint1, &endpoint.Result{Success: true, Timestamp: timestamp1, Duration: 100 * time.Millisecond})
		store.InsertEndpointResult(suiteEndpoint1, &endpoint.Result{Success: false, Timestamp: timestamp2, Duration: 50 * time.Millisecond, Errors: []string{"suite error"}})

		// Test regular endpoints
		status1, err := store.GetEndpointStatusByKey(endpoint1.Key(), &paging.EndpointStatusParams{})
		if err != nil {
			t.Fatalf("failed to get endpoint1 status: %v", err)
		}
		if status1.Name != "endpoint1" {
			t.Errorf("expected endpoint1, got %s", status1.Name)
		}
		if status1.Group != "group1" {
			t.Errorf("expected group1, got %s", status1.Group)
		}
		if len(status1.Results) != 1 {
			t.Errorf("expected 1 result for endpoint1, got %d", len(status1.Results))
		}
		if len(status1.Results) > 0 {
			result := status1.Results[0]
			if !result.Success {
				t.Error("expected endpoint1 result to be successful")
			}
			if result.Duration != 100*time.Millisecond {
				t.Errorf("expected endpoint1 result duration 100ms, got %v", result.Duration)
			}
		}

		// Test suite endpoints
		suiteStatus1, err := store.GetEndpointStatusByKey(suiteEndpoint1.Key(), &paging.EndpointStatusParams{})
		if err != nil {
			t.Fatalf("failed to get suite endpoint1 status: %v", err)
		}
		if suiteStatus1.Name != "suite-endpoint1" {
			t.Errorf("expected suite-endpoint1, got %s", suiteStatus1.Name)
		}
		if suiteStatus1.Group != "suite-group" {
			t.Errorf("expected suite-group, got %s", suiteStatus1.Group)
		}
		if len(suiteStatus1.Results) != 1 {
			t.Errorf("expected 1 result for suite-endpoint1, got %d", len(suiteStatus1.Results))
		}
		if len(suiteStatus1.Results) > 0 {
			result := suiteStatus1.Results[0]
			if result.Success {
				t.Error("expected suite-endpoint1 result to be unsuccessful")
			}
			if result.Duration != 50*time.Millisecond {
				t.Errorf("expected suite-endpoint1 result duration 50ms, got %v", result.Duration)
			}
			if len(result.Errors) != 1 || result.Errors[0] != "suite error" {
				t.Errorf("expected suite-endpoint1 to have error 'suite error', got %v", result.Errors)
			}
		}
	})

	// Test 6: GetSuiteStatusByKey should work for suites
	t.Run("GetSuiteStatusByKey", func(t *testing.T) {
		store, _, _, _, _, testSuite := setupStore(t)

		// InsertEndpointResult suite result with endpoint results
		timestamp := time.Now()
		store.InsertSuiteResult(testSuite, &suite.Result{
			Name: testSuite.Name, Group: testSuite.Group, Success: false,
			Timestamp: timestamp, Duration: 125 * time.Millisecond,
			EndpointResults: []*endpoint.Result{
				{Success: true, Duration: 50 * time.Millisecond},
				{Success: false, Duration: 75 * time.Millisecond, Errors: []string{"endpoint failed"}},
			},
		})
		suiteStatus, err := store.GetSuiteStatusByKey(testSuite.Key(), &paging.SuiteStatusParams{})
		if err != nil {
			t.Fatalf("failed to get suite status: %v", err)
		}
		if suiteStatus.Name != "test-suite" {
			t.Errorf("expected test-suite, got %s", suiteStatus.Name)
		}
		if suiteStatus.Group != "suite-group" {
			t.Errorf("expected suite-group, got %s", suiteStatus.Group)
		}
		if len(suiteStatus.Results) != 1 {
			t.Errorf("expected 1 suite result, got %d", len(suiteStatus.Results))
		}

		if len(suiteStatus.Results) > 0 {
			result := suiteStatus.Results[0]
			if result.Success {
				t.Error("expected suite result to be unsuccessful")
			}
			if result.Duration != 125*time.Millisecond {
				t.Errorf("expected suite result duration 125ms, got %v", result.Duration)
			}
			if len(result.EndpointResults) != 2 {
				t.Errorf("expected 2 endpoint results, got %d", len(result.EndpointResults))
			}
			if len(result.EndpointResults) >= 2 {
				if !result.EndpointResults[0].Success {
					t.Error("expected first endpoint result to be successful")
				}
				if result.EndpointResults[1].Success {
					t.Error("expected second endpoint result to be unsuccessful")
				}
				if len(result.EndpointResults[1].Errors) != 1 || result.EndpointResults[1].Errors[0] != "endpoint failed" {
					t.Errorf("expected second endpoint to have error 'endpoint failed', got %v", result.EndpointResults[1].Errors)
				}
			}
		}
	})

	// Test 7: DeleteAllEndpointStatusesNotInKeys should not affect suites
	t.Run("DeleteEndpointsNotInKeys", func(t *testing.T) {
		store, endpoint1, endpoint2, suiteEndpoint1, suiteEndpoint2, testSuite := setupStore(t)

		// InsertEndpointResult all test data
		store.InsertEndpointResult(endpoint1, &endpoint.Result{Success: true, Timestamp: time.Now(), Duration: 100 * time.Millisecond})
		store.InsertEndpointResult(endpoint2, &endpoint.Result{Success: false, Timestamp: time.Now(), Duration: 200 * time.Millisecond})
		store.InsertEndpointResult(suiteEndpoint1, &endpoint.Result{Success: true, Timestamp: time.Now(), Duration: 50 * time.Millisecond})
		store.InsertEndpointResult(suiteEndpoint2, &endpoint.Result{Success: true, Timestamp: time.Now(), Duration: 75 * time.Millisecond})
		store.InsertSuiteResult(testSuite, &suite.Result{
			Name: testSuite.Name, Group: testSuite.Group, Success: true,
			Timestamp: time.Now(), Duration: 125 * time.Millisecond,
		})
		// Keep only endpoint1 and suite-endpoint1
		keysToKeep := []string{endpoint1.Key(), suiteEndpoint1.Key()}
		deleted := store.DeleteAllEndpointStatusesNotInKeys(keysToKeep)

		// Should have deleted 2 endpoints (endpoint2 and suite-endpoint2)
		if deleted != 2 {
			t.Errorf("expected to delete 2 endpoints, deleted %d", deleted)
		}

		// Verify remaining endpoints
		statuses, _ := store.GetAllEndpointStatuses(&paging.EndpointStatusParams{})
		if len(statuses) != 2 {
			t.Errorf("expected 2 remaining endpoint statuses, got %d", len(statuses))
		}

		// Suite should still exist
		suiteStatuses, _ := store.GetAllSuiteStatuses(&paging.SuiteStatusParams{})
		if len(suiteStatuses) != 1 {
			t.Errorf("suite should not be affected by DeleteAllEndpointStatusesNotInKeys")
		}
	})

	// Test 8: DeleteAllSuiteStatusesNotInKeys should not affect endpoints
	t.Run("DeleteSuitesNotInKeys", func(t *testing.T) {
		store, endpoint1, _, _, _, testSuite := setupStore(t)

		// InsertEndpointResult test data
		store.InsertEndpointResult(endpoint1, &endpoint.Result{Success: true, Timestamp: time.Now(), Duration: 100 * time.Millisecond})
		store.InsertSuiteResult(testSuite, &suite.Result{
			Name: testSuite.Name, Group: testSuite.Group, Success: true,
			Timestamp: time.Now(), Duration: 125 * time.Millisecond,
		})
		// First, add another suite to test deletion
		anotherSuite := &suite.Suite{
			Name:  "another-suite",
			Group: "another-group",
		}
		anotherSuiteResult := &suite.Result{
			Name:      anotherSuite.Name,
			Group:     anotherSuite.Group,
			Success:   true,
			Timestamp: time.Now(),
			Duration:  100 * time.Millisecond,
		}
		store.InsertSuiteResult(anotherSuite, anotherSuiteResult)

		// Keep only the original test-suite
		deleted := store.DeleteAllSuiteStatusesNotInKeys([]string{testSuite.Key()})

		// Should have deleted 1 suite (another-suite)
		if deleted != 1 {
			t.Errorf("expected to delete 1 suite, deleted %d", deleted)
		}

		// Endpoints should still exist
		endpointStatuses, _ := store.GetAllEndpointStatuses(&paging.EndpointStatusParams{})
		if len(endpointStatuses) != 1 {
			t.Errorf("endpoints should not be affected by DeleteAllSuiteStatusesNotInKeys")
		}

		// Only one suite should remain
		suiteStatuses, _ := store.GetAllSuiteStatuses(&paging.SuiteStatusParams{})
		if len(suiteStatuses) != 1 {
			t.Errorf("expected 1 remaining suite, got %d", len(suiteStatuses))
		}
	})

	// Test 9: Clear should remove everything
	t.Run("Clear", func(t *testing.T) {
		store, endpoint1, _, _, _, testSuite := setupStore(t)

		// InsertEndpointResult test data
		store.InsertEndpointResult(endpoint1, &endpoint.Result{Success: true, Timestamp: time.Now(), Duration: 100 * time.Millisecond})
		store.InsertSuiteResult(testSuite, &suite.Result{
			Name: testSuite.Name, Group: testSuite.Group, Success: true,
			Timestamp: time.Now(), Duration: 125 * time.Millisecond,
		})
		store.Clear()

		// No endpoints should remain
		endpointStatuses, _ := store.GetAllEndpointStatuses(&paging.EndpointStatusParams{})
		if len(endpointStatuses) != 0 {
			t.Errorf("expected 0 endpoints after clear, got %d", len(endpointStatuses))
		}

		// No suites should remain
		suiteStatuses, _ := store.GetAllSuiteStatuses(&paging.SuiteStatusParams{})
		if len(suiteStatuses) != 0 {
			t.Errorf("expected 0 suites after clear, got %d", len(suiteStatuses))
		}
	})
}

// TestStore_EndpointStatusCastingSafety tests that type assertions are safe
func TestStore_EndpointStatusCastingSafety(t *testing.T) {
	store, err := NewStore(100, 50)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}

	// InsertEndpointResult an endpoint
	ep := &endpoint.Endpoint{
		Name:  "test-endpoint",
		Group: "test",
		URL:   "https://example.com",
	}
	result := &endpoint.Result{
		Success:   true,
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
	}
	store.InsertEndpointResult(ep, result)

	// InsertEndpointResult a suite
	testSuite := &suite.Suite{
		Name:  "test-suite",
		Group: "test",
	}
	suiteResult := &suite.Result{
		Name:      testSuite.Name,
		Group:     testSuite.Group,
		Success:   true,
		Timestamp: time.Now(),
		Duration:  200 * time.Millisecond,
	}
	store.InsertSuiteResult(testSuite, suiteResult)

	// This should not panic even with mixed types in cache
	statuses, err := store.GetAllEndpointStatuses(&paging.EndpointStatusParams{})
	if err != nil {
		t.Fatalf("failed to get all endpoint statuses: %v", err)
	}

	// Should only have the endpoint, not the suite
	if len(statuses) != 1 {
		t.Errorf("expected 1 endpoint status, got %d", len(statuses))
	}
	if statuses[0].Name != "test-endpoint" {
		t.Errorf("expected test-endpoint, got %s", statuses[0].Name)
	}
}

func TestStore_MaximumLimits(t *testing.T) {
	// Use small limits to test trimming behavior
	maxResults := 5
	maxEvents := 3
	store, err := NewStore(maxResults, maxEvents)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	defer store.Clear()

	t.Run("endpoint-result-limits", func(t *testing.T) {
		ep := &endpoint.Endpoint{Name: "test-endpoint", Group: "test", URL: "https://example.com"}

		// Insert more results than the maximum
		baseTime := time.Now().Add(-10 * time.Hour)
		for i := 0; i < maxResults*2; i++ {
			result := &endpoint.Result{
				Success:   i%2 == 0,
				Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
				Duration:  time.Duration(i*10) * time.Millisecond,
			}
			err := store.InsertEndpointResult(ep, result)
			if err != nil {
				t.Fatalf("failed to insert result %d: %v", i, err)
			}
		}

		// Verify only maxResults are kept
		status, err := store.GetEndpointStatusByKey(ep.Key(), nil)
		if err != nil {
			t.Fatalf("failed to get endpoint status: %v", err)
		}
		if len(status.Results) != maxResults {
			t.Errorf("expected %d results after trimming, got %d", maxResults, len(status.Results))
		}

		// Verify the newest results are kept (should be results 5-9, not 0-4)
		if len(status.Results) > 0 {
			firstResult := status.Results[0]
			lastResult := status.Results[len(status.Results)-1]
			// First result should be older than last result due to append order
			if !lastResult.Timestamp.After(firstResult.Timestamp) {
				t.Error("expected results to be in chronological order")
			}
			// The last result should be the most recent one we inserted
			expectedLastDuration := time.Duration((maxResults*2-1)*10) * time.Millisecond
			if lastResult.Duration != expectedLastDuration {
				t.Errorf("expected last result duration %v, got %v", expectedLastDuration, lastResult.Duration)
			}
		}
	})

	t.Run("suite-result-limits", func(t *testing.T) {
		testSuite := &suite.Suite{Name: "test-suite", Group: "test"}

		// Insert more results than the maximum
		baseTime := time.Now().Add(-10 * time.Hour)
		for i := 0; i < maxResults*2; i++ {
			result := &suite.Result{
				Name:      testSuite.Name,
				Group:     testSuite.Group,
				Success:   i%2 == 0,
				Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
				Duration:  time.Duration(i*10) * time.Millisecond,
			}
			err := store.InsertSuiteResult(testSuite, result)
			if err != nil {
				t.Fatalf("failed to insert suite result %d: %v", i, err)
			}
		}

		// Verify only maxResults are kept
		status, err := store.GetSuiteStatusByKey(testSuite.Key(), &paging.SuiteStatusParams{})
		if err != nil {
			t.Fatalf("failed to get suite status: %v", err)
		}
		if len(status.Results) != maxResults {
			t.Errorf("expected %d results after trimming, got %d", maxResults, len(status.Results))
		}

		// Verify the newest results are kept (should be results 5-9, not 0-4)
		if len(status.Results) > 0 {
			firstResult := status.Results[0]
			lastResult := status.Results[len(status.Results)-1]
			// First result should be older than last result due to append order
			if !lastResult.Timestamp.After(firstResult.Timestamp) {
				t.Error("expected results to be in chronological order")
			}
			// The last result should be the most recent one we inserted
			expectedLastDuration := time.Duration((maxResults*2-1)*10) * time.Millisecond
			if lastResult.Duration != expectedLastDuration {
				t.Errorf("expected last result duration %v, got %v", expectedLastDuration, lastResult.Duration)
			}
		}
	})
}

func TestSuiteResultOrdering(t *testing.T) {
	store, err := NewStore(10, 5)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	defer store.Clear()

	testSuite := &suite.Suite{Name: "ordering-suite", Group: "test"}

	// Insert results with distinct timestamps
	baseTime := time.Now().Add(-5 * time.Hour)
	timestamps := make([]time.Time, 5)

	for i := 0; i < 5; i++ {
		timestamp := baseTime.Add(time.Duration(i) * time.Hour)
		timestamps[i] = timestamp
		result := &suite.Result{
			Name:      testSuite.Name,
			Group:     testSuite.Group,
			Success:   true,
			Timestamp: timestamp,
			Duration:  time.Duration(i*100) * time.Millisecond,
		}
		err := store.InsertSuiteResult(testSuite, result)
		if err != nil {
			t.Fatalf("failed to insert result %d: %v", i, err)
		}
	}

	t.Run("chronological-append-order", func(t *testing.T) {
		status, err := store.GetSuiteStatusByKey(testSuite.Key(), nil)
		if err != nil {
			t.Fatalf("failed to get suite status: %v", err)
		}

		// Verify results are in chronological order (oldest first due to append)
		for i := 0; i < len(status.Results)-1; i++ {
			current := status.Results[i]
			next := status.Results[i+1]
			if !next.Timestamp.After(current.Timestamp) {
				t.Errorf("result %d timestamp %v should be before result %d timestamp %v",
					i, current.Timestamp, i+1, next.Timestamp)
			}
		}

		// Verify specific timestamp order
		if !status.Results[0].Timestamp.Equal(timestamps[0]) {
			t.Errorf("first result timestamp should be %v, got %v", timestamps[0], status.Results[0].Timestamp)
		}
		if !status.Results[len(status.Results)-1].Timestamp.Equal(timestamps[len(timestamps)-1]) {
			t.Errorf("last result timestamp should be %v, got %v", timestamps[len(timestamps)-1], status.Results[len(status.Results)-1].Timestamp)
		}
	})

	t.Run("pagination-newest-first", func(t *testing.T) {
		// Test reverse pagination (newest first in paginated results)
		page1 := ShallowCopySuiteStatus(
			&suite.Status{
				Name: testSuite.Name, Group: testSuite.Group, Key: testSuite.Key(),
				Results: []*suite.Result{
					{Timestamp: timestamps[0], Duration: 0 * time.Millisecond},
					{Timestamp: timestamps[1], Duration: 100 * time.Millisecond},
					{Timestamp: timestamps[2], Duration: 200 * time.Millisecond},
					{Timestamp: timestamps[3], Duration: 300 * time.Millisecond},
					{Timestamp: timestamps[4], Duration: 400 * time.Millisecond},
				},
			},
			paging.NewSuiteStatusParams().WithPagination(1, 3),
		)

		if len(page1.Results) != 3 {
			t.Errorf("expected 3 results in page 1, got %d", len(page1.Results))
		}

		// With reverse pagination, page 1 should have the 3 newest results
		// That means results[2], results[3], results[4] from original array
		if page1.Results[0].Duration != 200*time.Millisecond {
			t.Errorf("expected first result in page to have 200ms duration, got %v", page1.Results[0].Duration)
		}
		if page1.Results[2].Duration != 400*time.Millisecond {
			t.Errorf("expected last result in page to have 400ms duration, got %v", page1.Results[2].Duration)
		}
	})

	t.Run("trimming-preserves-newest", func(t *testing.T) {
		limitedStore, err := NewStore(3, 2) // Very small limits
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		defer limitedStore.Clear()

		smallSuite := &suite.Suite{Name: "small-suite", Group: "test"}

		// Insert 6 results, should keep only the newest 3
		for i := 0; i < 6; i++ {
			result := &suite.Result{
				Name:      smallSuite.Name,
				Group:     smallSuite.Group,
				Success:   true,
				Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
				Duration:  time.Duration(i*50) * time.Millisecond,
			}
			err := limitedStore.InsertSuiteResult(smallSuite, result)
			if err != nil {
				t.Fatalf("failed to insert result %d: %v", i, err)
			}
		}

		status, err := limitedStore.GetSuiteStatusByKey(smallSuite.Key(), nil)
		if err != nil {
			t.Fatalf("failed to get suite status: %v", err)
		}

		if len(status.Results) != 3 {
			t.Errorf("expected 3 results after trimming, got %d", len(status.Results))
		}

		// Should have results 3, 4, 5 (the newest ones)
		expectedDurations := []time.Duration{150 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond}
		for i, expectedDuration := range expectedDurations {
			if status.Results[i].Duration != expectedDuration {
				t.Errorf("result %d should have duration %v, got %v", i, expectedDuration, status.Results[i].Duration)
			}
		}
	})
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store, err := NewStore(100, 50)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	defer store.Clear()

	t.Run("concurrent-endpoint-insertions", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		resultsPerGoroutine := 5

		// Create endpoints for concurrent testing
		endpoints := make([]*endpoint.Endpoint, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			endpoints[i] = &endpoint.Endpoint{
				Name:  "endpoint-" + string(rune('A'+i)),
				Group: "concurrent",
				URL:   "https://example.com/" + string(rune('A'+i)),
			}
		}

		// Concurrently insert results for different endpoints
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(endpointIndex int) {
				defer wg.Done()
				ep := endpoints[endpointIndex]
				for j := 0; j < resultsPerGoroutine; j++ {
					result := &endpoint.Result{
						Success:   j%2 == 0,
						Timestamp: time.Now().Add(time.Duration(j) * time.Minute),
						Duration:  time.Duration(j*10) * time.Millisecond,
					}
					if err := store.InsertEndpointResult(ep, result); err != nil {
						t.Errorf("failed to insert result for endpoint %d: %v", endpointIndex, err)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify all endpoints were created and have correct result counts
		statuses, err := store.GetAllEndpointStatuses(&paging.EndpointStatusParams{})
		if err != nil {
			t.Fatalf("failed to get all endpoint statuses: %v", err)
		}
		if len(statuses) != numGoroutines {
			t.Errorf("expected %d endpoint statuses, got %d", numGoroutines, len(statuses))
		}

		// Verify each endpoint has the correct number of results
		for _, status := range statuses {
			if len(status.Results) != resultsPerGoroutine {
				t.Errorf("endpoint %s should have %d results, got %d", status.Name, resultsPerGoroutine, len(status.Results))
			}
		}
	})

	t.Run("concurrent-suite-insertions", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 5
		resultsPerGoroutine := 3

		// Create suites for concurrent testing
		suites := make([]*suite.Suite, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			suites[i] = &suite.Suite{
				Name:  "suite-" + string(rune('A'+i)),
				Group: "concurrent",
			}
		}

		// Concurrently insert results for different suites
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(suiteIndex int) {
				defer wg.Done()
				su := suites[suiteIndex]
				for j := 0; j < resultsPerGoroutine; j++ {
					result := &suite.Result{
						Name:      su.Name,
						Group:     su.Group,
						Success:   j%2 == 0,
						Timestamp: time.Now().Add(time.Duration(j) * time.Minute),
						Duration:  time.Duration(j*50) * time.Millisecond,
					}
					if err := store.InsertSuiteResult(su, result); err != nil {
						t.Errorf("failed to insert result for suite %d: %v", suiteIndex, err)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify all suites were created and have correct result counts
		statuses, err := store.GetAllSuiteStatuses(&paging.SuiteStatusParams{})
		if err != nil {
			t.Fatalf("failed to get all suite statuses: %v", err)
		}
		if len(statuses) != numGoroutines {
			t.Errorf("expected %d suite statuses, got %d", numGoroutines, len(statuses))
		}

		// Verify each suite has the correct number of results
		for _, status := range statuses {
			if len(status.Results) != resultsPerGoroutine {
				t.Errorf("suite %s should have %d results, got %d", status.Name, resultsPerGoroutine, len(status.Results))
			}
		}
	})

	t.Run("concurrent-mixed-operations", func(t *testing.T) {
		var wg sync.WaitGroup

		// Setup test data
		ep := &endpoint.Endpoint{Name: "mixed-endpoint", Group: "test", URL: "https://example.com"}
		testSuite := &suite.Suite{Name: "mixed-suite", Group: "test"}

		// Concurrent endpoint insertions
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				result := &endpoint.Result{
					Success:   true,
					Timestamp: time.Now(),
					Duration:  time.Duration(i*10) * time.Millisecond,
				}
				store.InsertEndpointResult(ep, result)
			}
		}()

		// Concurrent suite insertions
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				result := &suite.Result{
					Name:      testSuite.Name,
					Group:     testSuite.Group,
					Success:   true,
					Timestamp: time.Now(),
					Duration:  time.Duration(i*20) * time.Millisecond,
				}
				store.InsertSuiteResult(testSuite, result)
			}
		}()

		// Concurrent reads
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				store.GetAllEndpointStatuses(&paging.EndpointStatusParams{})
				store.GetAllSuiteStatuses(&paging.SuiteStatusParams{})
				time.Sleep(1 * time.Millisecond)
			}
		}()

		wg.Wait()

		// Verify final state is consistent
		endpointStatuses, err := store.GetAllEndpointStatuses(&paging.EndpointStatusParams{})
		if err != nil {
			t.Fatalf("failed to get endpoint statuses after concurrent operations: %v", err)
		}
		if len(endpointStatuses) == 0 {
			t.Error("expected at least one endpoint status after concurrent operations")
		}

		suiteStatuses, err := store.GetAllSuiteStatuses(&paging.SuiteStatusParams{})
		if err != nil {
			t.Fatalf("failed to get suite statuses after concurrent operations: %v", err)
		}
		if len(suiteStatuses) == 0 {
			t.Error("expected at least one suite status after concurrent operations")
		}
	})
}
