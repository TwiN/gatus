package endpoint

import (
	"time"
)

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
	// EventStart is a type of event that represents when an endpoint starts being monitored
	EventStart EventType = "START"

	// EventHealthy is a type of event that represents an endpoint passing all of its conditions
	EventHealthy EventType = "HEALTHY"

	// EventUnhealthy is a type of event that represents an endpoint failing one or more of its conditions
	EventUnhealthy EventType = "UNHEALTHY"
)

// NewEventFromResult creates an Event from a Result
func NewEventFromResult(result *Result) *Event {
	event := &Event{Timestamp: result.Timestamp}
	if result.Success {
		event.Type = EventHealthy
	} else {
		event.Type = EventUnhealthy
	}
	return event
}
