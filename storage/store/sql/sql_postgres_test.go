package sql

import (
	"fmt"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

// TestPostgresSuiteResultsNestedQuery tests that suite results with endpoint results
// can be retrieved without "pq: unexpected Parse response 'C'" errors.
// This bug occurs when nested queries are made in the same transaction without
// properly closing the previous query's rows first.
// See: https://github.com/TwiN/gatus/issues/1435
func TestPostgresSuiteResultsNestedQuery(t *testing.T) {
	// Start embedded PostgreSQL
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(15432).
		Database("gatus_test"))

	if err := postgres.Start(); err != nil {
		t.Fatalf("failed to start embedded postgres: %v", err)
	}
	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Errorf("failed to stop embedded postgres: %v", err)
		}
	}()

	// Create store with PostgreSQL
	connectionString := "host=localhost port=15432 user=postgres password=postgres dbname=gatus_test sslmode=disable"
	store, err := NewStore("postgres", connectionString, false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create a test suite with endpoints
	suiteEndpoint1 := &endpoint.Endpoint{
		Name:  "suite-ep-1",
		Group: "test-group",
		URL:   "https://example.com/1",
	}
	suiteEndpoint2 := &endpoint.Endpoint{
		Name:  "suite-ep-2",
		Group: "test-group",
		URL:   "https://example.com/2",
	}

	testSuite := &suite.Suite{
		Name:  "test-suite",
		Group: "test-group",
		Endpoints: []*endpoint.Endpoint{
			suiteEndpoint1,
			suiteEndpoint2,
		},
	}

	// Insert multiple suite results with endpoint results
	// This is important because the bug manifests when iterating over multiple results
	for i := 0; i < 5; i++ {
		suiteResult := &suite.Result{
			Name:      testSuite.Name,
			Group:     testSuite.Group,
			Success:   true,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Duration:  100 * time.Millisecond,
			EndpointResults: []*endpoint.Result{
				{
					Hostname:  "example.com",
					Success:   true,
					Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
					Duration:  50 * time.Millisecond,
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[STATUS] == 200", Success: true},
					},
				},
				{
					Hostname:  "example.com",
					Success:   true,
					Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
					Duration:  50 * time.Millisecond,
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[STATUS] == 200", Success: true},
					},
				},
			},
		}

		if err := store.InsertSuiteResult(testSuite, suiteResult); err != nil {
			t.Fatalf("failed to insert suite result %d: %v", i, err)
		}
	}

	// This is where the bug would occur - GetAllSuiteStatuses calls getSuiteResults
	// which performs nested queries. Without the fix, this would fail with:
	// "pq: unexpected Parse response 'C'"
	statuses, err := store.GetAllSuiteStatuses(&paging.SuiteStatusParams{})
	if err != nil {
		t.Fatalf("GetAllSuiteStatuses failed (this is the bug!): %v", err)
	}

	if len(statuses) != 1 {
		t.Errorf("expected 1 suite status, got %d", len(statuses))
	}

	// Also test GetSuiteStatusByKey which uses the same getSuiteResults function
	status, err := store.GetSuiteStatusByKey(testSuite.Key(), &paging.SuiteStatusParams{})
	if err != nil {
		t.Fatalf("GetSuiteStatusByKey failed: %v", err)
	}

	if status == nil {
		t.Fatal("expected suite status, got nil")
	}

	if len(status.Results) != 5 {
		t.Errorf("expected 5 results, got %d", len(status.Results))
	}

	// Verify endpoint results were retrieved correctly
	for i, result := range status.Results {
		if len(result.EndpointResults) != 2 {
			t.Errorf("result %d: expected 2 endpoint results, got %d", i, len(result.EndpointResults))
		}
	}

	fmt.Println("PostgreSQL nested query test passed - no 'unexpected Parse response' error!")
}
