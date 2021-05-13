package memory

import (
	"encoding/gob"

	"github.com/Meldiron/gatus/core"
	"github.com/Meldiron/gatus/util"
	"github.com/TwinProduction/gocache"
)

func init() {
	gob.Register(&core.ServiceStatus{})
	gob.Register(&core.Uptime{})
	gob.Register(&core.Result{})
	gob.Register(&core.Event{})
}

// Store that leverages gocache
type Store struct {
	file  string
	cache *gocache.Cache
}

// NewStore creates a new store
func NewStore(file string) (*Store, error) {
	store := &Store{
		file:  file,
		cache: gocache.NewCache().WithMaxSize(gocache.NoMaxSize),
	}
	if len(file) > 0 {
		_, err := store.cache.ReadFromFile(file)
		if err != nil {
			return nil, err
		}
	}
	return store, nil
}

// GetAllServiceStatusesWithResultPagination returns all monitored core.ServiceStatus
// with a subset of core.Result defined by the page and pageSize parameters
func (s *Store) GetAllServiceStatusesWithResultPagination(page, pageSize int) map[string]*core.ServiceStatus {
	serviceStatuses := s.cache.GetAll()
	pagedServiceStatuses := make(map[string]*core.ServiceStatus, len(serviceStatuses))
	for k, v := range serviceStatuses {
		pagedServiceStatuses[k] = v.(*core.ServiceStatus).WithResultPagination(page, pageSize)
	}
	return pagedServiceStatuses
}

// GetServiceStatus returns the service status for a given service name in the given group
func (s *Store) GetServiceStatus(groupName, serviceName string) *core.ServiceStatus {
	return s.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(groupName, serviceName))
}

// GetServiceStatusByKey returns the service status for a given key
func (s *Store) GetServiceStatusByKey(key string) *core.ServiceStatus {
	serviceStatus := s.cache.GetValue(key)
	if serviceStatus == nil {
		return nil
	}
	return serviceStatus.(*core.ServiceStatus)
}

// Insert adds the observed result for the specified service into the store
func (s *Store) Insert(service *core.Service, result *core.Result) {
	key := util.ConvertGroupAndServiceToKey(service.Group, service.Name)
	serviceStatus, exists := s.cache.Get(key)
	if !exists {
		serviceStatus = core.NewServiceStatus(service)
	}
	serviceStatus.(*core.ServiceStatus).AddResult(result)
	s.cache.Set(key, serviceStatus)
}

// DeleteAllServiceStatusesNotInKeys removes all ServiceStatus that are not within the keys provided
func (s *Store) DeleteAllServiceStatusesNotInKeys(keys []string) int {
	var keysToDelete []string
	for _, existingKey := range s.cache.GetKeysByPattern("*", 0) {
		shouldDelete := true
		for _, key := range keys {
			if existingKey == key {
				shouldDelete = false
				break
			}
		}
		if shouldDelete {
			keysToDelete = append(keysToDelete, existingKey)
		}
	}
	return s.cache.DeleteAll(keysToDelete)
}

// Clear deletes everything from the store
func (s *Store) Clear() {
	s.cache.Clear()
}

// Save persists the cache to the store file
func (s *Store) Save() error {
	if len(s.file) > 0 {
		return s.cache.SaveToFile(s.file)
	}
	return nil
}
