package health

import (
	"encoding/json"
	"net/http"
	"sync"
)

var (
	handler = &healthHandler{
		useJSON: false,
		status:  Up,
	}
)

type responseBody struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// healthHandler is the HTTP handler for serving the health endpoint
type healthHandler struct {
	useJSON         bool
	resetReasonOnUp bool

	status Status
	reason string

	mutex sync.RWMutex
}

// WithJSON configures whether the handler should output a response in JSON or in raw text
//
// Defaults to false
func (h *healthHandler) WithJSON(v bool) *healthHandler {
	h.useJSON = v
	return h
}

// ServeHTTP serves the HTTP request for the health handler
func (h *healthHandler) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	var statusCode int
	var body []byte
	h.mutex.RLock()
	status, reason, useJSON := h.status, h.reason, h.useJSON
	h.mutex.RUnlock()
	if status == Up {
		statusCode = http.StatusOK
	} else {
		statusCode = http.StatusInternalServerError
	}
	if useJSON {
		// We can safely ignore the error here because we know that both values are strings, therefore are supported encoders.
		body, _ = json.Marshal(responseBody{Status: string(status), Reason: reason})
		writer.Header().Set("Content-Type", "application/json")
	} else {
		if len(reason) == 0 {
			body = []byte(status)
		} else {
			body = []byte(string(status) + ": " + reason)
		}
	}
	writer.WriteHeader(statusCode)
	_, _ = writer.Write(body)
}

// Handler retrieves the health handler
func Handler() *healthHandler {
	return handler
}

// GetStatus retrieves the current status returned by the health handler
func GetStatus() Status {
	handler.mutex.RLock()
	defer handler.mutex.RUnlock()
	return handler.status
}

// SetStatus sets the status to be returned by the health handler
func SetStatus(status Status) {
	handler.mutex.Lock()
	handler.status = status
	handler.mutex.Unlock()
}

// GetReason retrieves the current status returned by the health handler
func GetReason() string {
	handler.mutex.RLock()
	defer handler.mutex.RUnlock()
	return handler.reason
}

// SetReason sets a reason for the current status to be returned by the health handler
func SetReason(reason string) {
	handler.mutex.Lock()
	handler.reason = reason
	handler.mutex.Unlock()
}

// SetStatusAndReason sets the status and reason to be returned by the health handler
func SetStatusAndReason(status Status, reason string) {
	handler.mutex.Lock()
	handler.status = status
	handler.reason = reason
	handler.mutex.Unlock()
}
