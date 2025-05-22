package memory

import (
	"sort"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/gocache/v2"
)

// Store that leverages gocache
type Store struct {
	sync.RWMutex

	cache *gocache.Cache

	maximumNumberOfResults int // maximum number of results that an endpoint can have
	maximumNumberOfEvents  int // maximum number of events that an endpoint can have
}

// NewStore creates a new store using gocache.Cache
//
// This store holds everything in memory, and if the file parameter is not blank,
// supports eventual persistence.
func NewStore(maximumNumberOfResults, maximumNumberOfEvents int) (*Store, error) {
	store := &Store{
		cache:                  gocache.NewCache().WithMaxSize(gocache.NoMaxSize),
		maximumNumberOfResults: maximumNumberOfResults,
		maximumNumberOfEvents:  maximumNumberOfEvents,
	}
	return store, nil
}

// GetAllEndpointStatuses returns all monitored endpoint.Status
// with a subset of endpoint.Result defined by the page and pageSize parameters
func (s *Store) GetAllEndpointStatuses(params *paging.EndpointStatusParams) ([]*endpoint.Status, error) {
	endpointStatuses := s.cache.GetAll()
	pagedEndpointStatuses := make([]*endpoint.Status, 0, len(endpointStatuses))
	for _, v := range endpointStatuses {
		pagedEndpointStatuses = append(pagedEndpointStatuses, ShallowCopyEndpointStatus(v.(*endpoint.Status), params))
	}
	sort.Slice(pagedEndpointStatuses, func(i, j int) bool {
		return pagedEndpointStatuses[i].Key < pagedEndpointStatuses[j].Key
	})
	return pagedEndpointStatuses, nil
}

// GetEndpointStatus returns the endpoint status for a given endpoint name in the given group
func (s *Store) GetEndpointStatus(groupName, endpointName string, params *paging.EndpointStatusParams) (*endpoint.Status, error) {
	return s.GetEndpointStatusByKey(endpoint.ConvertGroupAndEndpointNameToKey(groupName, endpointName), params)
}

// GetEndpointStatusByKey returns the endpoint status for a given key
func (s *Store) GetEndpointStatusByKey(key string, params *paging.EndpointStatusParams) (*endpoint.Status, error) {
	endpointStatus := s.cache.GetValue(key)
	if endpointStatus == nil {
		return nil, common.ErrEndpointNotFound
	}
	return ShallowCopyEndpointStatus(endpointStatus.(*endpoint.Status), params), nil
}

// GetUptimeByKey returns the uptime percentage during a time range
func (s *Store) GetUptimeByKey(key string, from, to time.Time) (float64, error) {
	if from.After(to) {
		return 0, common.ErrInvalidTimeRange
	}
	endpointStatus := s.cache.GetValue(key)
	if endpointStatus == nil || endpointStatus.(*endpoint.Status).Uptime == nil {
		return 0, common.ErrEndpointNotFound
	}
	successfulExecutions := uint64(0)
	totalExecutions := uint64(0)
	current := from
	for to.Sub(current) >= 0 {
		hourlyUnixTimestamp := current.Truncate(time.Hour).Unix()
		hourlyStats := endpointStatus.(*endpoint.Status).Uptime.HourlyStatistics[hourlyUnixTimestamp]
		if hourlyStats == nil || hourlyStats.TotalExecutions == 0 {
			current = current.Add(time.Hour)
			continue
		}
		successfulExecutions += hourlyStats.SuccessfulExecutions
		totalExecutions += hourlyStats.TotalExecutions
		current = current.Add(time.Hour)
	}
	if totalExecutions == 0 {
		return 0, nil
	}
	return float64(successfulExecutions) / float64(totalExecutions), nil
}

// GetAverageResponseTimeByKey returns the average response time in milliseconds (value) during a time range
func (s *Store) GetAverageResponseTimeByKey(key string, from, to time.Time) (int, error) {
	if from.After(to) {
		return 0, common.ErrInvalidTimeRange
	}
	endpointStatus := s.cache.GetValue(key)
	if endpointStatus == nil || endpointStatus.(*endpoint.Status).Uptime == nil {
		return 0, common.ErrEndpointNotFound
	}
	current := from
	var totalExecutions, totalResponseTime uint64
	for to.Sub(current) >= 0 {
		hourlyUnixTimestamp := current.Truncate(time.Hour).Unix()
		hourlyStats := endpointStatus.(*endpoint.Status).Uptime.HourlyStatistics[hourlyUnixTimestamp]
		if hourlyStats == nil || hourlyStats.TotalExecutions == 0 {
			current = current.Add(time.Hour)
			continue
		}
		totalExecutions += hourlyStats.TotalExecutions
		totalResponseTime += hourlyStats.TotalExecutionsResponseTime
		current = current.Add(time.Hour)
	}
	if totalExecutions == 0 {
		return 0, nil
	}
	return int(float64(totalResponseTime) / float64(totalExecutions)), nil
}

