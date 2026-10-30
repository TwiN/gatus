package watchdog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"golang.org/x/sync/semaphore"
)

// TestExecuteEndpoint_NoHistoryExecutesImmediately verifies that a fresh endpoint with
// no persisted history is executed synchronously without waiting for a seeded delay.
func TestExecuteEndpoint_NoHistoryExecutesImmediately(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatalf("store.Initialize: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	ep := &endpoint.Endpoint{
		Name:       "no-history",
		Group:      "test",
		URL:        server.URL,
		Interval:   100 * time.Millisecond,
		Conditions: []endpoint.Condition{"[STATUS] == 200"},
	}
	if err := ep.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("ValidateAndSetDefaults: %v", err)
	}
	cfg := &config.Config{Maintenance: maintenance.GetDefaultConfig()}
	ctx, cancelFunc = context.WithCancel(context.Background())
	monitoringSemaphore = semaphore.NewWeighted(1)
	defer cancelFunc()
	if delay := endpointInitialDelay(ep.Key(), ep.Interval); delay > 0 {
		t.Fatalf("expected no positive initial delay for endpoint with no history, got %v", delay)
	}
	start := time.Now()
	executeEndpoint(ep, cfg, nil)
	if elapsed := time.Since(start); elapsed >= ep.Interval {
		t.Fatalf("expected executeEndpoint to return quickly, took %v (interval=%v)", elapsed, ep.Interval)
	}
	status, err := store.Get().GetEndpointStatusByKey(ep.Key(), paging.NewEndpointStatusParams().WithResults(1, 1))
	if err != nil {
		t.Fatalf("GetEndpointStatusByKey: %v", err)
	}
	if len(status.Results) != 1 {
		t.Fatalf("expected exactly 1 persisted result, got %d", len(status.Results))
	}
	if !status.Results[0].Success {
		t.Errorf("expected persisted result to be successful")
	}
}

// TestMonitorEndpoint_RecentHistoryDelaysNextExecution verifies that an endpoint with a
// recent persisted result waits out the remainder of the seeded delay before its next
// execution, instead of executing immediately and instead of waiting a full extra interval.
func TestMonitorEndpoint_RecentHistoryDelaysNextExecution(t *testing.T) {
	if err := store.Initialize(nil); err != nil {
		t.Fatalf("store.Initialize: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	ep := &endpoint.Endpoint{
		Name:       "recent-history",
		Group:      "test",
		URL:        server.URL,
		Interval:   300 * time.Millisecond,
		Conditions: []endpoint.Condition{"[STATUS] == 200"},
	}
	if err := ep.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("ValidateAndSetDefaults: %v", err)
	}
	if err := store.Get().InsertEndpointResult(ep, &endpoint.Result{Success: true, Timestamp: time.Now().Add(-ep.Interval / 2)}); err != nil {
		t.Fatalf("InsertEndpointResult: %v", err)
	}
	cfg := &config.Config{Maintenance: maintenance.GetDefaultConfig()}
	ctx, cancelFunc = context.WithCancel(context.Background())
	monitoringSemaphore = semaphore.NewWeighted(1)
	defer cancelFunc()
	start := time.Now()
	go monitorEndpoint(ep, cfg, nil, ctx)
	deadline := time.Now().Add(2 * time.Second)
	for {
		status, err := store.Get().GetEndpointStatusByKey(ep.Key(), paging.NewEndpointStatusParams().WithResults(1, 10))
		if err != nil {
			t.Fatalf("GetEndpointStatusByKey: %v", err)
		}
		if len(status.Results) >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for second execution; results so far: %d", len(status.Results))
		}
		time.Sleep(10 * time.Millisecond)
	}
	elapsed := time.Since(start)
	if elapsed < ep.Interval/2-100*time.Millisecond {
		t.Errorf("expected second execution to honor seeded delay of ~%v, but it happened after only %v", ep.Interval/2, elapsed)
	}
	if elapsed >= ep.Interval+200*time.Millisecond {
		t.Errorf("expected second execution within ~one interval (%v) of start, took %v", ep.Interval, elapsed)
	}
}
