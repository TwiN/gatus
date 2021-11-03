package core

import (
	"time"
)

// Result of the evaluation of a Endpoint
type Result struct {
	// HTTPStatus is the HTTP response status code
	HTTPStatus int `json:"status"`

	// DNSRCode is the response code of a DNS query in a human-readable format
	DNSRCode string `json:"-"`

	// Hostname extracted from Endpoint.URL
	Hostname string `json:"hostname"`

	// IP resolved from the Endpoint URL
	IP string `json:"-"`

	// Connected whether a connection to the host was established successfully
	Connected bool `json:"-"`

	// Duration time that the request took
	Duration time.Duration `json:"duration"`

	// Errors encountered during the evaluation of the Endpoint's health
	Errors []string `json:"errors"`

	// ConditionResults results of the Endpoint's conditions
	ConditionResults []*ConditionResult `json:"conditionResults"`

	// Success whether the result signifies a success or not
	Success bool `json:"success"`

	// Timestamp when the request was sent
	Timestamp time.Time `json:"timestamp"`

	// CertificateExpiration is the duration before the certificate expires
	CertificateExpiration time.Duration `json:"-"`

	// body is the response body
	//
	// Note that this variable is only used during the evaluation of an Endpoint's health.
	// This means that the call Endpoint.EvaluateHealth both populates the body (if necessary)
	// and sets it to nil after the evaluation has been completed.
	body []byte
}

// AddError adds an error to the result's list of errors.
// It also ensures that there are no duplicates.
func (r *Result) AddError(error string) {
	for _, resultError := range r.Errors {
		if resultError == error {
			// If the error already exists, don't add it
			return
		}
	}
	r.Errors = append(r.Errors, error)
}
