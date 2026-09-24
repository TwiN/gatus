package api

import (
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/logr"
)

type remoteCircuitBreakerState struct {
	mu        sync.Mutex
	failures  int
	openUntil time.Time
}

var remoteCircuitBreakers sync.Map

func resetRemoteCircuitBreakers() {
	remoteCircuitBreakers = sync.Map{}
}

func getRemoteCircuitBreaker(instanceURL string) *remoteCircuitBreakerState {
	state, _ := remoteCircuitBreakers.LoadOrStore(instanceURL, &remoteCircuitBreakerState{})
	return state.(*remoteCircuitBreakerState)
}

func isRemoteCircuitOpen(instanceURL string, circuitBreaker *remote.CircuitBreakerConfig) bool {
	state := getRemoteCircuitBreaker(instanceURL)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.openUntil.IsZero() {
		return false
	}
	if time.Now().Before(state.openUntil) {
		return true
	}
	state.failures = 0
	state.openUntil = time.Time{}
	return false
}

func recordRemoteCircuitSuccess(instanceURL string) {
	state := getRemoteCircuitBreaker(instanceURL)
	state.mu.Lock()
	state.failures = 0
	state.openUntil = time.Time{}
	state.mu.Unlock()
}

func recordRemoteCircuitFailure(instanceURL string, circuitBreaker *remote.CircuitBreakerConfig) {
	state := getRemoteCircuitBreaker(instanceURL)
	threshold := circuitBreaker.FailureThresholdOrDefault()
	openDuration := circuitBreaker.OpenDurationOrDefault()
	state.mu.Lock()
	state.failures++
	if state.failures >= threshold {
		state.openUntil = time.Now().Add(openDuration)
		logr.Warnf("[api.remoteCircuitBreaker] Opening circuit for remote %s after %d consecutive failures", instanceURL, state.failures)
	}
	state.mu.Unlock()
}
