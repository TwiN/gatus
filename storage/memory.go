package storage

import (
	"encoding/json"
	"sync"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/util"
)

// InMemoryStore implements an in-memory store
type InMemoryStore struct {
	serviceStatuses     map[string]*core.ServiceStatus
	serviceResultsMutex sync.RWMutex
}

// NewInMemoryStore returns an in-memory store. Note that the store acts as a singleton, so although new-ing
// up in-memory stores will give you a unique reference to a struct each time, all structs returned
// by this function will act on the same in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		serviceStatuses: make(map[string]*core.ServiceStatus),
	}
}

// GetAllAsJSON returns the JSON encoding of all monitored core.ServiceStatus
func (ims *InMemoryStore) GetAllAsJSON() ([]byte, error) {
	ims.serviceResultsMutex.RLock()
	serviceStatuses, err := json.Marshal(ims.serviceStatuses)
	ims.serviceResultsMutex.RUnlock()
	return serviceStatuses, err
}

// GetServiceStatus returns the service status for a given service name in the given group
func (ims *InMemoryStore) GetServiceStatus(groupName, serviceName string) *core.ServiceStatus {
	key := util.ConvertGroupAndServiceToKey(groupName, serviceName)
	ims.serviceResultsMutex.RLock()
	serviceStatus := ims.serviceStatuses[key]
	ims.serviceResultsMutex.RUnlock()
	return serviceStatus
}

// GetServiceStatusByKey returns the service status for a given key
func (ims *InMemoryStore) GetServiceStatusByKey(key string) *core.ServiceStatus {
	ims.serviceResultsMutex.RLock()
	serviceStatus := ims.serviceStatuses[key]
	ims.serviceResultsMutex.RUnlock()
	return serviceStatus
}

// Insert inserts the observed result for the specified service into the in memory store
func (ims *InMemoryStore) Insert(service *core.Service, result *core.Result) {
	key := util.ConvertGroupAndServiceToKey(service.Group, service.Name)
	ims.serviceResultsMutex.Lock()
	serviceStatus, exists := ims.serviceStatuses[key]
	if !exists {
		serviceStatus = core.NewServiceStatus(service)
		ims.serviceStatuses[key] = serviceStatus
	}
	serviceStatus.AddResult(result)
	ims.serviceResultsMutex.Unlock()
}

// Clear will empty all the results from the in memory store
func (ims *InMemoryStore) Clear() {
	ims.serviceResultsMutex.Lock()
	ims.serviceStatuses = make(map[string]*core.ServiceStatus)
	ims.serviceResultsMutex.Unlock()
}
