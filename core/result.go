package core

import (
	"time"
)

// Severity type
type Severity struct {
	Low, Medium, High, Critical bool
}

// Severity status type
type SeverityStatus int

// Severity status enums
const (
	None SeverityStatus = iota
	Low
	Medium
	High
	Critical
)

// Result of the evaluation of a Endpoint
type Result struct {
	// HTTPStatus is the HTTP response status code
	HTTPStatus int `json:"status"`

	// DNSRCode is the response code of a DNS query in a human-readable format
	//
	// Possible values: NOERROR, FORMERR, SERVFAIL, NXDOMAIN, NOTIMP, REFUSED
	DNSRCode string `json:"-"`

	// Hostname extracted from Endpoint.URL
	Hostname string `json:"hostname,omitempty"`

	// IP resolved from the Endpoint URL
	IP string `json:"-"`

	// Connected whether a connection to the host was established successfully
	Connected bool `json:"-"`

	// Duration time that the request took
	Duration time.Duration `json:"duration"`

	// Errors encountered during the evaluation of the Endpoint's health
	Errors []string `json:"errors,omitempty"`

	// ConditionResults results of the Endpoint's conditions
	ConditionResults []*ConditionResult `json:"conditionResults"`

	// Success whether the result signifies a success or not
	Success bool `json:"success"`

	// Timestamp when the request was sent
	Timestamp time.Time `json:"timestamp"`

	// CertificateExpiration is the duration before the certificate expires
	CertificateExpiration time.Duration `json:"-"`

	// DomainExpiration is the duration before the domain expires
	DomainExpiration time.Duration `json:"-"`

	// Body is the response body
	//
	// Note that this field is not persisted in the storage.
	// It is used for health evaluation as well as debugging purposes.
	Body []byte `json:"-"`

	// Total severity counted from all ConditionResults
	Severity Severity `json:"-"`

	// Result serverity status (detrmined by Result.Severity)
	SeverityStatus SeverityStatus `json:"severity_status"`
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

// Determines severity status by highest result severity priority found.
// Returns SeverityStatus.
func (r *Result) determineSeverityStatus() SeverityStatus {
	switch severity := r.Severity; {
	case severity.Critical:
		return Critical
	case severity.High:
		return High
	case severity.Medium:
		return Medium
	case severity.Low:
		return Low
	default:
		return Critical
	}
}
