package core

import (
	"time"
)

type HealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Result of the evaluation of a Service
type Result struct {
	// HttpStatus is the HTTP response status code
	HttpStatus int `json:"status"`

	// Body is the response body
	Body []byte `json:"-"`

	// Hostname extracted from the Service Url
	Hostname string `json:"hostname"`

	// Ip resolved from the Service Url
	Ip string `json:"-"`

	// Connected whether a connection to the host was established successfully
	Connected bool `json:"-"`

	// Duration time that the request took
	Duration time.Duration `json:"duration"`

	// Errors encountered during the evaluation of the service's health
	Errors []string `json:"errors"`

	// ConditionResults results of the service's conditions
	ConditionResults []*ConditionResult `json:"condition-results"`

	// Success whether the result signifies a success or not
	Success bool `json:"success"`

	// Timestamp when the request was sent
	Timestamp time.Time `json:"timestamp"`
}

type ConditionResult struct {
	Condition string `json:"condition"`
	Success   bool   `json:"success"`
}
