package endpoint

import "github.com/TwiN/gatus/v5/config/key"

// Status contains the evaluation Results of an Endpoint
// This is essentially a DTO
type Status struct {
	// Name of the endpoint
	Name string `json:"name,omitempty"`

	// Groups the endpoint is a part of. Used for grouping multiple endpoints together on the front end.
	Groups []string `json:"groups,omitempty"`

	// Key of the Endpoint
	Key string `json:"key"`

	// Results is the list of endpoint evaluation results
	Results []*Result `json:"results"`

	// Events is a list of events
	Events []*Event `json:"events,omitempty"`

	// Uptime information on the endpoint's uptime
	//
	// Used by the memory store.
	//
	// To retrieve the uptime between two time, use store.GetUptimeByKey.
	Uptime *Uptime `json:"-"`
}

// NewStatus creates a new Status
func NewStatus(groups []string, name string) *Status {
	return &Status{
		Name:    name,
		Groups:  groups,
		Key:     key.ConvertGroupAndNameToKey(groups, name),
		Results: make([]*Result, 0),
		Events:  make([]*Event, 0),
		Uptime:  NewUptime(),
	}
}
