package endpoint

import (
	"strings"
	"time"
)

// FailureReason represents a failure with its timestamp
type FailureReason struct {
	// Description is the error message or failed condition
	Description string `json:"description"`

	// Timestamp is when this failure was first observed
	Timestamp time.Time `json:"timestamp"`
}

// Event is something that happens at a specific time
type Event struct {
	// Type is the kind of event
	Type EventType `json:"type"`

	// Timestamp is the moment at which the event happened
	Timestamp time.Time `json:"timestamp"`

	// Errors is a list of errors that occurred during the health check (for UNHEALTHY events)
	// Deprecated: Use ErrorReasons for new implementations
	Errors []string `json:"errors,omitempty"`

	// FailedConditions is a list of condition expressions that failed (for UNHEALTHY events)
	// Deprecated: Use FailedConditionReasons for new implementations
	FailedConditions []string `json:"failedConditions,omitempty"`

	// ErrorReasons is a list of errors with their timestamps (for UNHEALTHY events)
	ErrorReasons []FailureReason `json:"errorReasons,omitempty"`

	// FailedConditionReasons is a list of failed conditions with their timestamps (for UNHEALTHY events)
	FailedConditionReasons []FailureReason `json:"failedConditionReasons,omitempty"`
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
		// Capture error messages with timestamps
		if len(result.Errors) > 0 {
			event.ErrorReasons = make([]FailureReason, len(result.Errors))
			for i, err := range result.Errors {
				event.ErrorReasons[i] = FailureReason{
					Description: err,
					Timestamp:   result.Timestamp,
				}
			}
			// Also populate deprecated field for backwards compatibility
			event.Errors = make([]string, len(result.Errors))
			copy(event.Errors, result.Errors)
		}
		// Capture failed conditions with timestamps
		if len(result.ConditionResults) > 0 {
			event.FailedConditionReasons = make([]FailureReason, 0)
			event.FailedConditions = make([]string, 0)
			for _, conditionResult := range result.ConditionResults {
				if !conditionResult.Success {
					event.FailedConditionReasons = append(event.FailedConditionReasons, FailureReason{
						Description: conditionResult.Condition,
						Timestamp:   result.Timestamp,
					})
					// Also populate deprecated field for backwards compatibility
					event.FailedConditions = append(event.FailedConditions, conditionResult.Condition)
				}
			}
		}
	}
	return event
}

// extractConditionPattern extracts the base condition pattern without the actual value
// For example: "[RESPONSE_TIME] (2280) < 500" becomes "[RESPONSE_TIME] < 500"
func extractConditionPattern(condition string) string {
	// Look for pattern like "[SOMETHING] (actual_value) operator expected_value"
	// We want to remove the (actual_value) part
	var result []rune
	inParens := false
	for _, char := range condition {
		if char == '(' {
			inParens = true
			continue
		}
		if char == ')' {
			inParens = false
			continue
		}
		if !inParens {
			result = append(result, char)
		}
	}
	// Clean up extra spaces
	pattern := string(result)
	for strings.Contains(pattern, "  ") {
		pattern = strings.ReplaceAll(pattern, "  ", " ")
	}
	return strings.TrimSpace(pattern)
}

// AddUniqueFailuresFromResult adds new unique errors and failed conditions from a result to an existing UNHEALTHY event
func (e *Event) AddUniqueFailuresFromResult(result *Result) {
	if e.Type != EventUnhealthy {
		return
	}

	// Add unique errors with timestamps
	if len(result.Errors) > 0 {
		for _, newError := range result.Errors {
			found := false
			for _, existingErrorReason := range e.ErrorReasons {
				if existingErrorReason.Description == newError {
					found = true
					break
				}
			}
			if !found {
				e.ErrorReasons = append(e.ErrorReasons, FailureReason{
					Description: newError,
					Timestamp:   result.Timestamp,
				})
				// Also update deprecated field for backwards compatibility
				e.Errors = append(e.Errors, newError)
			}
		}
	}

	// Add unique failed conditions with timestamps (deduplicate by pattern, not exact match)
	if len(result.ConditionResults) > 0 {
		for _, conditionResult := range result.ConditionResults {
			if !conditionResult.Success {
				newPattern := extractConditionPattern(conditionResult.Condition)
				found := false
				for _, existingConditionReason := range e.FailedConditionReasons {
					existingPattern := extractConditionPattern(existingConditionReason.Description)
					if existingPattern == newPattern {
						found = true
						break
					}
				}
				if !found {
					e.FailedConditionReasons = append(e.FailedConditionReasons, FailureReason{
						Description: conditionResult.Condition,
						Timestamp:   result.Timestamp,
					})
					// Also update deprecated field for backwards compatibility
					e.FailedConditions = append(e.FailedConditions, conditionResult.Condition)
				}
			}
		}
	}
}
