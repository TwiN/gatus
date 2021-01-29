package core

import "time"

// Event is something that happens at a specific time
type Event struct {
	// Type is the kind of event
	Type EventType `json:"type"`

	// Timestamp is the moment at which the event happened
	Timestamp time.Time `json:"timestamp"`
}

// EventType is, uh, the types of events?
type EventType string

var (
	// EventStart is a type of event that represents when a service starts being monitored
	EventStart EventType = "START"

	// EventHealthy is a type of event that represents a service passing all of its conditions
	EventHealthy EventType = "HEALTHY"

	// EventUnhealthy is a type of event that represents a service failing one or more of its conditions
	EventUnhealthy EventType = "UNHEALTHY"
)
