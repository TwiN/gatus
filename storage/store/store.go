package store

import (
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/memory"
)

// Store is the interface that each stores should implement
type Store interface {
	// GetAllAsJSON returns the JSON encoding of all monitored core.ServiceStatus
	GetAllAsJSON() ([]byte, error)

	// GetServiceStatus returns the service status for a given service name in the given group
	GetServiceStatus(groupName, serviceName string) *core.ServiceStatus

	// GetServiceStatusByKey returns the service status for a given key
	GetServiceStatusByKey(key string) *core.ServiceStatus

	// Insert adds the observed result for the specified service into the store
	Insert(service *core.Service, result *core.Result)

	// Clear deletes everything from the store
	Clear()

	// Close closes or stops whatever needs to be closed, if applicable.
	//
	// Note that once the store is closed, it may no longer be usable.
	Close()
}

var (
	// Validate interface implementation on compile
	_ Store = (*memory.Store)(nil)
)
