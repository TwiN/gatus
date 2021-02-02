package memory

import (
	"encoding/gob"
	"encoding/json"
	"log"
	"time"

	"github.com/TwinProduction/gatus/core"
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

// GetAllAsJSON returns the JSON encoding of all monitored core.ServiceStatus
func (s *Store) GetAllAsJSON() ([]byte, error) {
	return json.Marshal(s.cache.GetAll())
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

// Clear deletes everything from the store
func (s *Store) Clear() {
	s.cache.Clear()
}

// Close closes everything which needs to be closed.
// For this provider, there's nothing that needs to be closed, but it saves on last time before closing.
func (s *Store) Close() {
	if len(s.file) > 0 {
		err := s.cache.SaveToFile(s.file)
		if err != nil {
			log.Printf("[memorywithautosave][Close] Failed to save to file=%s: %s", s.file, err.Error())
		}
	}
}

// Save persists the cache to the store file
func (s *Store) Save() error {
	return s.cache.SaveToFile(s.file)
}

// AutoSave automatically calls the Save function at every interval
func (s *Store) AutoSave(interval time.Duration) {
	for {
		log.Printf("[memorywithautosave][AutoSave] Persisting data to file")
		err := s.Save()
		if err != nil {
			log.Printf("[memorywithautosave][AutoSave] failed to save to file=%s: %s", s.file, err.Error())
		}
		time.Sleep(interval)
	}
}
