package memory

import (
	"encoding/gob"
	"io/fs"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TwiN/gatus/v3/core"
	"github.com/TwiN/gatus/v3/storage/store/common"
	"github.com/TwiN/gatus/v3/storage/store/common/paging"
	"github.com/TwiN/gatus/v3/util"
	"github.com/TwiN/gocache"
)

func init() {
	gob.Register(&core.EndpointStatus{})
	gob.Register(&core.HourlyUptimeStatistics{})
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

// NewStore creates a new store using gocache.Cache
//
// This store holds everything in memory, and if the file parameter is not blank,
// supports eventual persistence.
func NewStore(file string) (*Store, error) {
	store := &Store{
		file:  file,
		cache: gocache.NewCache().WithMaxSize(gocache.NoMaxSize),
	}
	if len(file) > 0 {
		_, err := store.cache.ReadFromFile(file)
		if err != nil {
			// XXX: Remove the block below in v4.0.0
			if data, err2 := ioutil.ReadFile(file); err2 == nil {
				isFromOldVersion := strings.Contains(string(data), "*core.ServiceStatus")
				if isFromOldVersion {
					log.Println("WARNING: Couldn't read file due to recent change in v3.3.0, see https://github.com/TwiN/gatus/issues/191")
					log.Println("WARNING: Will automatically rename old file to " + file + ".old and overwrite the current file")
					if err = ioutil.WriteFile(file+".old", data, fs.ModePerm); err != nil {
						log.Println("WARNING: Tried my best to keep the old file, but it wasn't enough. Sorry, your file will be overwritten :(")
					}
					// Return the store regardless of whether there was an error or not
					return store, nil
				}
			}
			// XXX: Remove the block above in v4.0.0
			return nil, err
		}
	}
	return store, nil
}

// GetAllEndpointStatuses returns all monitored core.EndpointStatus
// with a subset of core.Result defined by the page and pageSize parameters
func (s *Store) GetAllEndpointStatuses(params *paging.EndpointStatusParams) ([]*core.EndpointStatus, error) {
	endpointStatuses := s.cache.GetAll()
	pagedEndpointStatuses := make([]*core.EndpointStatus, 0, len(endpointStatuses))
	for _, v := range endpointStatuses {
		pagedEndpointStatuses = append(pagedEndpointStatuses, ShallowCopyEndpointStatus(v.(*core.EndpointStatus), params))
	}
	sort.Slice(pagedEndpointStatuses, func(i, j int) bool {
		return pagedEndpointStatuses[i].Key < pagedEndpointStatuses[j].Key
	})
	return pagedEndpointStatuses, nil
}

// GetEndpointStatus returns the endpoint status for a given endpoint name in the given group
func (s *Store) GetEndpointStatus(groupName, endpointName string, params *paging.EndpointStatusParams) (*core.EndpointStatus, error) {
	return s.GetEndpointStatusByKey(util.ConvertGroupAndEndpointNameToKey(groupName, endpointName), params)
}

// GetEndpointStatusByKey returns the endpoint status for a given key
func (s *Store) GetEndpointStatusByKey(key string, params *paging.EndpointStatusParams) (*core.EndpointStatus, error) {
	endpointStatus := s.cache.GetValue(key)
	if endpointStatus == nil {
		return nil, common.ErrEndpointNotFound
	}
	return ShallowCopyEndpointStatus(endpointStatus.(*core.EndpointStatus), params), nil
}

// GetUptimeByKey returns the uptime percentage during a time range
func (s *Store) GetUptimeByKey(key string, from, to time.Time) (float64, error) {
	if from.After(to) {
		return 0, common.ErrInvalidTimeRange
	}
	endpointStatus := s.cache.GetValue(key)
	if endpointStatus == nil || endpointStatus.(*core.EndpointStatus).Uptime == nil {
		return 0, common.ErrEndpointNotFound
	}
	successfulExecutions := uint64(0)
	totalExecutions := uint64(0)
	current := from
	for to.Sub(current) >= 0 {
		hourlyUnixTimestamp := current.Truncate(time.Hour).Unix()
		hourlyStats := endpointStatus.(*core.EndpointStatus).Uptime.HourlyStatistics[hourlyUnixTimestamp]
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
	if endpointStatus == nil || endpointStatus.(*core.EndpointStatus).Uptime == nil {
		return 0, common.ErrEndpointNotFound
	}
	current := from
	var totalExecutions, totalResponseTime uint64
	for to.Sub(current) >= 0 {
		hourlyUnixTimestamp := current.Truncate(time.Hour).Unix()
		hourlyStats := endpointStatus.(*core.EndpointStatus).Uptime.HourlyStatistics[hourlyUnixTimestamp]
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
	if endpointStatus == nil || endpointStatus.(*core.EndpointStatus).Uptime == nil {
		return nil, common.ErrEndpointNotFound
	}
	hourlyAverageResponseTimes := make(map[int64]int)
	current := from
	for to.Sub(current) >= 0 {
		hourlyUnixTimestamp := current.Truncate(time.Hour).Unix()
		hourlyStats := endpointStatus.(*core.EndpointStatus).Uptime.HourlyStatistics[hourlyUnixTimestamp]
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
func (s *Store) Insert(endpoint *core.Endpoint, result *core.Result) error {
	key := endpoint.Key()
	s.Lock()
	status, exists := s.cache.Get(key)
	if !exists {
		status = core.NewEndpointStatus(endpoint.Group, endpoint.Name)
		status.(*core.EndpointStatus).Events = append(status.(*core.EndpointStatus).Events, &core.Event{
			Type:      core.EventStart,
			Timestamp: time.Now(),
		})
	}
	AddResult(status.(*core.EndpointStatus), result)
	s.cache.Set(key, status)
	s.Unlock()
	return nil
}

// DeleteAllEndpointStatusesNotInKeys removes all EndpointStatus that are not within the keys provided
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
