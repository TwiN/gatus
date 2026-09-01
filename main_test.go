package main

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
)

func TestRestoreNumberOfEvaluationsInARow(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatal("failed to initialize store:", err.Error())
	}
	defer store.Get().Close()
	ep := &endpoint.Endpoint{Name: "endpoint", Group: "group"}
	now := time.Now()
	// Simulate two failures in a row that haven't reached an alert's failure threshold yet, followed by a reload:
	// without restoring the in-a-row counters from the persisted results, this progress would be lost.
	for i := 0; i < 2; i++ {
		result := &endpoint.Result{Success: false, Timestamp: now.Add(time.Duration(i) * time.Minute)}
		if err := store.Get().InsertEndpointResult(ep, result); err != nil {
			t.Fatal("failed to insert endpoint result:", err.Error())
		}
	}
	// Simulate a configuration reload creating a brand-new Endpoint struct with the counters reset to 0.
	reloadedEndpoint := &endpoint.Endpoint{Name: "endpoint", Group: "group"}
	restoreNumberOfEvaluationsInARow(reloadedEndpoint, storage.DefaultMaximumNumberOfResults)
	if reloadedEndpoint.NumberOfFailuresInARow != 2 {
		t.Errorf("expected NumberOfFailuresInARow to be restored to 2, got %d", reloadedEndpoint.NumberOfFailuresInARow)
	}
	if reloadedEndpoint.NumberOfSuccessesInARow != 0 {
		t.Errorf("expected NumberOfSuccessesInARow to remain 0, got %d", reloadedEndpoint.NumberOfSuccessesInARow)
	}
}

func TestRestoreNumberOfEvaluationsInARowWithTrailingSuccesses(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatal("failed to initialize store:", err.Error())
	}
	defer store.Get().Close()
	ep := &endpoint.Endpoint{Name: "endpoint", Group: "group"}
	now := time.Now()
	results := []bool{false, false, false, true, true}
	for i, success := range results {
		result := &endpoint.Result{Success: success, Timestamp: now.Add(time.Duration(i) * time.Minute)}
		if err := store.Get().InsertEndpointResult(ep, result); err != nil {
			t.Fatal("failed to insert endpoint result:", err.Error())
		}
	}
	reloadedEndpoint := &endpoint.Endpoint{Name: "endpoint", Group: "group"}
	restoreNumberOfEvaluationsInARow(reloadedEndpoint, storage.DefaultMaximumNumberOfResults)
	if reloadedEndpoint.NumberOfSuccessesInARow != 2 {
		t.Errorf("expected NumberOfSuccessesInARow to be restored to 2, got %d", reloadedEndpoint.NumberOfSuccessesInARow)
	}
	if reloadedEndpoint.NumberOfFailuresInARow != 0 {
		t.Errorf("expected NumberOfFailuresInARow to remain 0, got %d", reloadedEndpoint.NumberOfFailuresInARow)
	}
}

func TestRestoreNumberOfEvaluationsInARowWithNoHistory(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatal("failed to initialize store:", err.Error())
	}
	defer store.Get().Close()
	ep := &endpoint.Endpoint{Name: "unknown-endpoint", Group: "group"}
	restoreNumberOfEvaluationsInARow(ep, storage.DefaultMaximumNumberOfResults)
	if ep.NumberOfFailuresInARow != 0 || ep.NumberOfSuccessesInARow != 0 {
		t.Errorf("expected counters to remain 0 when there's no persisted history, got failures=%d successes=%d", ep.NumberOfFailuresInARow, ep.NumberOfSuccessesInARow)
	}
}
