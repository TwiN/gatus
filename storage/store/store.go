package store

import (
	"time"

	"github.com/TwiN/gatus/v3/core"
	"github.com/TwiN/gatus/v3/storage/store/common/paging"
	"github.com/TwiN/gatus/v3/storage/store/memory"
	"github.com/TwiN/gatus/v3/storage/store/sql"
)

// Store is the interface that each stores should implement
type Store interface {
	// GetAllEndpointStatuses returns the JSON encoding of all monitored core.EndpointStatus
	// with a subset of core.Result defined by the page and pageSize parameters
	GetAllEndpointStatuses(params *paging.EndpointStatusParams) ([]*core.EndpointStatus, error)

	// GetEndpointStatus returns the endpoint status for a given endpoint name in the given group
	GetEndpointStatus(groupName, endpointName string, params *paging.EndpointStatusParams) (*core.EndpointStatus, error)

	// GetEndpointStatusByKey returns the endpoint status for a given key
	GetEndpointStatusByKey(key string, params *paging.EndpointStatusParams) (*core.EndpointStatus, error)

	// GetUptimeByKey returns the uptime percentage during a time range
	GetUptimeByKey(key string, from, to time.Time) (float64, error)

	// GetAverageResponseTimeByKey returns the average response time in milliseconds (value) during a time range
	GetAverageResponseTimeByKey(key string, from, to time.Time) (int, error)

	// GetHourlyAverageResponseTimeByKey returns a map of hourly (key) average response time in milliseconds (value) during a time range
	GetHourlyAverageResponseTimeByKey(key string, from, to time.Time) (map[int64]int, error)

	// Insert adds the observed result for the specified endpoint into the store
	Insert(endpoint *core.Endpoint, result *core.Result) error

	// DeleteAllEndpointStatusesNotInKeys removes all EndpointStatus that are not within the keys provided
	//
	// Used to delete endpoints that have been persisted but are no longer part of the configured endpoints
	DeleteAllEndpointStatusesNotInKeys(keys []string) int

	// Clear deletes everything from the store
	Clear()

	// Save persists the data if and where it needs to be persisted
	Save() error

	// Close terminates every connection and closes the store, if applicable.
	// Should only be used before stopping the application.
	Close()
}

// TODO: add method to check state of store (by keeping track of silent errors)

var (
	// Validate interface implementation on compile
	_ Store = (*memory.Store)(nil)
	_ Store = (*sql.Store)(nil)
)
