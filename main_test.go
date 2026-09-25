package main

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
)

func TestRestoreNumberOfEvaluationsInARow(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatal("failed to initialize store:", err.Error())
	}
	defer store.Get().Close()
	alerts := []*alert.Alert{{FailureThreshold: 3, SuccessThreshold: 2}}
	ep := &endpoint.Endpoint{Name: "endpoint", Group: "group", Alerts: alerts}
	now := time.Now()
	// Simulate two failures in a row that haven't reached the alert's failure threshold of 3 yet, followed by a
	// reload: without restoring the in-a-row counters from the persisted results, this progress would be lost.
	for i := 0; i < 2; i++ {
		result := &endpoint.Result{Success: false, Timestamp: now.Add(time.Duration(i) * time.Minute)}
		if err := store.Get().InsertEndpointResult(ep, result); err != nil {
			t.Fatal("failed to insert endpoint result:", err.Error())
		}
	}
	// Simulate a configuration reload creating a brand-new Endpoint struct with the counters reset to 0.
	reloadedEndpoint := &endpoint.Endpoint{Name: "endpoint", Group: "group", Alerts: alerts}
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
	alerts := []*alert.Alert{{FailureThreshold: 3, SuccessThreshold: 2}}
	ep := &endpoint.Endpoint{Name: "endpoint", Group: "group", Alerts: alerts}
	now := time.Now()
	results := []bool{false, false, false, true, true}
	for i, success := range results {
		result := &endpoint.Result{Success: success, Timestamp: now.Add(time.Duration(i) * time.Minute)}
		if err := store.Get().InsertEndpointResult(ep, result); err != nil {
			t.Fatal("failed to insert endpoint result:", err.Error())
		}
	}
	reloadedEndpoint := &endpoint.Endpoint{Name: "endpoint", Group: "group", Alerts: alerts}
	restoreNumberOfEvaluationsInARow(reloadedEndpoint, storage.DefaultMaximumNumberOfResults)
	if reloadedEndpoint.NumberOfSuccessesInARow != 2 {
		t.Errorf("expected NumberOfSuccessesInARow to be restored to 2, got %d", reloadedEndpoint.NumberOfSuccessesInARow)
	}
	if reloadedEndpoint.NumberOfFailuresInARow != 0 {
		t.Errorf("expected NumberOfFailuresInARow to remain 0, got %d", reloadedEndpoint.NumberOfFailuresInARow)
	}
}

func TestRestoreNumberOfEvaluationsInARowWithTrailingFailureAfterSuccesses(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatal("failed to initialize store:", err.Error())
	}
	defer store.Get().Close()
	alerts := []*alert.Alert{{FailureThreshold: 3, SuccessThreshold: 2}}
	ep := &endpoint.Endpoint{Name: "endpoint", Group: "group", Alerts: alerts}
	now := time.Now()
	results := []bool{true, true, false}
	for i, success := range results {
		result := &endpoint.Result{Success: success, Timestamp: now.Add(time.Duration(i) * time.Minute)}
		if err := store.Get().InsertEndpointResult(ep, result); err != nil {
			t.Fatal("failed to insert endpoint result:", err.Error())
		}
	}
	reloadedEndpoint := &endpoint.Endpoint{Name: "endpoint", Group: "group", Alerts: alerts}
	restoreNumberOfEvaluationsInARow(reloadedEndpoint, storage.DefaultMaximumNumberOfResults)
	if reloadedEndpoint.NumberOfFailuresInARow != 1 {
		t.Errorf("expected NumberOfFailuresInARow to be restored to 1, got %d", reloadedEndpoint.NumberOfFailuresInARow)
	}
	if reloadedEndpoint.NumberOfSuccessesInARow != 0 {
		t.Errorf("expected NumberOfSuccessesInARow to remain 0, got %d", reloadedEndpoint.NumberOfSuccessesInARow)
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

// TestRestoreNumberOfEvaluationsInARowStreakAtWindowBoundary makes sure that when the failure streak is exactly as
// long as the lookup window (bounded by the endpoint's largest alert threshold), the full streak is still counted
// correctly instead of being off by one.
func TestRestoreNumberOfEvaluationsInARowStreakAtWindowBoundary(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatal("failed to initialize store:", err.Error())
	}
	defer store.Get().Close()
	alerts := []*alert.Alert{{FailureThreshold: 5, SuccessThreshold: 2}}
	ep := &endpoint.Endpoint{Name: "endpoint", Group: "group", Alerts: alerts}
	now := time.Now()
	// Exactly FailureThreshold (5) failures in a row: the lookup window is bounded to 5 results, so this exercises
	// the boundary where the oldest fetched result is still part of the streak.
	for i := 0; i < 5; i++ {
		result := &endpoint.Result{Success: false, Timestamp: now.Add(time.Duration(i) * time.Minute)}
		if err := store.Get().InsertEndpointResult(ep, result); err != nil {
			t.Fatal("failed to insert endpoint result:", err.Error())
		}
	}
	reloadedEndpoint := &endpoint.Endpoint{Name: "endpoint", Group: "group", Alerts: alerts}
	restoreNumberOfEvaluationsInARow(reloadedEndpoint, storage.DefaultMaximumNumberOfResults)
	if reloadedEndpoint.NumberOfFailuresInARow != 5 {
		t.Errorf("expected NumberOfFailuresInARow to be restored to 5, got %d", reloadedEndpoint.NumberOfFailuresInARow)
	}
}

// TestRestoreNumberOfEvaluationsInARowForExternalEndpoint makes sure the counters computed by
// restoreNumberOfEvaluationsInARow for an ExternalEndpoint's converted Endpoint are copied back onto the
// ExternalEndpoint itself, since ExternalEndpoint.ToEndpoint() returns a new, unrelated Endpoint value.
func TestRestoreNumberOfEvaluationsInARowForExternalEndpoint(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatal("failed to initialize store:", err.Error())
	}
	defer store.Get().Close()
	alerts := []*alert.Alert{{FailureThreshold: 3, SuccessThreshold: 2}}
	ee := &endpoint.ExternalEndpoint{Name: "external-endpoint", Group: "group", Alerts: alerts}
	convertedEndpoint := ee.ToEndpoint()
	now := time.Now()
	for i := 0; i < 2; i++ {
		result := &endpoint.Result{Success: false, Timestamp: now.Add(time.Duration(i) * time.Minute)}
		if err := store.Get().InsertEndpointResult(convertedEndpoint, result); err != nil {
			t.Fatal("failed to insert endpoint result:", err.Error())
		}
	}
	reloadedExternalEndpoint := &endpoint.ExternalEndpoint{Name: "external-endpoint", Group: "group", Alerts: alerts}
	reloadedConvertedEndpoint := reloadedExternalEndpoint.ToEndpoint()
	restoreNumberOfEvaluationsInARow(reloadedConvertedEndpoint, storage.DefaultMaximumNumberOfResults)
	reloadedExternalEndpoint.NumberOfFailuresInARow, reloadedExternalEndpoint.NumberOfSuccessesInARow = reloadedConvertedEndpoint.NumberOfFailuresInARow, reloadedConvertedEndpoint.NumberOfSuccessesInARow
	if reloadedExternalEndpoint.NumberOfFailuresInARow != 2 {
		t.Errorf("expected ExternalEndpoint.NumberOfFailuresInARow to be restored to 2, got %d", reloadedExternalEndpoint.NumberOfFailuresInARow)
	}
}
