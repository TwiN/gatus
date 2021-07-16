package memory

import (
	"encoding/gob"
	"sync"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/paging"
	"github.com/TwinProduction/gatus/util"
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
	sync.RWMutex
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

// GetAllServiceStatuses returns all monitored core.ServiceStatus
// with a subset of core.Result defined by the page and pageSize parameters
func (s *Store) GetAllServiceStatuses(params *paging.ServiceStatusParams) map[string]*core.ServiceStatus {
	serviceStatuses := s.cache.GetAll()
	pagedServiceStatuses := make(map[string]*core.ServiceStatus, len(serviceStatuses))
	for k, v := range serviceStatuses {
		pagedServiceStatuses[k] = ShallowCopyServiceStatus(v.(*core.ServiceStatus), params)
	}
	return pagedServiceStatuses
}

// GetServiceStatus returns the service status for a given service name in the given group
func (s *Store) GetServiceStatus(groupName, serviceName string, params *paging.ServiceStatusParams) *core.ServiceStatus {
	return s.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(groupName, serviceName), params)
}

// GetServiceStatusByKey returns the service status for a given key
func (s *Store) GetServiceStatusByKey(key string, params *paging.ServiceStatusParams) *core.ServiceStatus {
	serviceStatus := s.cache.GetValue(key)
	if serviceStatus == nil {
		return nil
	}
	return ShallowCopyServiceStatus(serviceStatus.(*core.ServiceStatus), params)
}

// Insert adds the observed result for the specified service into the store
func (s *Store) Insert(service *core.Service, result *core.Result) {
	key := service.Key()
	s.Lock()
	serviceStatus, exists := s.cache.Get(key)
	if !exists {
		serviceStatus = core.NewServiceStatus(key, service.Group, service.Name)
		serviceStatus.(*core.ServiceStatus).Events = append(serviceStatus.(*core.ServiceStatus).Events, &core.Event{
			Type:      core.EventStart,
			Timestamp: time.Now(),
		})
	}
	AddResult(serviceStatus.(*core.ServiceStatus), result)
	s.cache.Set(key, serviceStatus)
	s.Unlock()
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

// Close does nothing, because there's nothing to close
func (s *Store) Close() {
	return
}
