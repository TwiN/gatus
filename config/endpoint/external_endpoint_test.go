package endpoint

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint/heartbeat"
	"github.com/TwiN/gatus/v5/config/maintenance"
)

func TestExternalEndpoint_ValidateAndSetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *ExternalEndpoint
		wantErr  error
	}{
		{
			name: "valid-external-endpoint",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Group: "test-group",
				Token: "valid-token",
			},
			wantErr: nil,
		},
		{
			name: "valid-external-endpoint-with-heartbeat",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Token: "valid-token",
				Heartbeat: heartbeat.Config{
					Interval: 30 * time.Second,
				},
			},
			wantErr: nil,
		},
		{
			name: "missing-token",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Group: "test-group",
			},
			wantErr: ErrExternalEndpointWithNoToken,
		},
		{
			name: "empty-token",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Token: "",
			},
			wantErr: ErrExternalEndpointWithNoToken,
		},
		{
			name: "heartbeat-interval-too-low",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Token: "valid-token",
				Heartbeat: heartbeat.Config{
					Interval: 5 * time.Second, // Less than 10 seconds
				},
			},
			wantErr: ErrExternalEndpointHeartbeatIntervalTooLow,
		},
		{
			name: "heartbeat-interval-exactly-10-seconds",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Token: "valid-token",
				Heartbeat: heartbeat.Config{
					Interval: 10 * time.Second,
				},
			},
			wantErr: nil,
		},
		{
			name: "heartbeat-interval-zero-is-allowed",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Token: "valid-token",
				Heartbeat: heartbeat.Config{
					Interval: 0, // Zero means no heartbeat monitoring
				},
			},
			wantErr: nil,
		},
		{
			name: "missing-name",
			endpoint: &ExternalEndpoint{
				Group: "test-group",
				Token: "valid-token",
			},
			wantErr: ErrEndpointWithNoName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.endpoint.ValidateAndSetDefaults()
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, but got none", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got %v", err)
				}
			}
		})
	}
}

