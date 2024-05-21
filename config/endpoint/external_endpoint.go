package endpoint

import (
	"errors"

	"github.com/TwiN/gatus/v5/alerting/alert"
)

var (
	// ErrExternalEndpointWithNoToken is the error with which Gatus will panic if an external endpoint is configured without a token.
	ErrExternalEndpointWithNoToken = errors.New("you must specify a token for each external endpoint")
)

// ExternalEndpoint is an endpoint whose result is pushed from outside Gatus, which means that
// said endpoints are not monitored by Gatus itself; Gatus only displays their results and takes
// care of alerting
type ExternalEndpoint struct {
	// Enabled defines whether to enable the monitoring of the endpoint
	Enabled *bool `yaml:"enabled,omitempty"`

	// Name of the endpoint. Can be anything.
	Name string `yaml:"name"`

	// Group the endpoint is a part of. Used for grouping multiple endpoints together on the front end.
	Group string `yaml:"group,omitempty"`

	// Token is the bearer token that must be provided through the Authorization header to push results to the endpoint
	Token string `yaml:"token,omitempty"`

	// Alerts is the alerting configuration for the endpoint in case of failure
	Alerts []*alert.Alert `yaml:"alerts,omitempty"`

	// NumberOfFailuresInARow is the number of unsuccessful evaluations in a row
	NumberOfFailuresInARow int `yaml:"-"`

	// NumberOfSuccessesInARow is the number of successful evaluations in a row
	NumberOfSuccessesInARow int `yaml:"-"`
}

// ValidateAndSetDefaults validates the ExternalEndpoint and sets the default values
func (externalEndpoint *ExternalEndpoint) ValidateAndSetDefaults() error {
	if err := validateEndpointNameGroupAndAlerts(externalEndpoint.Name, externalEndpoint.Group, externalEndpoint.Alerts); err != nil {
		return err
	}
	if len(externalEndpoint.Token) == 0 {
		return ErrExternalEndpointWithNoToken
	}
	return nil
}

// IsEnabled returns whether the endpoint is enabled or not
func (externalEndpoint *ExternalEndpoint) IsEnabled() bool {
	if externalEndpoint.Enabled == nil {
		return true
	}
	return *externalEndpoint.Enabled
}

// DisplayName returns an identifier made up of the Name and, if not empty, the Group.
func (externalEndpoint *ExternalEndpoint) DisplayName() string {
	if len(externalEndpoint.Group) > 0 {
		return externalEndpoint.Group + "/" + externalEndpoint.Name
	}
	return externalEndpoint.Name
}

// Key returns the unique key for the Endpoint
func (externalEndpoint *ExternalEndpoint) Key() string {
	return ConvertGroupAndEndpointNameToKey(externalEndpoint.Group, externalEndpoint.Name)
}

// ToEndpoint converts the ExternalEndpoint to an Endpoint
func (externalEndpoint *ExternalEndpoint) ToEndpoint() *Endpoint {
	endpoint := &Endpoint{
		Enabled:                 externalEndpoint.Enabled,
		Name:                    externalEndpoint.Name,
		Group:                   externalEndpoint.Group,
		Alerts:                  externalEndpoint.Alerts,
		NumberOfFailuresInARow:  externalEndpoint.NumberOfFailuresInARow,
		NumberOfSuccessesInARow: externalEndpoint.NumberOfSuccessesInARow,
	}
	return endpoint
}
