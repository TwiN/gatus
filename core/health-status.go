package core

// HealthStatus is the status of Gatus
type HealthStatus struct {
	// Status is the state of Gatus (UP/DOWN)
	Status string `json:"status"`

	// Message is an accompanying description of why the status is as reported.
	// If the Status is UP, no message will be provided
	Message string `json:"message,omitempty"`
}
