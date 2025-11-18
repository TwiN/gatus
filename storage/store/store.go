package store

import (
	"context"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/gatus/v5/storage/store/memory"
	"github.com/TwiN/gatus/v5/storage/store/sql"
	"github.com/TwiN/logr"
)

// Store is the interface that each store should implement
type Store interface {
	// GetAllEndpointStatuses returns the JSON encoding of all monitored endpoint.Status
	// with a subset of endpoint.Result defined by the page and pageSize parameters
	GetAllEndpointStatuses(params *paging.EndpointStatusParams) ([]*endpoint.Status, error)

	// GetAllSuiteStatuses returns all monitored suite statuses
	GetAllSuiteStatuses(params *paging.SuiteStatusParams) ([]*suite.Status, error)

	// GetEndpointStatus returns the endpoint status for a given endpoint name in the given group
	GetEndpointStatus(groupName, endpointName string, params *paging.EndpointStatusParams) (*endpoint.Status, error)

	// GetEndpointStatusByKey returns the endpoint status for a given key
	GetEndpointStatusByKey(key string, params *paging.EndpointStatusParams) (*endpoint.Status, error)

	// GetSuiteStatusByKey returns the suite status for a given key
	GetSuiteStatusByKey(key string, params *paging.SuiteStatusParams) (*suite.Status, error)

	// GetUptimeByKey returns the uptime percentage during a time range
	GetUptimeByKey(key string, from, to time.Time) (float64, error)

	// GetAverageResponseTimeByKey returns the average response time in milliseconds (value) during a time range
	GetAverageResponseTimeByKey(key string, from, to time.Time) (int, error)

	// GetHourlyAverageResponseTimeByKey returns a map of hourly (key) average response time in milliseconds (value) during a time range
	GetHourlyAverageResponseTimeByKey(key string, from, to time.Time) (map[int64]int, error)

	// InsertEndpointResult adds the observed result for the specified endpoint into the store
	InsertEndpointResult(ep *endpoint.Endpoint, result *endpoint.Result) error

	// InsertSuiteResult adds the observed result for the specified suite into the store
	InsertSuiteResult(s *suite.Suite, result *suite.Result) error

	// DeleteAllEndpointStatusesNotInKeys removes all Status that are not within the keys provided
	//
	// Used to delete endpoints that have been persisted but are no longer part of the configured endpoints
	DeleteAllEndpointStatusesNotInKeys(keys []string) int

	// DeleteAllSuiteStatusesNotInKeys removes all suite statuses that are not within the keys provided
	DeleteAllSuiteStatusesNotInKeys(keys []string) int

	// GetTriggeredEndpointAlert returns whether the triggered alert for the specified endpoint as well as the necessary information to resolve it
	GetTriggeredEndpointAlert(ep *endpoint.Endpoint, alert *alert.Alert) (exists bool, resolveKey string, numberOfSuccessesInARow int, err error)

	// UpsertTriggeredEndpointAlert inserts/updates a triggered alert for an endpoint
	// Used for persistence of triggered alerts across application restarts
	UpsertTriggeredEndpointAlert(ep *endpoint.Endpoint, triggeredAlert *alert.Alert) error

	// DeleteTriggeredEndpointAlert deletes a triggered alert for an endpoint
	DeleteTriggeredEndpointAlert(ep *endpoint.Endpoint, triggeredAlert *alert.Alert) error

	// DeleteAllTriggeredAlertsNotInChecksumsByEndpoint removes all triggered alerts owned by an endpoint whose alert
	// configurations are not provided in the checksums list.
	// This prevents triggered alerts that have been removed or modified from lingering in the database.
	DeleteAllTriggeredAlertsNotInChecksumsByEndpoint(ep *endpoint.Endpoint, checksums []string) int

	// HasEndpointStatusNewerThan checks whether an endpoint has a status newer than the provided timestamp
	HasEndpointStatusNewerThan(key string, timestamp time.Time) (bool, error)

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
		logr.Info("[store.Get] Provider requested before it was initialized, automatically initializing")
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
		logr.Warn("[store.Initialize] nil storage config passed as parameter. This should only happen in tests. Defaulting to an empty config.")
		cfg = &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		}
	}
	if len(cfg.Path) == 0 && cfg.Type != storage.TypePostgres {
		logr.Infof("[store.Initialize] Creating storage provider of type=%s", cfg.Type)
	}
	ctx, cancelFunc = context.WithCancel(context.Background())
	switch cfg.Type {
	case storage.TypeSQLite, storage.TypePostgres:
		store, err = sql.NewStore(string(cfg.Type), cfg.Path, cfg.Caching, cfg.MaximumNumberOfResults, cfg.MaximumNumberOfEvents)
		if err != nil {
			return err
		}
	case storage.TypeMemory:
		fallthrough
	default:
		store, _ = memory.NewStore(cfg.MaximumNumberOfResults, cfg.MaximumNumberOfEvents)
	}
	return nil
}

// autoSave automatically calls the Save function of the provider at every interval
func autoSave(ctx context.Context, store Store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logr.Info("[store.autoSave] Stopping active job")
			return
		case <-ticker.C:
			logr.Info("[store.autoSave] Saving")
			if err := store.Save(); err != nil {
				logr.Errorf("[store.autoSave] Save failed: %s", err.Error())
			}
		}
	}
}
