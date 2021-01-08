package storage

import (
	"encoding/json"
	"fmt"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gocache"
)

// InMemoryStore implements an in-memory store
type InMemoryStore struct {
	serviceStatuses *gocache.Cache
}

// NewInMemoryStore returns an in-memory store. Note that the store acts as a singleton, so although new-ing
// up in-memory stores will give you a unique reference to a struct each time, all structs returned
// by this function will act on the same in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		serviceStatuses: gocache.NewCache(),
	}
}

// GetAllAsJSON returns the JSON encoding of all monitored core.ServiceStatus
func (ims *InMemoryStore) GetAllAsJSON() ([]byte, error) {
	return json.Marshal(ims.serviceStatuses.GetAll())
}

// GetServiceStatus returns the service status for a given service name in the given group
func (ims *InMemoryStore) GetServiceStatus(group, name string) *core.ServiceStatus {
	key := fmt.Sprintf("%s_%s", group, name)
	serviceStatus, _ := ims.serviceStatuses.Get(key)
	if serviceStatus == nil {
		return nil
	}

	status, ok := serviceStatus.(*core.ServiceStatus)
	if !ok {
		panic("Service status was an unexpected format.")
	}
	return status
}

// Insert inserts the observed result for the specified service into the in memory store
func (ims *InMemoryStore) Insert(service *core.Service, result *core.Result) {
	key := fmt.Sprintf("%s_%s", service.Group, service.Name)
	serviceStatus, exists := ims.serviceStatuses.Get(key)
	if !exists {
		serviceStatus = core.NewServiceStatus(service)
		ims.serviceStatuses.Set(key, serviceStatus)
	}

	status, ok := serviceStatus.(*core.ServiceStatus)
	if !ok {
		panic("Service status was an unexpected format.")
	}
	status.AddResult(result)
}

// Clear will empty all the results from the in memory store
func (ims *InMemoryStore) Clear() {
	ims.serviceStatuses.Clear()
}
