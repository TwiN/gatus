package endpoint

import "github.com/TwiN/gatus/v5/config/key"

// Status contains the evaluation Results of an Endpoint
// This is essentially a DTO
type Status struct {
	// Name of the endpoint
	Name string `json:"name,omitempty"`

	// Group the endpoint is a part of. Used for grouping multiple endpoints together on the front end.
	Group string `json:"group,omitempty"`

	// Key of the Endpoint
	Key string `json:"key"`

	// Suspended is whether the endpoint's monitoring is currently suspended. This is populated from the
	// configuration by the API layer, not persisted by the storage provider.
	Suspended bool `json:"suspended,omitempty"`

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
func NewStatus(group, name string) *Status {
	return &Status{
		Name:    name,
		Group:   group,
		Key:     key.ConvertGroupAndNameToKey(group, name),
		Results: make([]*Result, 0),
		Events:  make([]*Event, 0),
		Uptime:  NewUptime(),
	}
}
