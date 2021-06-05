package core

import (
	"time"
)

// Result of the evaluation of a Service
type Result struct {
	// HTTPStatus is the HTTP response status code
	HTTPStatus int `json:"status"`

	// DNSRCode is the response code of a DNS query in a human readable format
	DNSRCode string `json:"-"`

	// Hostname extracted from Service.URL
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
	ConditionResults []*ConditionResult `json:"conditionResults"`

	// Success whether the result signifies a success or not
	Success bool `json:"success"`

	// Timestamp when the request was sent
	Timestamp time.Time `json:"timestamp"`

	// CertificateExpiration is the duration before the certificate expires
	CertificateExpiration time.Duration `json:"-"`

	// body is the response body
	//
	// Note that this variable is only used during the evaluation of a service's health.
	// This means that the call Service.EvaluateHealth both populates the body (if necessary)
	// and sets it to nil after the evaluation has been completed.
	body []byte

	// etag is from the ETag header recieved in the response
	etag string

	// previousEtag is the ETag that was received in the last request to the service.
	// Used to check whether it changed since the last request.
	previousEtag string
}
