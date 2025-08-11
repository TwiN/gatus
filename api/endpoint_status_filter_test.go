package api

import (
	"testing"

	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestFilterEndpointStatuses(t *testing.T) {
	// Create test endpoint statuses
	endpointStatuses := []*endpoint.Status{
		{
			Name:  "endpoint1",
			Group: "core",
			Results: []*endpoint.Result{
				{Success: true},
			},
		},
		{
			Name:  "endpoint2",
			Group: "api",
			Results: []*endpoint.Result{
				{Success: false},
			},
		},
		{
			Name:  "endpoint3",
			Group: "core",
			Results: []*endpoint.Result{
				{Success: false},
			},
		},
		{
			Name:  "endpoint4",
			Group: "web",
			Results: []*endpoint.Result{
				{Success: true},
			},
		},
	}

	// Test group filter
	t.Run("filter by group", func(t *testing.T) {
		filtered := filterEndpointStatuses(endpointStatuses, "core", "")
		if len(filtered) != 2 {
			t.Errorf("Expected 2 endpoints, got %d", len(filtered))
		}
		for _, ep := range filtered {
			if ep.Group != "core" {
				t.Errorf("Expected group 'core', got '%s'", ep.Group)
			}
		}
	})

	// Test status filter
	t.Run("filter by status up", func(t *testing.T) {
		filtered := filterEndpointStatuses(endpointStatuses, "", "up")
		if len(filtered) != 2 {
			t.Errorf("Expected 2 endpoints, got %d", len(filtered))
		}
		for _, ep := range filtered {
			if !ep.Results[0].Success {
				t.Errorf("Expected successful endpoint, got failed")
			}
		}
	})

	// Test status filter down
	t.Run("filter by status down", func(t *testing.T) {
		filtered := filterEndpointStatuses(endpointStatuses, "", "down")
		if len(filtered) != 2 {
			t.Errorf("Expected 2 endpoints, got %d", len(filtered))
		}
		for _, ep := range filtered {
			if ep.Results[0].Success {
				t.Errorf("Expected failed endpoint, got successful")
			}
		}
	})

	// Test combined filters
	t.Run("filter by group and status", func(t *testing.T) {
		filtered := filterEndpointStatuses(endpointStatuses, "core", "up")
		if len(filtered) != 1 {
			t.Errorf("Expected 1 endpoint, got %d", len(filtered))
		}
		if len(filtered) > 0 {
			if filtered[0].Group != "core" || !filtered[0].Results[0].Success {
				t.Errorf("Expected core group with success=true")
			}
		}
	})

	// Test no filters
	t.Run("no filters", func(t *testing.T) {
		filtered := filterEndpointStatuses(endpointStatuses, "", "")
		if len(filtered) != 4 {
			t.Errorf("Expected 4 endpoints, got %d", len(filtered))
		}
	})
}
