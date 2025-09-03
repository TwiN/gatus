package common

import "errors"

var (
	ErrEndpointNotFound = errors.New("endpoint not found")               // When an endpoint does not exist in the store
	ErrSuiteNotFound    = errors.New("suite not found")                  // When a suite does not exist in the store
	ErrInvalidTimeRange = errors.New("'from' cannot be older than 'to'") // When an invalid time range is provided
)
