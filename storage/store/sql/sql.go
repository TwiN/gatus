package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/gatus/v5/util"
	"github.com/TwiN/gocache/v2"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// Note that only exported functions in this file may create, commit, or rollback a transaction //
//////////////////////////////////////////////////////////////////////////////////////////////////

const (
	// arraySeparator is the separator used to separate multiple strings in a single column.
	// It's a dirty hack, but it's only used for persisting errors, and since this data will likely only ever be used
	// for aesthetic purposes, I deemed it wasn't worth the performance impact of yet another one-to-many table.
	arraySeparator = "|~|"

	uptimeCleanUpThreshold = 10 * 24 * time.Hour // Maximum uptime age before triggering a clean up

	uptimeRetention = 7 * 24 * time.Hour

	cacheTTL = 10 * time.Minute

	eventsAboveMaximumCleanUpThreshold = 10

	resultsAboveMaximumCleanUpThreshold = 10
)

var (
	// ErrPathNotSpecified is the error returned when the path parameter passed in NewStore is blank
	ErrPathNotSpecified = errors.New("path cannot be empty")

	// ErrDatabaseDriverNotSpecified is the error returned when the driver parameter passed in NewStore is blank
	ErrDatabaseDriverNotSpecified = errors.New("database driver cannot be empty")

	errNoRowsReturned = errors.New("expected a row to be returned, but none was")
)

// Store that leverages a database
type Store struct {
	driver, path string

	db *sql.DB

	// writeThroughCache is a cache used to drastically decrease read latency by pre-emptively
	// caching writes as they happen. If nil, writes are not cached.
	writeThroughCache *gocache.Cache

	// maximumNumberOfResults is the maximum number of results that an endpoint can have
	maximumNumberOfResults int
	// maximumNumberOfEvents is the maximum number of events that an endpoint can have
	maximumNumberOfEvents int
}

// NewStore initializes the database and creates the schema if it doesn't already exist in the path specified
func NewStore(driver, path string, caching bool, maximumNumberOfResults int, maximumNumberOfEvents int) (*Store, error) {
	if len(driver) == 0 {
		return nil, ErrDatabaseDriverNotSpecified
	}
	if len(path) == 0 {
		return nil, ErrPathNotSpecified
	}
	store := &Store{
		driver:                 driver,
		path:                   path,
		maximumNumberOfResults: maximumNumberOfResults,
		maximumNumberOfEvents:  maximumNumberOfEvents,
	}
	var err error
	if store.db, err = sql.Open(driver, path); err != nil {
		return nil, err
	}
	if err := store.db.Ping(); err != nil {
		return nil, err
	}
	if driver == "sqlite" {
		_, _ = store.db.Exec("PRAGMA foreign_keys=ON")
		_, _ = store.db.Exec("PRAGMA journal_mode=WAL")
		_, _ = store.db.Exec("PRAGMA synchronous=NORMAL")
		// Prevents driver from running into "database is locked" errors
		// This is because we're using WAL to improve performance
		store.db.SetMaxOpenConns(1)
	}
	if err = store.createSchema(); err != nil {
		_ = store.db.Close()
		return nil, err
	}
	if caching {
		store.writeThroughCache = gocache.NewCache().WithMaxSize(10000)
	}
	return store, nil
}

// createSchema creates the schema required to perform all database operations.
func (s *Store) createSchema() error {
	if s.driver == "sqlite" {
		return s.createSQLiteSchema()
	}
	return s.createPostgresSchema()
}

