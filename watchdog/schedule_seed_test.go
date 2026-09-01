package watchdog

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/storage/store"
)

func TestEndpointInitialDelay(t *testing.T) {
	const interval = 30 * time.Second
	ep := &endpoint.Endpoint{
		Name:  "seed-ep",
		Group: "test",
	}
	tests := []struct {
		name    string
		setup   func()
		wantPos bool // true => expect delay > 0
	}{
		{
			name:    "no history -> run immediately",
			setup:   func() {}, // no insert; store returns ErrEndpointNotFound
			wantPos: false,
		},
		{
			name: "overdue -> run immediately",
			setup: func() {
				r := &endpoint.Result{
					Timestamp: time.Now().Add(-2 * interval),
					Success:   true,
				}
				_ = store.Get().InsertEndpointResult(ep, r)
			},
			wantPos: false,
		},
		{
			name: "recent -> positive delay",
			setup: func() {
				r := &endpoint.Result{
					Timestamp: time.Now().Add(-interval / 2),
					Success:   true,
				}
				_ = store.Get().InsertEndpointResult(ep, r)
			},
			wantPos: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fresh in-memory store isolates each case.
			if err := store.Initialize(nil); err != nil {
				t.Fatalf("store.Initialize: %v", err)
			}
			tt.setup()
			delay := endpointInitialDelay(ep.Key(), interval)
			if tt.wantPos {
				if delay <= 0 {
					t.Errorf("expected delay > 0, got %v", delay)
				}
				if delay > interval {
					t.Errorf("expected delay <= interval (%v), got %v", interval, delay)
				}
			} else {
				if delay > 0 {
					t.Errorf("expected delay <= 0, got %v", delay)
				}
			}
		})
	}
}

func TestSuiteInitialDelay(t *testing.T) {
	const interval = 30 * time.Second
	s := &suite.Suite{
		Name:  "seed-suite",
		Group: "test",
	}
	tests := []struct {
		name    string
		setup   func()
		wantPos bool
	}{
		{
			name:    "no history -> run immediately",
			setup:   func() {}, // no insert; store returns ErrSuiteNotFound
			wantPos: false,
		},
		{
			name: "overdue -> run immediately",
			setup: func() {
				r := &suite.Result{
					Timestamp: time.Now().Add(-2 * interval),
					Success:   true,
				}
				_ = store.Get().InsertSuiteResult(s, r)
			},
			wantPos: false,
		},
		{
			name: "recent -> positive delay",
			setup: func() {
				r := &suite.Result{
					Timestamp: time.Now().Add(-interval / 2),
					Success:   true,
				}
				_ = store.Get().InsertSuiteResult(s, r)
			},
			wantPos: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.Initialize(nil); err != nil {
				t.Fatalf("store.Initialize: %v", err)
			}
			tt.setup()
			delay := suiteInitialDelay(s.Key(), interval)
			if tt.wantPos {
				if delay <= 0 {
					t.Errorf("expected delay > 0, got %v", delay)
				}
				if delay > interval {
					t.Errorf("expected delay <= interval (%v), got %v", interval, delay)
				}
			} else {
				if delay > 0 {
					t.Errorf("expected delay <= 0, got %v", delay)
				}
			}
		})
	}
}
