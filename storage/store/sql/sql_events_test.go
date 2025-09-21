package sql

import (
	"database/sql"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
)

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// The tests in this file were originally written to reproduce and fix https://github.com/TwiN/gatus/issues/1040 //
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// TestEventCreationWithFailedLastResultQuery reproduces the issue where events stop being created
// when getLastEndpointResultSuccessValue fails to retrieve the last result
func TestEventCreationWithFailedLastResultQuery(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/test.db", false, 100, 100)
	defer store.Close()

	// Create an endpoint
	ep := &endpoint.Endpoint{
		Name:  "test-endpoint",
		Group: "test-group",
		URL:   "https://example.com",
	}

	// Insert first result - should create EventStart and first event
	result1 := &endpoint.Result{
		Success:   true,
		Timestamp: time.Now(),
	}
	err := store.InsertEndpointResult(ep, result1)
	if err != nil {
		t.Fatalf("Failed to insert first result: %v", err)
	}

	// Verify events were created
	tx, _ := store.db.Begin()
	endpointID, _, _, _ := store.getEndpointIDGroupAndNameByKey(tx, ep.Key())
	numberOfEvents, _ := store.getNumberOfEventsByEndpointID(tx, endpointID)
	tx.Commit()
	if numberOfEvents != 2 {
		t.Errorf("Expected 2 events after first insert (EventStart + EventHealthy), got %d", numberOfEvents)
	}

	// Insert second result with different state - should create new event
	result2 := &endpoint.Result{
		Success:   false,
		Timestamp: time.Now().Add(1 * time.Minute),
	}
	err = store.InsertEndpointResult(ep, result2)
	if err != nil {
		t.Fatalf("Failed to insert second result: %v", err)
	}

	// Verify new event was created
	tx, _ = store.db.Begin()
	numberOfEvents, _ = store.getNumberOfEventsByEndpointID(tx, endpointID)
	tx.Commit()
	if numberOfEvents != 3 {
		t.Errorf("Expected 3 events after state change (EventStart + EventHealthy + EventUnhealthy), got %d", numberOfEvents)
	}

	// Now simulate the problematic scenario: delete all results but keep events
	// This simulates what might happen if results are cleaned up but the query fails
	tx, _ = store.db.Begin()
	tx.Exec("DELETE FROM endpoint_results WHERE endpoint_id = ?", endpointID)
	tx.Commit()

	// Insert third result with another state change
	// Since getLastEndpointResultSuccessValue will fail (no results in table),
	// the event creation will be silently skipped
	result3 := &endpoint.Result{
		Success:   true,
		Timestamp: time.Now().Add(2 * time.Minute),
	}
	err = store.InsertEndpointResult(ep, result3)
	if err != nil {
		t.Fatalf("Failed to insert third result: %v", err)
	}

	// Check if event was created (it shouldn't be due to the bug)
	tx, _ = store.db.Begin()
	numberOfEvents, _ = store.getNumberOfEventsByEndpointID(tx, endpointID)
	var lastEventType string
	tx.QueryRow("SELECT event_type FROM endpoint_events WHERE endpoint_id = ? ORDER BY event_timestamp DESC LIMIT 1", endpointID).Scan(&lastEventType)
	tx.Commit()

	// Verify the fix: event should be created even when getLastEndpointResultSuccessValue failed
	if numberOfEvents != 4 {
		t.Errorf("Expected 4 events after state change, but got %d", numberOfEvents)
		t.Logf("Last event type: %s", lastEventType)
	} else {
		t.Logf("Fix confirmed: Event was created successfully using fallback (got %d events)", numberOfEvents)
		if lastEventType != "HEALTHY" {
			t.Errorf("Expected last event to be HEALTHY, got %s", lastEventType)
		}
	}

	// Continue inserting results to verify events continue working properly
	result4 := &endpoint.Result{
		Success:   false,
		Timestamp: time.Now().Add(3 * time.Minute),
	}
	err = store.InsertEndpointResult(ep, result4)
	if err != nil {
		t.Fatalf("Failed to insert fourth result: %v", err)
	}

	result5 := &endpoint.Result{
		Success:   true,
		Timestamp: time.Now().Add(4 * time.Minute),
	}
	err = store.InsertEndpointResult(ep, result5)
	if err != nil {
		t.Fatalf("Failed to insert fifth result: %v", err)
	}

	// Check final event count - should have 6 events now (EventStart + 5 state changes)
	tx, _ = store.db.Begin()
	numberOfEvents, _ = store.getNumberOfEventsByEndpointID(tx, endpointID)
	tx.QueryRow("SELECT event_type FROM endpoint_events WHERE endpoint_id = ? ORDER BY event_timestamp DESC LIMIT 1", endpointID).Scan(&lastEventType)
	tx.Commit()

	// Verify events are being created properly
	if numberOfEvents != 6 {
		t.Errorf("Expected 6 total events after all state changes, got %d", numberOfEvents)
	} else {
		t.Logf("Success: Events continue to be created properly (total: %d events)", numberOfEvents)
	}
	if lastEventType != "HEALTHY" {
		t.Errorf("Expected final event to be HEALTHY, got %s", lastEventType)
	}
}

