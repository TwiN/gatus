package endpoint

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/test"
)

func TestEndpoint_EvaluateHealthWithMaintenanceConditions(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	scenarios := []struct {
		name                       string
		body                       string
		maintenanceMessage         string
		expectedSuccess            bool
		expectedMaintenance        bool
		expectedMaintenanceMessage string
	}{
		{
			name:                       "maintenance-with-placeholder-message",
			body:                       `{"status": "maintenance", "database": "healthy", "detail": "The API is under maintenance until 10:00 UTC."}`,
			maintenanceMessage:         "[BODY].detail",
			expectedSuccess:            true,
			expectedMaintenance:        true,
			expectedMaintenanceMessage: "The API is under maintenance until 10:00 UTC.",
		},
		{
			name:                       "maintenance-with-literal-message",
			body:                       `{"status": "maintenance", "database": "healthy"}`,
			maintenanceMessage:         "Planned maintenance",
			expectedSuccess:            true,
			expectedMaintenance:        true,
			expectedMaintenanceMessage: "Planned maintenance",
		},
		{
			name:                       "maintenance-with-unresolvable-message",
			body:                       `{"status": "maintenance", "database": "healthy"}`,
			maintenanceMessage:         "[BODY].detail",
			expectedSuccess:            true,
			expectedMaintenance:        true,
			expectedMaintenanceMessage: "",
		},
		{
			name:                "no-maintenance",
			body:                `{"status": "healthy", "database": "healthy", "detail": "All good."}`,
			maintenanceMessage:  "[BODY].detail",
			expectedSuccess:     true,
			expectedMaintenance: false,
		},
		{
			// A failure is more important than a maintenance, so a failed result is never marked as maintenance
			name:                "failed-result-is-not-maintenance",
			body:                `{"status": "maintenance", "database": "down", "detail": "The API is under maintenance."}`,
			maintenanceMessage:  "[BODY].detail",
			expectedSuccess:     false,
			expectedMaintenance: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(scenario.body))}
			})})
			ep := Endpoint{
				Name:                  "api",
				URL:                   "https://example.org/status",
				Conditions:            []Condition{"[STATUS] == 200", "[BODY].status == any(healthy, maintenance)", "[BODY].database == healthy"},
				MaintenanceConditions: []Condition{"[BODY].status == maintenance"},
				MaintenanceMessage:    scenario.maintenanceMessage,
			}
			if err := ep.ValidateAndSetDefaults(); err != nil {
				t.Fatal("endpoint validation failed:", err)
			}
			result := ep.EvaluateHealth()
			if result.Success != scenario.expectedSuccess {
				t.Errorf("expected success=%v, got %v (conditions: %v)", scenario.expectedSuccess, result.Success, result.ConditionResults)
			}
			if result.Maintenance != scenario.expectedMaintenance {
				t.Errorf("expected maintenance=%v, got %v", scenario.expectedMaintenance, result.Maintenance)
			}
			if result.MaintenanceMessage != scenario.expectedMaintenanceMessage {
				t.Errorf("expected maintenance message %q, got %q", scenario.expectedMaintenanceMessage, result.MaintenanceMessage)
			}
			// Maintenance conditions must never show up as condition results, nor add errors
			if len(result.ConditionResults) != len(ep.Conditions) {
				t.Errorf("expected %d condition results, got %d", len(ep.Conditions), len(result.ConditionResults))
			}
			if len(result.Errors) != 0 {
				t.Errorf("expected no errors, got %v", result.Errors)
			}
		})
	}
}

func TestEndpoint_ValidateAndSetDefaultsWithMaintenanceConditions(t *testing.T) {
	ep := Endpoint{
		Name:               "api",
		URL:                "https://example.org/status",
		Conditions:         []Condition{"[STATUS] == 200"},
		MaintenanceMessage: "[BODY].detail",
	}
	if err := ep.ValidateAndSetDefaults(); !errors.Is(err, ErrMaintenanceMessageWithoutMaintenanceConditions) {
		t.Errorf("expected %v, got %v", ErrMaintenanceMessageWithoutMaintenanceConditions, err)
	}
	ep = Endpoint{
		Name:                  "api",
		URL:                   "https://example.org/status",
		Conditions:            []Condition{"[STATUS] == 200"},
		MaintenanceConditions: []Condition{"[BODY].status"},
	}
	if err := ep.ValidateAndSetDefaults(); err == nil {
		t.Error("expected an invalid maintenance condition to fail validation")
	}
}

func TestEndpoint_needsToReadBodyWithMaintenanceConditions(t *testing.T) {
	if !(&Endpoint{Conditions: []Condition{"[STATUS] == 200"}, MaintenanceConditions: []Condition{"[BODY].status == maintenance"}}).needsToReadBody() {
		t.Error("expected body to be needed by a maintenance condition")
	}
	if !(&Endpoint{Conditions: []Condition{"[STATUS] == 200"}, MaintenanceConditions: []Condition{"[STATUS] == 503"}, MaintenanceMessage: "[BODY].detail"}).needsToReadBody() {
		t.Error("expected body to be needed by the maintenance message")
	}
}
