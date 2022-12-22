package store

import (
	"context"
	"log"
	"time"

	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/gatus/v5/storage/store/memory"
	"github.com/TwiN/gatus/v5/storage/store/sql"
)

// Store is the interface that each store should implement
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

var (
	store Store

	// initialized keeps track of whether the storage provider was initialized
	// Because store.Store is an interface, a nil check wouldn't be sufficient, so instead of doing reflection
	// every single time Get is called, we'll just lazily keep track of its existence through this variable
	initialized bool

	ctx        context.Context
	cancelFunc context.CancelFunc
)

func Get() Store {
	if !initialized {
		// This only happens in tests
		log.Println("[store][Get] Provider requested before it was initialized, automatically initializing")
		err := Initialize(nil)
		if err != nil {
			panic("failed to automatically initialize store: " + err.Error())
		}
	}
	return store
}

// Initialize instantiates the storage provider based on the Config provider
func Initialize(cfg *storage.Config) error {
	initialized = true
	var err error
	if cancelFunc != nil {
		// Stop the active autoSave task, if there's already one
		cancelFunc()
	}
	if cfg == nil {
		// This only happens in tests
		log.Println("[store][Initialize] nil storage config passed as parameter. This should only happen in tests. Defaulting to an empty config.")
		cfg = &storage.Config{}
	}
	if len(cfg.Path) == 0 && cfg.Type != storage.TypePostgres {
		log.Printf("[store][Initialize] Creating storage provider of type=%s", cfg.Type)
	}
	ctx, cancelFunc = context.WithCancel(context.Background())
	switch cfg.Type {
	case storage.TypeSQLite, storage.TypePostgres:
		store, err = sql.NewStore(string(cfg.Type), cfg.Path, cfg.Caching)
		if err != nil {
			return err
		}
	case storage.TypeMemory:
		fallthrough
	default:
		store, _ = memory.NewStore()
	}
	return nil
}

// autoSave automatically calls the Save function of the provider at every interval
func autoSave(ctx context.Context, store Store, interval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[store][autoSave] Stopping active job")
			return
		case <-time.After(interval):
			log.Printf("[store][autoSave] Saving")
			err := store.Save()
			if err != nil {
				log.Println("[store][autoSave] Save failed:", err.Error())
			}
		}
	}
}
