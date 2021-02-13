package health

import "net/http"

var (
	handler = &healthHandler{
		useJSON: false,
		status:  Up,
	}
)

type healthHandler struct {
	useJSON bool
	status  Status
}

// WithJSON configures whether the handler should output a response in JSON or in raw text
//
// Defaults to false
func (h *healthHandler) WithJSON(v bool) *healthHandler {
	h.useJSON = v
	return h
}

func (h healthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var status int
	var body []byte
	if h.status == Up {
		status = http.StatusOK
	} else {
		status = http.StatusInternalServerError
	}
	if h.useJSON {
		writer.Header().Set("Content-Type", "application/json")
		body = []byte(`{"status":"` + h.status + `"}`)
	} else {
		body = []byte(h.status)
	}
	writer.WriteHeader(status)
	_, _ = writer.Write(body)
}

// Handler retrieves the health handler
func Handler() *healthHandler {
	return handler
}

// SetStatus sets the status to be reflected by the health handler
func SetStatus(status Status) {
	handler.status = status
}
