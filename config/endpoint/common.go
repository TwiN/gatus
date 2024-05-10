package endpoint

import (
	"errors"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
)

var (
	// ErrEndpointWithNoName is the error with which Gatus will panic if an endpoint is configured with no name
	ErrEndpointWithNoName = errors.New("you must specify a name for each endpoint")

	// ErrEndpointWithInvalidNameOrGroup is the error with which Gatus will panic if an endpoint has an invalid character where it shouldn't
	ErrEndpointWithInvalidNameOrGroup = errors.New("endpoint name and group must not have \" or \\")
)

// validateEndpointNameGroupAndAlerts validates the name, group and alerts of an endpoint
func validateEndpointNameGroupAndAlerts(name, group string, alerts []*alert.Alert) error {
	if len(name) == 0 {
		return ErrEndpointWithNoName
	}
	if strings.ContainsAny(name, "\"\\") || strings.ContainsAny(group, "\"\\") {
		return ErrEndpointWithInvalidNameOrGroup
	}
	for _, endpointAlert := range alerts {
		if err := endpointAlert.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}