// GetHourlyAverageResponseTimeByKey returns a map of hourly (key) average response time in milliseconds (value) during a time range
func (s *Store) GetHourlyAverageResponseTimeByKey(key string, from, to time.Time) (map[int64]int, error) {
	if from.After(to) {
		return nil, common.ErrInvalidTimeRange
	}
	endpointStatus := s.cache.GetValue(key)
	if endpointStatus == nil || endpointStatus.(*endpoint.Status).Uptime == nil {
		return nil, common.ErrEndpointNotFound
	}
	hourlyAverageResponseTimes := make(map[int64]int)
	current := from
	for to.Sub(current) >= 0 {
		hourlyUnixTimestamp := current.Truncate(time.Hour).Unix()
		hourlyStats := endpointStatus.(*endpoint.Status).Uptime.HourlyStatistics[hourlyUnixTimestamp]
		if hourlyStats == nil || hourlyStats.TotalExecutions == 0 {
			current = current.Add(time.Hour)
			continue
		}
		hourlyAverageResponseTimes[hourlyUnixTimestamp] = int(float64(hourlyStats.TotalExecutionsResponseTime) / float64(hourlyStats.TotalExecutions))
		current = current.Add(time.Hour)
	}
	return hourlyAverageResponseTimes, nil
}

// Insert adds the observed result for the specified endpoint into the store
func (s *Store) Insert(ep *endpoint.Endpoint, result *endpoint.Result) error {
	key := ep.Key()
	s.Lock()
	status, exists := s.cache.Get(key)
	if !exists {
		status = endpoint.NewStatus(ep.Group, ep.Name)
		status.(*endpoint.Status).Events = append(status.(*endpoint.Status).Events, &endpoint.Event{
			Type:      endpoint.EventStart,
			Timestamp: time.Now(),
		})
	}
	AddResult(status.(*endpoint.Status), result, s.maximumNumberOfResults, s.maximumNumberOfEvents)
	s.cache.Set(key, status)
	s.Unlock()
	return nil
}

// DeleteAllEndpointStatusesNotInKeys removes all Status that are not within the keys provided
func (s *Store) DeleteAllEndpointStatusesNotInKeys(keys []string) int {
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

// GetTriggeredEndpointAlert returns whether the triggered alert for the specified endpoint as well as the necessary information to resolve it
//
// Always returns that the alert does not exist for the in-memory store since it does not support persistence across restarts
func (s *Store) GetTriggeredEndpointAlert(ep *endpoint.Endpoint, alert *alert.Alert) (exists bool, resolveKey string, numberOfSuccessesInARow int, err error) {
	return false, "", 0, nil
}

// UpsertTriggeredEndpointAlert inserts/updates a triggered alert for an endpoint
// Used for persistence of triggered alerts across application restarts
//
// Does nothing for the in-memory store since it does not support persistence across restarts
func (s *Store) UpsertTriggeredEndpointAlert(ep *endpoint.Endpoint, triggeredAlert *alert.Alert) error {
	return nil
}

// DeleteTriggeredEndpointAlert deletes a triggered alert for an endpoint
//
// Does nothing for the in-memory store since it does not support persistence across restarts
func (s *Store) DeleteTriggeredEndpointAlert(ep *endpoint.Endpoint, triggeredAlert *alert.Alert) error {
	return nil
}

// DeleteAllTriggeredAlertsNotInChecksumsByEndpoint removes all triggered alerts owned by an endpoint whose alert
// configurations are not provided in the checksums list.
// This prevents triggered alerts that have been removed or modified from lingering in the database.
//
// Does nothing for the in-memory store since it does not support persistence across restarts
func (s *Store) DeleteAllTriggeredAlertsNotInChecksumsByEndpoint(ep *endpoint.Endpoint, checksums []string) int {
	return 0
}

// Clear deletes everything from the store
func (s *Store) Clear() {
	s.cache.Clear()
}

// Save persists the cache to the store file
func (s *Store) Save() error {
	return nil
}

// Close does nothing, because there's nothing to close
func (s *Store) Close() {
	return
}
