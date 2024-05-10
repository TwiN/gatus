package core

import (
	"testing"

	"github.com/TwiN/gatus/v5/core/result"
)

func TestNewEventFromResult(t *testing.T) {
	if event := NewEventFromResult(&result.Result{Success: true}); event.Type != EventHealthy {
		t.Error("expected event.Type to be EventHealthy")
	}
	if event := NewEventFromResult(&result.Result{Success: false}); event.Type != EventUnhealthy {
		t.Error("expected event.Type to be EventUnhealthy")
	}
}
