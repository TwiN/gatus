package common

import "errors"

var (
	ErrServiceNotFound  = errors.New("service not found")                // When a service does not exist in the store
	ErrInvalidTimeRange = errors.New("'from' cannot be older than 'to'") // When an invalid time range is provided
)
