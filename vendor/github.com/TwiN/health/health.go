package health

import (
	"net/http"
	"sync"
)

var (
	handler = &healthHandler{
		useJSON: false,
		status:  Up,
	}
)

// healthHandler is the HTTP handler for serving the health endpoint
type healthHandler struct {
	useJSON bool
	status  Status

	sync.RWMutex
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
	handlerStatus := h.getStatus()
	if handlerStatus == Up {
		statusCode = http.StatusOK
	} else {
		statusCode = http.StatusInternalServerError
	}
	if h.useJSON {
		writer.Header().Set("Content-Type", "application/json")
		body = []byte(`{"status":"` + handlerStatus + `"}`)
	} else {
		body = []byte(handlerStatus)
	}
	writer.WriteHeader(statusCode)
	_, _ = writer.Write(body)
}

func (h *healthHandler) getStatus() Status {
	h.Lock()
	defer h.Unlock()
	return h.status
}

func (h *healthHandler) setStatus(status Status) {
	h.Lock()
	h.status = status
	h.Unlock()
}

// Handler retrieves the health handler
func Handler() *healthHandler {
	return handler
}

// GetStatus retrieves the current status returned by the health handler
func GetStatus() Status {
	return handler.getStatus()
}

// SetStatus sets the status to be returned by the health handler
func SetStatus(status Status) {
	handler.setStatus(status)
}
