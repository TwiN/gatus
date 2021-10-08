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
	// GetAllServiceStatuses returns the JSON encoding of all monitored core.ServiceStatus
	// with a subset of core.Result defined by the page and pageSize parameters
	GetAllServiceStatuses(params *paging.ServiceStatusParams) ([]*core.ServiceStatus, error)

	// GetServiceStatus returns the service status for a given service name in the given group
	GetServiceStatus(groupName, serviceName string, params *paging.ServiceStatusParams) (*core.ServiceStatus, error)

	// GetServiceStatusByKey returns the service status for a given key
	GetServiceStatusByKey(key string, params *paging.ServiceStatusParams) (*core.ServiceStatus, error)

	// GetUptimeByKey returns the uptime percentage during a time range
	GetUptimeByKey(key string, from, to time.Time) (float64, error)

	// GetAverageResponseTimeByKey returns the average response time in milliseconds (value) during a time range
	GetAverageResponseTimeByKey(key string, from, to time.Time) (int, error)

	// GetHourlyAverageResponseTimeByKey returns a map of hourly (key) average response time in milliseconds (value) during a time range
	GetHourlyAverageResponseTimeByKey(key string, from, to time.Time) (map[int64]int, error)

	// Insert adds the observed result for the specified service into the store
	Insert(service *core.Service, result *core.Result) error

	// DeleteAllServiceStatusesNotInKeys removes all ServiceStatus that are not within the keys provided
	//
	// Used to delete services that have been persisted but are no longer part of the configured services
	DeleteAllServiceStatusesNotInKeys(keys []string) int

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