// GetAllEndpointStatuses returns all monitored core.EndpointStatus
// with a subset of core.Result defined by the page and pageSize parameters
func (s *Store) GetAllEndpointStatuses(params *paging.EndpointStatusParams) ([]*core.EndpointStatus, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	keys, err := s.getAllEndpointKeys(tx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	endpointStatuses := make([]*core.EndpointStatus, 0, len(keys))
	for _, key := range keys {
		endpointStatus, err := s.getEndpointStatusByKey(tx, key, params)
		if err != nil {
			continue
		}
		endpointStatuses = append(endpointStatuses, endpointStatus)
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return endpointStatuses, err
}

// GetEndpointStatus returns the endpoint status for a given endpoint name in the given group
func (s *Store) GetEndpointStatus(groupName, endpointName string, params *paging.EndpointStatusParams) (*core.EndpointStatus, error) {
	return s.GetEndpointStatusByKey(util.ConvertGroupAndEndpointNameToKey(groupName, endpointName), params)
}

// GetEndpointStatusByKey returns the endpoint status for a given key
func (s *Store) GetEndpointStatusByKey(key string, params *paging.EndpointStatusParams) (*core.EndpointStatus, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	endpointStatus, err := s.getEndpointStatusByKey(tx, key, params)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return endpointStatus, err
}

// GetUptimeByKey returns the uptime percentage during a time range
func (s *Store) GetUptimeByKey(key string, from, to time.Time) (float64, error) {
	if from.After(to) {
		return 0, common.ErrInvalidTimeRange
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	endpointID, _, _, err := s.getEndpointIDGroupAndNameByKey(tx, key)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	uptime, _, err := s.getEndpointUptime(tx, endpointID, from, to)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return uptime, nil
}

// GetAverageResponseTimeByKey returns the average response time in milliseconds (value) during a time range
func (s *Store) GetAverageResponseTimeByKey(key string, from, to time.Time) (int, error) {
	if from.After(to) {
		return 0, common.ErrInvalidTimeRange
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	endpointID, _, _, err := s.getEndpointIDGroupAndNameByKey(tx, key)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	averageResponseTime, err := s.getEndpointAverageResponseTime(tx, endpointID, from, to)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return averageResponseTime, nil
}

// GetHourlyAverageResponseTimeByKey returns a map of hourly (key) average response time in milliseconds (value) during a time range
func (s *Store) GetHourlyAverageResponseTimeByKey(key string, from, to time.Time) (map[int64]int, error) {
	if from.After(to) {
		return nil, common.ErrInvalidTimeRange
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	endpointID, _, _, err := s.getEndpointIDGroupAndNameByKey(tx, key)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	hourlyAverageResponseTimes, err := s.getEndpointHourlyAverageResponseTimes(tx, endpointID, from, to)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return hourlyAverageResponseTimes, nil
}

// Insert adds the observed result for the specified endpoint into the store
func (s *Store) Insert(endpoint *core.Endpoint, result *core.Result) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	endpointID, err := s.getEndpointID(tx, endpoint)
	if err != nil {
		if err == common.ErrEndpointNotFound {
			// Endpoint doesn't exist in the database, insert it
			if endpointID, err = s.insertEndpoint(tx, endpoint); err != nil {
				_ = tx.Rollback()
				log.Printf("[sql][Insert] Failed to create endpoint with group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
				return err
			}
		} else {
			_ = tx.Rollback()
			log.Printf("[sql][Insert] Failed to retrieve id of endpoint with group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
			return err
		}
	}
	// First, we need to check if we need to insert a new event.
	//
	// A new event must be added if either of the following cases happen:
	// 1. There is only 1 event. The total number of events for a endpoint can only be 1 if the only existing event is
	//    of type EventStart, in which case we will have to create a new event of type EventHealthy or EventUnhealthy
	//    based on result.Success.
	// 2. The lastResult.Success != result.Success. This implies that the endpoint went from healthy to unhealthy or
	//    vice-versa, in which case we will have to create a new event of type EventHealthy or EventUnhealthy
	//	  based on result.Success.
	numberOfEvents, err := s.getNumberOfEventsByEndpointID(tx, endpointID)
	if err != nil {
		// Silently fail
		log.Printf("[sql][Insert] Failed to retrieve total number of events for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
	}
	if numberOfEvents == 0 {
		// There's no events yet, which means we need to add the EventStart and the first healthy/unhealthy event
		err = s.insertEndpointEvent(tx, endpointID, &core.Event{
			Type:      core.EventStart,
			Timestamp: result.Timestamp.Add(-50 * time.Millisecond),
		})
		if err != nil {
			// Silently fail
			log.Printf("[sql][Insert] Failed to insert event=%s for group=%s; endpoint=%s: %s", core.EventStart, endpoint.Group, endpoint.Name, err.Error())
		}
		event := core.NewEventFromResult(result)
		if err = s.insertEndpointEvent(tx, endpointID, event); err != nil {
			// Silently fail
			log.Printf("[sql][Insert] Failed to insert event=%s for group=%s; endpoint=%s: %s", event.Type, endpoint.Group, endpoint.Name, err.Error())
		}
	} else {
		// Get the success value of the previous result
		var lastResultSuccess bool
		if lastResultSuccess, err = s.getLastEndpointResultSuccessValue(tx, endpointID); err != nil {
			log.Printf("[sql][Insert] Failed to retrieve outcome of previous result for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
		} else {
			// If we managed to retrieve the outcome of the previous result, we'll compare it with the new result.
			// If the final outcome (success or failure) of the previous and the new result aren't the same, it means
			// that the endpoint either went from Healthy to Unhealthy or Unhealthy -> Healthy, therefore, we'll add
			// an event to mark the change in state
			if lastResultSuccess != result.Success {
				event := core.NewEventFromResult(result)
				if err = s.insertEndpointEvent(tx, endpointID, event); err != nil {
					// Silently fail
					log.Printf("[sql][Insert] Failed to insert event=%s for group=%s; endpoint=%s: %s", event.Type, endpoint.Group, endpoint.Name, err.Error())
				}
			}
		}
		// Clean up old events if there's more than twice the maximum number of events
		// This lets us both keep the table clean without impacting performance too much
		// (since we're only deleting MaximumNumberOfEvents at a time instead of 1)
		if numberOfEvents > int64(s.maximumNumberOfEvents+eventsAboveMaximumCleanUpThreshold) {
			if err = s.deleteOldEndpointEvents(tx, endpointID); err != nil {
				log.Printf("[sql][Insert] Failed to delete old events for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
			}
		}
	}
	// Second, we need to insert the result.
	if err = s.insertEndpointResult(tx, endpointID, result); err != nil {
		log.Printf("[sql][Insert] Failed to insert result for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
		_ = tx.Rollback() // If we can't insert the result, we'll rollback now since there's no point continuing
		return err
	}
	// Clean up old results
	numberOfResults, err := s.getNumberOfResultsByEndpointID(tx, endpointID)
	if err != nil {
		log.Printf("[sql][Insert] Failed to retrieve total number of results for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
	} else {
		if numberOfResults > int64(s.maximumNumberOfResults+resultsAboveMaximumCleanUpThreshold) {
			if err = s.deleteOldEndpointResults(tx, endpointID); err != nil {
				log.Printf("[sql][Insert] Failed to delete old results for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
			}
		}
	}
	// Finally, we need to insert the uptime data.
	// Because the uptime data significantly outlives the results, we can't rely on the results for determining the uptime
	if err = s.updateEndpointUptime(tx, endpointID, result); err != nil {
		log.Printf("[sql][Insert] Failed to update uptime for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
	}
	// Clean up old uptime entries
	ageOfOldestUptimeEntry, err := s.getAgeOfOldestEndpointUptimeEntry(tx, endpointID)
	if err != nil {
		log.Printf("[sql][Insert] Failed to retrieve oldest endpoint uptime entry for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
	} else {
		if ageOfOldestUptimeEntry > uptimeCleanUpThreshold {
			if err = s.deleteOldUptimeEntries(tx, endpointID, time.Now().Add(-(uptimeRetention + time.Hour))); err != nil {
				log.Printf("[sql][Insert] Failed to delete old uptime entries for group=%s; endpoint=%s: %s", endpoint.Group, endpoint.Name, err.Error())
			}
		}
	}
	if s.writeThroughCache != nil {
		cacheKeysToRefresh := s.writeThroughCache.GetKeysByPattern(endpoint.Key()+"*", 0)
		for _, cacheKey := range cacheKeysToRefresh {
			s.writeThroughCache.Delete(cacheKey)
			endpointKey, params, err := extractKeyAndParamsFromCacheKey(cacheKey)
			if err != nil {
				log.Printf("[sql][Insert] Silently deleting cache key %s instead of refreshing due to error: %s", cacheKey, err.Error())
				continue
			}
			// Retrieve the endpoint status by key, which will in turn refresh the cache
			_, _ = s.getEndpointStatusByKey(tx, endpointKey, params)
		}
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return err
}

// DeleteAllEndpointStatusesNotInKeys removes all rows owned by an endpoint whose key is not within the keys provided
func (s *Store) DeleteAllEndpointStatusesNotInKeys(keys []string) int {
	var err error
	var result sql.Result
	if len(keys) == 0 {
		// Delete everything
		result, err = s.db.Exec("DELETE FROM endpoints")
	} else {
		args := make([]interface{}, 0, len(keys))
		query := "DELETE FROM endpoints WHERE endpoint_key NOT IN ("
		for i := range keys {
			query += fmt.Sprintf("$%d,", i+1)
			args = append(args, keys[i])
		}
		query = query[:len(query)-1] + ")" // Remove the last comma and add the closing parenthesis
		result, err = s.db.Exec(query, args...)
	}
	if err != nil {
		log.Printf("[sql][DeleteAllEndpointStatusesNotInKeys] Failed to delete rows that do not belong to any of keys=%v: %s", keys, err.Error())
		return 0
	}
	if s.writeThroughCache != nil {
		// It's easier to just wipe out the entire cache than to try to find all keys that are not in the keys list
		_ = s.writeThroughCache.DeleteKeysByPattern("*")
	}
	// Return number of rows deleted
	rowsAffects, _ := result.RowsAffected()
	return int(rowsAffects)
}

// Clear deletes everything from the store
func (s *Store) Clear() {
	_, _ = s.db.Exec("DELETE FROM endpoints")
	if s.writeThroughCache != nil {
		_ = s.writeThroughCache.DeleteKeysByPattern("*")
	}
}

// Save does nothing, because this store is immediately persistent.
func (s *Store) Save() error {
	return nil
}

// Close the database handle
func (s *Store) Close() {
	_ = s.db.Close()
	if s.writeThroughCache != nil {
		// Clear the cache too. If the store's been closed, we don't want to keep the cache around.
		_ = s.writeThroughCache.DeleteKeysByPattern("*")
	}
}

// insertEndpoint inserts an endpoint in the store and returns the generated id of said endpoint
func (s *Store) insertEndpoint(tx *sql.Tx, endpoint *core.Endpoint) (int64, error) {
	//log.Printf("[sql][insertEndpoint] Inserting endpoint with group=%s and name=%s", endpoint.Group, endpoint.Name)
	var id int64
	err := tx.QueryRow(
		"INSERT INTO endpoints (endpoint_key, endpoint_name, endpoint_group) VALUES ($1, $2, $3) RETURNING endpoint_id",
		endpoint.Key(),
		endpoint.Name,
		endpoint.Group,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// insertEndpointEvent inserts en event in the store
func (s *Store) insertEndpointEvent(tx *sql.Tx, endpointID int64, event *core.Event) error {
	_, err := tx.Exec(
		"INSERT INTO endpoint_events (endpoint_id, event_type, event_timestamp) VALUES ($1, $2, $3)",
		endpointID,
		event.Type,
		event.Timestamp.UTC(),
	)
	if err != nil {
		return err
	}
	return nil
}

// insertEndpointResult inserts a result in the store
func (s *Store) insertEndpointResult(tx *sql.Tx, endpointID int64, result *core.Result) error {
	var endpointResultID int64
	err := tx.QueryRow(
		`
			INSERT INTO endpoint_results (endpoint_id, success, errors, connected, status, dns_rcode, certificate_expiration, domain_expiration, hostname, ip, duration, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING endpoint_result_id
		`,
		endpointID,
		result.Success,
		strings.Join(result.Errors, arraySeparator),
		result.Connected,
		result.HTTPStatus,
		result.DNSRCode,
		result.CertificateExpiration,
		result.DomainExpiration,
		result.Hostname,
		result.IP,
		result.Duration,
		result.Timestamp.UTC(),
	).Scan(&endpointResultID)
	if err != nil {
		return err
	}
	return s.insertConditionResults(tx, endpointResultID, result.ConditionResults)
}

func (s *Store) insertConditionResults(tx *sql.Tx, endpointResultID int64, conditionResults []*core.ConditionResult) error {
	var err error
	for _, cr := range conditionResults {
		_, err = tx.Exec("INSERT INTO endpoint_result_conditions (endpoint_result_id, condition, success) VALUES ($1, $2, $3)",
			endpointResultID,
			cr.Condition,
			cr.Success,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) updateEndpointUptime(tx *sql.Tx, endpointID int64, result *core.Result) error {
	unixTimestampFlooredAtHour := result.Timestamp.Truncate(time.Hour).Unix()
	var successfulExecutions int
	if result.Success {
		successfulExecutions = 1
	}
	_, err := tx.Exec(
		`
			INSERT INTO endpoint_uptimes (endpoint_id, hour_unix_timestamp, total_executions, successful_executions, total_response_time) 
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT(endpoint_id, hour_unix_timestamp) DO UPDATE SET
				total_executions = excluded.total_executions + endpoint_uptimes.total_executions,
				successful_executions = excluded.successful_executions + endpoint_uptimes.successful_executions,
				total_response_time = excluded.total_response_time + endpoint_uptimes.total_response_time
		`,
		endpointID,
		unixTimestampFlooredAtHour,
		1,
		successfulExecutions,
		result.Duration.Milliseconds(),
	)
	return err
}

func (s *Store) getAllEndpointKeys(tx *sql.Tx) (keys []string, err error) {
	rows, err := tx.Query("SELECT endpoint_key FROM endpoints ORDER BY endpoint_key")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key string
		_ = rows.Scan(&key)
		keys = append(keys, key)
	}
	return
}

func (s *Store) getEndpointStatusByKey(tx *sql.Tx, key string, parameters *paging.EndpointStatusParams) (*core.EndpointStatus, error) {
	var cacheKey string
	if s.writeThroughCache != nil {
		cacheKey = generateCacheKey(key, parameters)
		if cachedEndpointStatus, exists := s.writeThroughCache.Get(cacheKey); exists {
			if castedCachedEndpointStatus, ok := cachedEndpointStatus.(*core.EndpointStatus); ok {
				return castedCachedEndpointStatus, nil
			}
		}
	}
	endpointID, group, endpointName, err := s.getEndpointIDGroupAndNameByKey(tx, key)
	if err != nil {
		return nil, err
	}
	endpointStatus := core.NewEndpointStatus(group, endpointName)
	if parameters.EventsPageSize > 0 {
		if endpointStatus.Events, err = s.getEndpointEventsByEndpointID(tx, endpointID, parameters.EventsPage, parameters.EventsPageSize); err != nil {
			log.Printf("[sql][getEndpointStatusByKey] Failed to retrieve events for key=%s: %s", key, err.Error())
		}
	}
	if parameters.ResultsPageSize > 0 {
		if endpointStatus.Results, err = s.getEndpointResultsByEndpointID(tx, endpointID, parameters.ResultsPage, parameters.ResultsPageSize); err != nil {
			log.Printf("[sql][getEndpointStatusByKey] Failed to retrieve results for key=%s: %s", key, err.Error())
		}
	}
	if s.writeThroughCache != nil {
		s.writeThroughCache.SetWithTTL(cacheKey, endpointStatus, cacheTTL)
	}
	return endpointStatus, nil
}

func (s *Store) getEndpointIDGroupAndNameByKey(tx *sql.Tx, key string) (id int64, group, name string, err error) {
	err = tx.QueryRow(
		`
			SELECT endpoint_id, endpoint_group, endpoint_name
			FROM endpoints
			WHERE endpoint_key = $1
			LIMIT 1
		`,
		key,
	).Scan(&id, &group, &name)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", "", common.ErrEndpointNotFound
		}
		return 0, "", "", err
	}
	return
}

func (s *Store) getEndpointEventsByEndpointID(tx *sql.Tx, endpointID int64, page, pageSize int) (events []*core.Event, err error) {
	rows, err := tx.Query(
		`
			SELECT event_type, event_timestamp
			FROM endpoint_events
			WHERE endpoint_id = $1
			ORDER BY endpoint_event_id ASC
			LIMIT $2 OFFSET $3
		`,
		endpointID,
		pageSize,
		(page-1)*pageSize,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		event := &core.Event{}
		_ = rows.Scan(&event.Type, &event.Timestamp)
		events = append(events, event)
	}
	return
}

func (s *Store) getEndpointResultsByEndpointID(tx *sql.Tx, endpointID int64, page, pageSize int) (results []*core.Result, err error) {
	rows, err := tx.Query(
		`
			SELECT endpoint_result_id, success, errors, connected, status, dns_rcode, certificate_expiration, domain_expiration, hostname, ip, duration, timestamp
			FROM endpoint_results
			WHERE endpoint_id = $1
			ORDER BY endpoint_result_id DESC -- Normally, we'd sort by timestamp, but sorting by endpoint_result_id is faster
			LIMIT $2 OFFSET $3
		`,
		endpointID,
		pageSize,
		(page-1)*pageSize,
	)
	if err != nil {
		return nil, err
	}
	idResultMap := make(map[int64]*core.Result)
	for rows.Next() {
		result := &core.Result{}
		var id int64
		var joinedErrors string
		err = rows.Scan(&id, &result.Success, &joinedErrors, &result.Connected, &result.HTTPStatus, &result.DNSRCode, &result.CertificateExpiration, &result.DomainExpiration, &result.Hostname, &result.IP, &result.Duration, &result.Timestamp)
		if err != nil {
			log.Printf("[sql][getEndpointResultsByEndpointID] Silently failed to retrieve endpoint result for endpointID=%d: %s", endpointID, err.Error())
			err = nil
		}
		if len(joinedErrors) != 0 {
			result.Errors = strings.Split(joinedErrors, arraySeparator)
		}
		// This is faster than using a subselect
		results = append([]*core.Result{result}, results...)
		idResultMap[id] = result
	}
	if len(idResultMap) == 0 {
		// If there's no result, we'll just return an empty/nil slice
		return
	}
	// Get condition results
	args := make([]interface{}, 0, len(idResultMap))
	query := `SELECT endpoint_result_id, condition, success
				FROM endpoint_result_conditions
				WHERE endpoint_result_id IN (`
	index := 1
	for endpointResultID := range idResultMap {
		query += "$" + strconv.Itoa(index) + ","
		args = append(args, endpointResultID)
		index++
	}
	query = query[:len(query)-1] + ")"
	rows, err = tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // explicitly defer the close in case an error happens during the scan
	for rows.Next() {
		conditionResult := &core.ConditionResult{}
		var endpointResultID int64
		if err = rows.Scan(&endpointResultID, &conditionResult.Condition, &conditionResult.Success); err != nil {
			return
		}
		idResultMap[endpointResultID].ConditionResults = append(idResultMap[endpointResultID].ConditionResults, conditionResult)
	}
	return
}

func (s *Store) getEndpointUptime(tx *sql.Tx, endpointID int64, from, to time.Time) (uptime float64, avgResponseTime time.Duration, err error) {
	rows, err := tx.Query(
		`
			SELECT SUM(total_executions), SUM(successful_executions), SUM(total_response_time)
			FROM endpoint_uptimes
			WHERE endpoint_id = $1
				AND hour_unix_timestamp >= $2
				AND hour_unix_timestamp <= $3
		`,
		endpointID,
		from.Unix(),
		to.Unix(),
	)
	if err != nil {
		return 0, 0, err
	}
	var totalExecutions, totalSuccessfulExecutions, totalResponseTime int
	for rows.Next() {
		_ = rows.Scan(&totalExecutions, &totalSuccessfulExecutions, &totalResponseTime)
	}
	if totalExecutions > 0 {
		uptime = float64(totalSuccessfulExecutions) / float64(totalExecutions)
		avgResponseTime = time.Duration(float64(totalResponseTime)/float64(totalExecutions)) * time.Millisecond
	}
	return
}

func (s *Store) getEndpointAverageResponseTime(tx *sql.Tx, endpointID int64, from, to time.Time) (int, error) {
	rows, err := tx.Query(
		`
			SELECT SUM(total_executions), SUM(total_response_time)
			FROM endpoint_uptimes
			WHERE endpoint_id = $1
				AND total_executions > 0
				AND hour_unix_timestamp >= $2
				AND hour_unix_timestamp <= $3
		`,
		endpointID,
		from.Unix(),
		to.Unix(),
	)
	if err != nil {
		return 0, err
	}
	var totalExecutions, totalResponseTime int
	for rows.Next() {
		_ = rows.Scan(&totalExecutions, &totalResponseTime)
	}
	if totalExecutions == 0 {
		return 0, nil
	}
	return int(float64(totalResponseTime) / float64(totalExecutions)), nil
}

func (s *Store) getEndpointHourlyAverageResponseTimes(tx *sql.Tx, endpointID int64, from, to time.Time) (map[int64]int, error) {
	rows, err := tx.Query(
		`
			SELECT hour_unix_timestamp, total_executions, total_response_time
			FROM endpoint_uptimes
			WHERE endpoint_id = $1
				AND total_executions > 0
				AND hour_unix_timestamp >= $2
				AND hour_unix_timestamp <= $3
		`,
		endpointID,
		from.Unix(),
		to.Unix(),
	)
	if err != nil {
		return nil, err
	}
	var totalExecutions, totalResponseTime int
	var unixTimestampFlooredAtHour int64
	hourlyAverageResponseTimes := make(map[int64]int)
	for rows.Next() {
		_ = rows.Scan(&unixTimestampFlooredAtHour, &totalExecutions, &totalResponseTime)
		hourlyAverageResponseTimes[unixTimestampFlooredAtHour] = int(float64(totalResponseTime) / float64(totalExecutions))
	}
	return hourlyAverageResponseTimes, nil
}

func (s *Store) getEndpointID(tx *sql.Tx, endpoint *core.Endpoint) (int64, error) {
	var id int64
	err := tx.QueryRow("SELECT endpoint_id FROM endpoints WHERE endpoint_key = $1", endpoint.Key()).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, common.ErrEndpointNotFound
		}
		return 0, err
	}
	return id, nil
}

func (s *Store) getNumberOfEventsByEndpointID(tx *sql.Tx, endpointID int64) (int64, error) {
	var numberOfEvents int64
	err := tx.QueryRow("SELECT COUNT(1) FROM endpoint_events WHERE endpoint_id = $1", endpointID).Scan(&numberOfEvents)
	return numberOfEvents, err
}

func (s *Store) getNumberOfResultsByEndpointID(tx *sql.Tx, endpointID int64) (int64, error) {
	var numberOfResults int64
	err := tx.QueryRow("SELECT COUNT(1) FROM endpoint_results WHERE endpoint_id = $1", endpointID).Scan(&numberOfResults)
	return numberOfResults, err
}

func (s *Store) getAgeOfOldestEndpointUptimeEntry(tx *sql.Tx, endpointID int64) (time.Duration, error) {
	rows, err := tx.Query(
		`
			SELECT hour_unix_timestamp 
			FROM endpoint_uptimes 
			WHERE endpoint_id = $1 
			ORDER BY hour_unix_timestamp
			LIMIT 1
		`,
		endpointID,
	)
	if err != nil {
		return 0, err
	}
	var oldestEndpointUptimeUnixTimestamp int64
	var found bool
	for rows.Next() {
		_ = rows.Scan(&oldestEndpointUptimeUnixTimestamp)
		found = true
	}
	if !found {
		return 0, errNoRowsReturned
	}
	return time.Since(time.Unix(oldestEndpointUptimeUnixTimestamp, 0)), nil
}

func (s *Store) getLastEndpointResultSuccessValue(tx *sql.Tx, endpointID int64) (bool, error) {
	var success bool
	err := tx.QueryRow("SELECT success FROM endpoint_results WHERE endpoint_id = $1 ORDER BY endpoint_result_id DESC LIMIT 1", endpointID).Scan(&success)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errNoRowsReturned
		}
		return false, err
	}
	return success, nil
}

// deleteOldEndpointEvents deletes endpoint events that are no longer needed
func (s *Store) deleteOldEndpointEvents(tx *sql.Tx, endpointID int64) error {
	_, err := tx.Exec(
		`
			DELETE FROM endpoint_events 
			WHERE endpoint_id = $1
				AND endpoint_event_id NOT IN (
					SELECT endpoint_event_id 
					FROM endpoint_events
					WHERE endpoint_id = $1
					ORDER BY endpoint_event_id DESC
					LIMIT $2
				)
		`,
		endpointID,
		s.maximumNumberOfEvents,
	)
	return err
}

// deleteOldEndpointResults deletes endpoint results that are no longer needed
func (s *Store) deleteOldEndpointResults(tx *sql.Tx, endpointID int64) error {
	_, err := tx.Exec(
		`
			DELETE FROM endpoint_results
			WHERE endpoint_id = $1 
				AND endpoint_result_id NOT IN (
					SELECT endpoint_result_id
					FROM endpoint_results
					WHERE endpoint_id = $1
					ORDER BY endpoint_result_id DESC
					LIMIT $2
				)
		`,
		endpointID,
		s.maximumNumberOfResults,
	)
	return err
}

func (s *Store) deleteOldUptimeEntries(tx *sql.Tx, endpointID int64, maxAge time.Time) error {
	_, err := tx.Exec("DELETE FROM endpoint_uptimes WHERE endpoint_id = $1 AND hour_unix_timestamp < $2", endpointID, maxAge.Unix())
	return err
}

func generateCacheKey(endpointKey string, p *paging.EndpointStatusParams) string {
	return fmt.Sprintf("%s-%d-%d-%d-%d", endpointKey, p.EventsPage, p.EventsPageSize, p.ResultsPage, p.ResultsPageSize)
}

func extractKeyAndParamsFromCacheKey(cacheKey string) (string, *paging.EndpointStatusParams, error) {
	parts := strings.Split(cacheKey, "-")
	if len(parts) < 5 {
		return "", nil, fmt.Errorf("invalid cache key: %s", cacheKey)
	}
	params := &paging.EndpointStatusParams{}
	var err error
	if params.EventsPage, err = strconv.Atoi(parts[len(parts)-4]); err != nil {
		return "", nil, fmt.Errorf("invalid cache key: %w", err)
	}
	if params.EventsPageSize, err = strconv.Atoi(parts[len(parts)-3]); err != nil {
		return "", nil, fmt.Errorf("invalid cache key: %w", err)
	}
	if params.ResultsPage, err = strconv.Atoi(parts[len(parts)-2]); err != nil {
		return "", nil, fmt.Errorf("invalid cache key: %w", err)
	}
	if params.ResultsPageSize, err = strconv.Atoi(parts[len(parts)-1]); err != nil {
		return "", nil, fmt.Errorf("invalid cache key: %w", err)
	}
	return strings.Join(parts[:len(parts)-4], "-"), params, nil
}
