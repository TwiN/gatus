package g8

import (
	"sync"
	"time"
)

// RateLimiter is a fixed rate limiter
type RateLimiter struct {
	maximumExecutionsPerSecond int
	executionsLeftInWindow     int
	windowStartTime            time.Time
	mutex                      sync.Mutex
}

// NewRateLimiter creates a RateLimiter
func NewRateLimiter(maximumExecutionsPerSecond int) *RateLimiter {
	return &RateLimiter{
		windowStartTime:            time.Now(),
		executionsLeftInWindow:     maximumExecutionsPerSecond,
		maximumExecutionsPerSecond: maximumExecutionsPerSecond,
	}
}

// Try updates the number of executions if the rate limit quota hasn't been reached and returns whether the
// attempt was successful or not.
//
// Returns false if the execution was not successful (rate limit quota has been reached)
// Returns true if the execution was successful (rate limit quota has not been reached)
func (r *RateLimiter) Try() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if time.Now().Add(-time.Second).After(r.windowStartTime) {
		r.windowStartTime = time.Now()
		r.executionsLeftInWindow = r.maximumExecutionsPerSecond
	}
	if r.executionsLeftInWindow == 0 {
		return false
	}
	r.executionsLeftInWindow--
	return true
}
