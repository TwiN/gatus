package endpoint

import (
	"errors"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
)

func TestValidateEndpointNameGroupAndAlerts(t *testing.T) {
	scenarios := []struct {
		name        string
		group       string
		alerts      []*alert.Alert
		expectedErr error
	}{
		{
			name:   "n",
			group:  "g",
			alerts: []*alert.Alert{{Type: "slack"}},
		},
		{
			name:   "n",
			alerts: []*alert.Alert{{Type: "slack"}},
		},
		{
			group:       "g",
			alerts:      []*alert.Alert{{Type: "slack"}},
			expectedErr: ErrEndpointWithNoName,
		},
		{
			name:        "\"",
			alerts:      []*alert.Alert{{Type: "slack"}},
			expectedErr: ErrEndpointWithInvalidNameOrGroup,
		},
		{
			name:        "n",
			group:       "\\",
			alerts:      []*alert.Alert{{Type: "slack"}},
			expectedErr: ErrEndpointWithInvalidNameOrGroup,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := validateEndpointNameGroupAndAlerts(scenario.name, scenario.group, scenario.alerts)
			if !errors.Is(err, scenario.expectedErr) {
				t.Errorf("expected error to be %v but got %v", scenario.expectedErr, err)
			}
		})
	}
}