func TestExternalEndpoint_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  *bool
		expected bool
	}{
		{
			name:     "nil-enabled-defaults-to-true",
			enabled:  nil,
			expected: true,
		},
		{
			name:     "explicitly-enabled",
			enabled:  boolPtr(true),
			expected: true,
		},
		{
			name:     "explicitly-disabled",
			enabled:  boolPtr(false),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &ExternalEndpoint{
				Name:    "test-endpoint",
				Token:   "test-token",
				Enabled: tt.enabled,
			}
			result := endpoint.IsEnabled()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExternalEndpoint_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *ExternalEndpoint
		expected string
	}{
		{
			name: "with-group",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Group: "test-group",
			},
			expected: "test-group/test-endpoint",
		},
		{
			name: "without-group",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Group: "",
			},
			expected: "test-endpoint",
		},
		{
			name: "empty-group-string",
			endpoint: &ExternalEndpoint{
				Name:  "api-health",
				Group: "",
			},
			expected: "api-health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.endpoint.DisplayName()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExternalEndpoint_Key(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *ExternalEndpoint
		expected string
	}{
		{
			name: "with-group",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Group: "test-group",
			},
			expected: "test-group_test-endpoint",
		},
		{
			name: "without-group",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Group: "",
			},
			expected: "_test-endpoint",
		},
		{
			name: "special-characters-in-name",
			endpoint: &ExternalEndpoint{
				Name:  "test endpoint with spaces",
				Group: "test-group",
			},
			expected: "test-group_test-endpoint-with-spaces",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.endpoint.Key()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExternalEndpoint_ToEndpoint(t *testing.T) {
	tests := []struct {
		name             string
		externalEndpoint *ExternalEndpoint
	}{
		{
			name: "complete-external-endpoint",
			externalEndpoint: &ExternalEndpoint{
				Enabled: boolPtr(true),
				Name:    "test-endpoint",
				Group:   "test-group",
				Token:   "test-token",
				Alerts: []*alert.Alert{
					{
						Type: alert.TypeSlack,
					},
				},
				MaintenanceWindows: []*maintenance.Config{
					{
						Start:    "02:00",
						Duration: time.Hour,
					},
				},
				NumberOfFailuresInARow:  3,
				NumberOfSuccessesInARow: 5,
			},
		},
		{
			name: "minimal-external-endpoint",
			externalEndpoint: &ExternalEndpoint{
				Name:  "minimal-endpoint",
				Token: "minimal-token",
			},
		},
		{
			name: "disabled-external-endpoint",
			externalEndpoint: &ExternalEndpoint{
				Enabled: boolPtr(false),
				Name:    "disabled-endpoint",
				Token:   "disabled-token",
			},
		},
		{
			name: "original-test-case",
			externalEndpoint: &ExternalEndpoint{
				Name:  "name",
				Group: "group",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.externalEndpoint.ToEndpoint()
			// Verify all fields are correctly copied
			if result.Enabled != tt.externalEndpoint.Enabled {
				t.Errorf("Expected Enabled=%v, got %v", tt.externalEndpoint.Enabled, result.Enabled)
			}
			if result.Name != tt.externalEndpoint.Name {
				t.Errorf("Expected Name=%q, got %q", tt.externalEndpoint.Name, result.Name)
			}
			if result.Group != tt.externalEndpoint.Group {
				t.Errorf("Expected Group=%q, got %q", tt.externalEndpoint.Group, result.Group)
			}
			if len(result.Alerts) != len(tt.externalEndpoint.Alerts) {
				t.Errorf("Expected %d alerts, got %d", len(tt.externalEndpoint.Alerts), len(result.Alerts))
			}
			if result.NumberOfFailuresInARow != tt.externalEndpoint.NumberOfFailuresInARow {
				t.Errorf("Expected NumberOfFailuresInARow=%d, got %d", tt.externalEndpoint.NumberOfFailuresInARow, result.NumberOfFailuresInARow)
			}
			if result.NumberOfSuccessesInARow != tt.externalEndpoint.NumberOfSuccessesInARow {
				t.Errorf("Expected NumberOfSuccessesInARow=%d, got %d", tt.externalEndpoint.NumberOfSuccessesInARow, result.NumberOfSuccessesInARow)
			}
			// Original test assertions
			if tt.externalEndpoint.Key() != result.Key() {
				t.Errorf("expected %s, got %s", tt.externalEndpoint.Key(), result.Key())
			}
			if tt.externalEndpoint.DisplayName() != result.DisplayName() {
				t.Errorf("expected %s, got %s", tt.externalEndpoint.DisplayName(), result.DisplayName())
			}
			// Verify it's a proper Endpoint type
			if result == nil {
				t.Error("ToEndpoint() returned nil")
			}
		})
	}
}

func TestExternalEndpoint_ValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *ExternalEndpoint
		wantErr  bool
	}{
		{
			name: "very-long-name",
			endpoint: &ExternalEndpoint{
				Name:  "this-is-a-very-long-endpoint-name-that-might-cause-issues-in-some-systems-but-should-be-handled-gracefully",
				Token: "valid-token",
			},
			wantErr: false,
		},
		{
			name: "special-characters-in-name",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint@#$%^&*()",
				Token: "valid-token",
			},
			wantErr: false,
		},
		{
			name: "unicode-characters-in-name",
			endpoint: &ExternalEndpoint{
				Name:  "测试端点",
				Token: "valid-token",
			},
			wantErr: false,
		},
		{
			name: "very-long-token",
			endpoint: &ExternalEndpoint{
				Name:  "test-endpoint",
				Token: "very-long-token-that-should-still-be-valid-even-though-it-is-extremely-long-and-might-not-be-practical-in-real-world-scenarios",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.endpoint.ValidateAndSetDefaults()
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}
