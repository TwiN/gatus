package database

import "github.com/TwinProduction/gatus/core"

func generateEventBasedOnResult(result *core.Result) *core.Event {
	var eventType core.EventType
	if result.Success {
		eventType = core.EventHealthy
	} else {
		eventType = core.EventUnhealthy
	}
	return &core.Event{
		Type:      eventType,
		Timestamp: result.Timestamp,
	}
}
