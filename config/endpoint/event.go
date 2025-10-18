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

	// Errors is a list of errors that occurred during the health check (for UNHEALTHY events)
	Errors []string `json:"errors,omitempty"`

	// FailedConditions is a list of condition expressions that failed (for UNHEALTHY events)
	FailedConditions []string `json:"failedConditions,omitempty"`
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
		// Capture error messages
		if len(result.Errors) > 0 {
			event.Errors = make([]string, len(result.Errors))
			copy(event.Errors, result.Errors)
		}
		// Capture failed conditions
		if len(result.ConditionResults) > 0 {
			event.FailedConditions = make([]string, 0)
			for _, conditionResult := range result.ConditionResults {
				if !conditionResult.Success {
					event.FailedConditions = append(event.FailedConditions, conditionResult.Condition)
				}
			}
		}
	}
	return event
}
