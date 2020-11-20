package core

import (
	"time"
)

// HealthStatus is the status of Gatus
type HealthStatus struct {
	// Status is the state of Gatus (UP/DOWN)
	Status string `json:"status"`

	// Message is an accompanying description of why the status is as reported.
	// If the Status is UP, no message will be provided
	Message string `json:"message,omitempty"`
}

// Result of the evaluation of a Service
type Result struct {
	// HTTPStatus is the HTTP response status code
	HTTPStatus int `json:"status"`

	// DNSRCode is the response code of DNS query in human readable version
	DNSRCode string `json:"dns-rcode"`

	// Body is the response body
	Body []byte `json:"-"`

	// Hostname extracted from the Service URL
	Hostname string `json:"hostname"`

	// IP resolved from the Service URL
	IP string `json:"-"`

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

	// CertificateExpiration is the duration before the certificate expires
	CertificateExpiration time.Duration `json:"certificate-expiration,omitempty"`
}

// ConditionResult result of a Condition
type ConditionResult struct {
	// Condition that was evaluated
	Condition string `json:"condition"`

	// Success whether the condition was met (successful) or not (failed)
	Success bool `json:"success"`
}
