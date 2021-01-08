package storage

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gocache"
	"io/ioutil"
	"os"
	"time"
)

// Store implements an in-memory store
type Store struct {
	serviceStatuses *gocache.Cache
	interval        *time.Duration
	filePath        string
}

// NewStore returns an in-memory store. Note that the store acts as a singleton, so although new-ing
// up in-memory stores will give you a unique reference to a struct each time, all structs returned
// by this function will act on the same in-memory store.
func NewStore() *Store {
	return &Store{
		serviceStatuses: gocache.NewCache(),
	}
}

func init() {
	gob.Register(&core.Result{})
	gob.Register(&core.Uptime{})
	gob.Register(&core.ServiceStatus{})
}

// WithPersistence configures the in-memory store to read and write to a given filepath at the specified interval.
// If interval is nil then every cache write causes an immediate flush to disk.
func (ims *Store) WithPersistence(filePath string, interval *time.Duration) *Store {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err := ioutil.WriteFile(filePath, []byte{}, 0644)
		if err != nil {
			panic(fmt.Sprintf("Could not write to file: %s", err))
		}
	} else {
		numEvictions, err := ims.serviceStatuses.ReadFromFile(filePath)
		if numEvictions != 0 {
			panic(fmt.Sprintf("Unexpectedly dropped %d cache entries", numEvictions))
		}

		if err != nil {
			panic(fmt.Sprintf("Could not read from file: %s", err))
		}
	}

	ims.interval = interval
	ims.filePath = filePath

	if interval != nil {
		go flushToDisk(ims)
	}
	return ims
}

// GetAllAsJSON returns the JSON encoding of all monitored core.ServiceStatus
func (ims *Store) GetAllAsJSON() ([]byte, error) {
	return json.Marshal(ims.serviceStatuses.GetAll())
}

// GetServiceStatus returns the service status for a given service name in the given group
func (ims *Store) GetServiceStatus(group, name string) *core.ServiceStatus {
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
func (ims *Store) Insert(service *core.Service, result *core.Result) {
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

	if ims.interval == nil {
		err := ims.serviceStatuses.SaveToFile(ims.filePath)
		if err != nil {
			panic(fmt.Sprintf("Unable to save to file: %s", err))
		}
	}
}

// Clear will empty all the results from the in memory store
func (ims *Store) Clear() {
	ims.serviceStatuses.Clear()
}

func flushToDisk(store *Store) {
	for {
		err := store.serviceStatuses.SaveToFile(store.filePath)
		if err != nil {
			panic(fmt.Sprintf("failed to flush the cache to file: %s", err))
		}
		time.Sleep(*store.interval)
	}
}
