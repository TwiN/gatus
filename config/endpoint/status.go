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

// NewStatus creates a new Status with a key derived from the group and name
func NewStatus(group, name string) *Status {
	return NewStatusWithKey(key.ConvertGroupAndNameToKey(group, name), group, name)
}

// NewStatusWithKey creates a new Status with an explicit key, which may differ from the derived one when the endpoint has an explicit key configured
func NewStatusWithKey(statusKey, group, name string) *Status {
	return &Status{
		Name:    name,
		Group:   group,
		Key:     statusKey,
		Results: make([]*Result, 0),
		Events:  make([]*Event, 0),
		Uptime:  NewUptime(),
	}
}