// TestEventCreationRaceCondition tests potential race conditions in event creation
func TestEventCreationRaceCondition(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/test.db", false, 100, 100)
	defer store.Close()

	ep := &endpoint.Endpoint{
		Name:  "race-test",
		Group: "test",
		URL:   "https://example.com",
	}

	// Insert initial result
	result1 := &endpoint.Result{
		Success:   true,
		Timestamp: time.Now(),
	}
	store.InsertEndpointResult(ep, result1)

	// Simulate concurrent inserts with alternating states
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			result := &endpoint.Result{
				Success:   index%2 == 0, // Alternate between true/false
				Timestamp: time.Now().Add(time.Duration(index) * time.Second),
			}
			store.InsertEndpointResult(ep, result)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check events
	tx, _ := store.db.Begin()
	endpointID, _, _, _ := store.getEndpointIDGroupAndNameByKey(tx, ep.Key())

	var events []struct {
		Type      string
		Timestamp time.Time
	}
	rows, _ := tx.Query("SELECT event_type, event_timestamp FROM endpoint_events WHERE endpoint_id = ? ORDER BY event_timestamp", endpointID)
	for rows.Next() {
		var e struct {
			Type      string
			Timestamp time.Time
		}
		rows.Scan(&e.Type, &e.Timestamp)
		events = append(events, e)
	}
	rows.Close()
	tx.Commit()

	t.Logf("Total events created: %d", len(events))
	for i, e := range events {
		t.Logf("Event %d: %s at %v", i+1, e.Type, e.Timestamp)
	}

	// With proper synchronization, we should have events for state changes
	// But with race conditions, events might be missing or incorrect
}

// TestEventCreationAfterMaxResultsCleanup tests event creation after results are cleaned up
func TestEventCreationAfterMaxResultsCleanup(t *testing.T) {
	// Create store with very small maximumNumberOfResults to trigger cleanup
	store, _ := NewStore("sqlite", t.TempDir()+"/test.db", false, 2, 100)
	defer store.Close()

	ep := &endpoint.Endpoint{
		Name:  "cleanup-test",
		Group: "test",
		URL:   "https://example.com",
	}

	// Insert many results to trigger cleanup
	for i := 0; i < 10; i++ {
		result := &endpoint.Result{
			Success:   i%2 == 0,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
		err := store.InsertEndpointResult(ep, result)
		if err != nil {
			t.Fatalf("Failed to insert result %d: %v", i, err)
		}
	}

	// Check how many results remain
	tx, _ := store.db.Begin()
	endpointID, _, _, _ := store.getEndpointIDGroupAndNameByKey(tx, ep.Key())
	var resultCount int
	tx.QueryRow("SELECT COUNT(*) FROM endpoint_results WHERE endpoint_id = ?", endpointID).Scan(&resultCount)

	// Check events
	var eventCount int
	tx.QueryRow("SELECT COUNT(*) FROM endpoint_events WHERE endpoint_id = ?", endpointID).Scan(&eventCount)
	tx.Commit()

	t.Logf("After 10 inserts with max 2 results:")
	t.Logf("  Results in DB: %d (should be around 2)", resultCount)
	t.Logf("  Events in DB: %d", eventCount)

	// Try to insert a result with state change after cleanup
	result := &endpoint.Result{
		Success:   true, // Different from last (was false)
		Timestamp: time.Now().Add(11 * time.Minute),
	}
	store.InsertEndpointResult(ep, result)

	// Check if new event was created
	tx, _ = store.db.Begin()
	var newEventCount int
	tx.QueryRow("SELECT COUNT(*) FROM endpoint_events WHERE endpoint_id = ?", endpointID).Scan(&newEventCount)
	tx.Commit()

	if newEventCount == eventCount {
		t.Errorf("No new event created after state change (still %d events)", eventCount)
	} else {
		t.Logf("New event created successfully (now %d events)", newEventCount)
	}
}

// TestGetLastEndpointResultSuccessValueWithNoResults specifically tests the problematic function
func TestGetLastEndpointResultSuccessValueWithNoResults(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/test.db", false, 100, 100)
	defer store.Close()

	// Create endpoint and get its ID
	ep := &endpoint.Endpoint{
		Name:  "test",
		Group: "test",
		URL:   "https://example.com",
	}

	// Insert and then delete a result to create the endpoint
	result := &endpoint.Result{
		Success:   true,
		Timestamp: time.Now(),
	}
	store.InsertEndpointResult(ep, result)

	tx, _ := store.db.Begin()
	endpointID, _, _, _ := store.getEndpointIDGroupAndNameByKey(tx, ep.Key())

	// Delete all results
	tx.Exec("DELETE FROM endpoint_results WHERE endpoint_id = ?", endpointID)
	tx.Commit()

	// Now test getLastEndpointResultSuccessValue with no results
	tx, _ = store.db.Begin()
	success, err := store.getLastEndpointResultSuccessValue(tx, endpointID)
	tx.Commit()

	if err == nil {
		t.Errorf("Expected error when no results exist, got success=%v", success)
	} else if err == sql.ErrNoRows || err == errNoRowsReturned {
		t.Logf("Correctly returned error when no results: %v", err)
	} else {
		t.Errorf("Unexpected error: %v", err)
	}
}
