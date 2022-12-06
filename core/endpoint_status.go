package core

import "github.com/TwiN/gatus/v5/util"

// EndpointStatus contains the evaluation Results of an Endpoint
type EndpointStatus struct {
	// Name of the endpoint
	Name string `json:"name,omitempty"`

	// Group the endpoint is a part of. Used for grouping multiple endpoints together on the front end.
	Group string `json:"group,omitempty"`

	// Key is the key representing the EndpointStatus
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

// NewEndpointStatus creates a new EndpointStatus
func NewEndpointStatus(group, name string) *EndpointStatus {
	return &EndpointStatus{
		Name:    name,
		Group:   group,
		Key:     util.ConvertGroupAndEndpointNameToKey(group, name),
		Results: make([]*Result, 0),
		Events:  make([]*Event, 0),
		Uptime:  NewUptime(),
	}
}
