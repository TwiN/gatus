package database

import (
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/paging"
	"github.com/TwinProduction/gatus/util"
	_ "modernc.org/sqlite"
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// Note that only exported functions in this file may create, commit, or rollback a transaction //
//////////////////////////////////////////////////////////////////////////////////////////////////

const (
	arraySeparator = "|~|"

	uptimeCleanUpThreshold  = 10 * 24 * time.Hour             // Maximum uptime age before triggering a clean up
	eventsCleanUpThreshold  = core.MaximumNumberOfEvents * 2  // Maximum number of events before triggering a clean up
	resultsCleanUpThreshold = core.MaximumNumberOfResults * 2 // Maximum number of results before triggering a clean up

	uptimeRetention = 7 * 24 * time.Hour
)

var (
	// ErrFilePathNotSpecified is the error returned when path parameter passed in NewStore is blank
	ErrFilePathNotSpecified = errors.New("file path cannot be empty")

	// ErrDatabaseDriverNotSpecified is the error returned when the driver parameter passed in NewStore is blank
	ErrDatabaseDriverNotSpecified = errors.New("database driver cannot be empty")

	errServiceNotFoundInDatabase = errors.New("service does not exist in database")
	errNoRowsReturned            = errors.New("expected a row to be returned, but none was")
)

// Store that leverages a database
type Store struct {
	driver, file string

	db *sql.DB
}

// NewStore initializes the database and creates the schema if it doesn't already exist in the file specified
func NewStore(driver, path string) (*Store, error) {
	if len(driver) == 0 {
		return nil, ErrDatabaseDriverNotSpecified
	}
	if len(path) == 0 {
		return nil, ErrFilePathNotSpecified
	}
	store := &Store{driver: driver, file: path}
	var err error
	if store.db, err = sql.Open(driver, path); err != nil {
		return nil, err
	}
	if driver == "sqlite" {
		_, _ = store.db.Exec("PRAGMA foreign_keys=ON")
		_, _ = store.db.Exec("PRAGMA journal_mode=WAL")
		// Prevents driver from running into "database is locked" errors
		// This is because we're using WAL to improve performance
		store.db.SetMaxOpenConns(1)
	}
	if err = store.createSchema(); err != nil {
		_ = store.db.Close()
		return nil, err
	}
	return store, nil
}

// createSchema creates the schema required to perform all database operations.
func (s *Store) createSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS service (
		    service_id    INTEGER PRIMARY KEY,
			service_key   TEXT UNIQUE,
			service_name  TEXT,
			service_group TEXT,
		    UNIQUE(service_name, service_group)
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS service_event (
		    service_event_id   INTEGER PRIMARY KEY,
		    service_id         INTEGER REFERENCES service(service_id) ON DELETE CASCADE,
			event_type         TEXT,
			event_timestamp    TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS service_result (
		    service_result_id      INTEGER PRIMARY KEY,
		    service_id             INTEGER REFERENCES service(service_id) ON DELETE CASCADE,
			success                INTEGER,
		    errors                 TEXT,
		    connected              INTEGER,
		    status                 INTEGER,
		    dns_rcode              TEXT,
		    certificate_expiration INTEGER,
		    hostname               TEXT,
		    ip                     TEXT,
		    duration               INTEGER,
		    timestamp              TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS service_result_condition (
		    service_result_condition_id  INTEGER PRIMARY KEY,
		    service_result_id            INTEGER REFERENCES service_result(service_result_id) ON DELETE CASCADE,
		    condition                    TEXT,
		    success                      INTEGER
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS service_uptime (
		    service_uptime_id     INTEGER PRIMARY KEY,
		    service_id            INTEGER REFERENCES service(service_id) ON DELETE CASCADE,
		    hour_unix_timestamp   INTEGER,
		    total_executions      INTEGER,
		    successful_executions INTEGER,
		    total_response_time   INTEGER,
		    UNIQUE(service_id, hour_unix_timestamp)
		)
	`)
	return err
}

// GetAllServiceStatuses returns all monitored core.ServiceStatus
// with a subset of core.Result defined by the page and pageSize parameters
func (s *Store) GetAllServiceStatuses(params *paging.ServiceStatusParams) map[string]*core.ServiceStatus {
	tx, err := s.db.Begin()
	if err != nil {
		return nil
	}
	keys, err := s.getAllServiceKeys(tx)
	if err != nil {
		_ = tx.Rollback()
		return nil
	}
	serviceStatuses := make(map[string]*core.ServiceStatus, len(keys))
	for _, key := range keys {
		serviceStatus, err := s.getServiceStatusByKey(tx, key, params)
		if err != nil {
			continue
		}
		serviceStatuses[key] = serviceStatus
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return serviceStatuses
}

// GetServiceStatus returns the service status for a given service name in the given group
func (s *Store) GetServiceStatus(groupName, serviceName string, parameters *paging.ServiceStatusParams) *core.ServiceStatus {
	return s.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(groupName, serviceName), parameters)
}

// GetServiceStatusByKey returns the service status for a given key
func (s *Store) GetServiceStatusByKey(key string, params *paging.ServiceStatusParams) *core.ServiceStatus {
	tx, err := s.db.Begin()
	if err != nil {
		return nil
	}
	serviceStatus, err := s.getServiceStatusByKey(tx, key, params)
	if err != nil {
		_ = tx.Rollback()
		return nil
	}
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return serviceStatus
}

// Insert adds the observed result for the specified service into the store
func (s *Store) Insert(service *core.Service, result *core.Result) {
	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	//start := time.Now()
	serviceID, err := s.getServiceID(tx, service)
	if err != nil {
		if err == errServiceNotFoundInDatabase {
			// Service doesn't exist in the database, insert it
			if serviceID, err = s.insertService(tx, service); err != nil {
				_ = tx.Rollback()
				return // failed to insert service
			}
		} else {
			_ = tx.Rollback()
			return
		}
	}
	// First, we need to check if we need to insert a new event.
	//
	// A new event must be added if either of the following cases happen:
	// 1. There is only 1 event. The total number of events for a service can only be 1 if the only existing event is
	//    of type EventStart, in which case we will have to create a new event of type EventHealthy or EventUnhealthy
	//    based on result.Success.
	// 2. The lastResult.Success != result.Success. This implies that the service went from healthy to unhealthy or
	//    vice-versa, in which case we will have to create a new event of type EventHealthy or EventUnhealthy
	//	  based on result.Success.
	numberOfEvents, err := s.getNumberOfEventsByServiceID(tx, serviceID)
	if err != nil {
		log.Printf("[database][Insert] Failed to retrieve total number of events for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
	}
	if numberOfEvents == 0 {
		// There's no events yet, which means we need to add the EventStart and the first healthy/unhealthy event
		err = s.insertEvent(tx, serviceID, &core.Event{
			Type:      core.EventStart,
			Timestamp: result.Timestamp.Add(-50 * time.Millisecond),
		})
		if err != nil {
			// Silently fail
			log.Printf("[database][Insert] Failed to insert event=%s for group=%s; service=%s: %s", core.EventStart, service.Group, service.Name, err.Error())
		}
		event := generateEventBasedOnResult(result)
		if err = s.insertEvent(tx, serviceID, event); err != nil {
			// Silently fail
			log.Printf("[database][Insert] Failed to insert event=%s for group=%s; service=%s: %s", event.Type, service.Group, service.Name, err.Error())
		}
	} else {
		// Get the success value of the previous result
		var lastResultSuccess bool
		if lastResultSuccess, err = s.getLastServiceResultSuccessValue(tx, serviceID); err != nil {
			log.Printf("[database][Insert] Failed to retrieve outcome of previous result for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
		} else {
			// If we managed to retrieve the outcome of the previous result, we'll compare it with the new result.
			// If the final outcome (success or failure) of the previous and the new result aren't the same, it means
			// that the service either went from Healthy to Unhealthy or Unhealthy -> Healthy, therefore, we'll add
			// an event to mark the change in state
			if lastResultSuccess != result.Success {
				event := generateEventBasedOnResult(result)
				if err = s.insertEvent(tx, serviceID, event); err != nil {
					// Silently fail
					log.Printf("[database][Insert] Failed to insert event=%s for group=%s; service=%s: %s", event.Type, service.Group, service.Name, err.Error())
				}
			}
		}
		// Clean up old events if there's more than twice the maximum number of events
		// This lets us both keep the table clean without impacting performance too much
		// (since we're only deleting MaximumNumberOfEvents at a time instead of 1)
		if numberOfEvents > resultsCleanUpThreshold {
			if err = s.deleteOldServiceEvents(tx, serviceID); err != nil {
				log.Printf("[database][Insert] Failed to delete old events for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
			}
		}
	}
	// Second, we need to insert the result.
	if err = s.insertResult(tx, serviceID, result); err != nil {
		log.Printf("[database][Insert] Failed to insert result for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
		_ = tx.Rollback() // If we can't insert the result, we'll rollback now since there's no point continuing
		return
	}
	// Clean up old results
	numberOfResults, err := s.getNumberOfResultsByServiceID(tx, serviceID)
	if err != nil {
		log.Printf("[database][Insert] Failed to retrieve total number of results for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
	} else {
		if numberOfResults > eventsCleanUpThreshold {
			if err = s.deleteOldServiceResults(tx, serviceID); err != nil {
				log.Printf("[database][Insert] Failed to delete old results for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
			}
		}
	}
	// Finally, we need to insert the uptime data.
	// Because the uptime data significantly outlives the results, we can't rely on the results for determining the uptime
	if err = s.updateServiceUptime(tx, serviceID, result); err != nil {
		log.Printf("[database][Insert] Failed to update uptime for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
	}
	// Clean up old uptime entries
	ageOfOldestUptimeEntry, err := s.getAgeOfOldestServiceUptimeEntry(tx, serviceID)
	if err != nil {
		log.Printf("[database][Insert] Failed to retrieve oldest service uptime entry for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
	} else {
		if ageOfOldestUptimeEntry > uptimeCleanUpThreshold {
			if err = s.deleteOldUptimeEntries(tx, serviceID, time.Now().Add(-(uptimeRetention + time.Hour))); err != nil {
				log.Printf("[database][Insert] Failed to delete old uptime entries for group=%s; service=%s: %s", service.Group, service.Name, err.Error())
			}
		}
	}
	//log.Printf("[database][Insert] Successfully inserted result in duration=%dns", time.Since(start).Nanoseconds())
	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
	}
	return
}

// DeleteAllServiceStatusesNotInKeys removes all rows owned by a service whose key is not within the keys provided
func (s *Store) DeleteAllServiceStatusesNotInKeys(keys []string) int {
	panic("implement me")
}

// Clear deletes everything from the store
func (s *Store) Clear() {
	_, _ = s.db.Exec("DELETE FROM service")
}

// Save does nothing, because this store is immediately persistent.
func (s *Store) Save() error {
	return nil
}

// Close the database handle
func (s *Store) Close() {
	_ = s.db.Close()
}

// insertService inserts a service in the store and returns the generated id of said service
func (s *Store) insertService(tx *sql.Tx, service *core.Service) (int64, error) {
	//log.Printf("[database][insertService] Inserting service with group=%s and name=%s", service.Group, service.Name)
	result, err := tx.Exec(
		"INSERT INTO service (service_key, service_name, service_group) VALUES ($1, $2, $3)",
		service.Key(),
		service.Name,
		service.Group,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// insertEvent inserts a service event in the store
func (s *Store) insertEvent(tx *sql.Tx, serviceID int64, event *core.Event) error {
	_, err := tx.Exec(
		"INSERT INTO service_event (service_id, event_type, event_timestamp) VALUES ($1, $2, $3)",
		serviceID,
		event.Type,
		event.Timestamp,
	)
	if err != nil {
		return err
	}
	return nil
}

// insertResult inserts a result in the store
func (s *Store) insertResult(tx *sql.Tx, serviceID int64, result *core.Result) error {
	res, err := tx.Exec(
		`
			INSERT INTO service_result (service_id, success, errors, connected, status, dns_rcode, certificate_expiration, hostname, ip, duration, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`,
		serviceID,
		result.Success,
		strings.Join(result.Errors, arraySeparator),
		result.Connected,
		result.HTTPStatus,
		result.DNSRCode,
		result.CertificateExpiration,
		result.Hostname,
		result.IP,
		result.Duration,
		result.Timestamp,
	)
	if err != nil {
		return err
	}
	serviceResultID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	return s.insertConditionResults(tx, serviceResultID, result.ConditionResults)
}

func (s *Store) insertConditionResults(tx *sql.Tx, serviceResultID int64, conditionResults []*core.ConditionResult) error {
	var err error
	for _, cr := range conditionResults {
		_, err = tx.Exec("INSERT INTO service_result_condition (service_result_id, condition, success) VALUES ($1, $2, $3)",
			serviceResultID,
			cr.Condition,
			cr.Success,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) updateServiceUptime(tx *sql.Tx, serviceID int64, result *core.Result) error {
	unixTimestampFlooredAtHour := result.Timestamp.Truncate(time.Hour).Unix()
	var successfulExecutions int
	if result.Success {
		successfulExecutions = 1
	}
	_, err := tx.Exec(
		`
			INSERT INTO service_uptime (service_id, hour_unix_timestamp, total_executions, successful_executions, total_response_time) 
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT(service_id, hour_unix_timestamp) DO UPDATE SET
				total_executions = excluded.total_executions + total_executions,
			    successful_executions = excluded.successful_executions + successful_executions,
			    total_response_time = excluded.total_response_time + total_response_time
		`,
		serviceID,
		unixTimestampFlooredAtHour,
		1,
		successfulExecutions,
		result.Duration.Milliseconds(),
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) getAllServiceKeys(tx *sql.Tx) (keys []string, err error) {
	rows, err := tx.Query("SELECT service_key FROM service")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key string
		_ = rows.Scan(&key)
		keys = append(keys, key)
	}
	_ = rows.Close()
	return
}

func (s *Store) getServiceStatusByKey(tx *sql.Tx, key string, parameters *paging.ServiceStatusParams) (*core.ServiceStatus, error) {
	serviceID, serviceName, serviceGroup, err := s.getServiceIDGroupAndNameByKey(tx, key)
	if err != nil {
		return nil, err
	}
	serviceStatus := core.NewServiceStatus(key, serviceGroup, serviceName)
	if parameters.EventsPageSize > 0 {
		if serviceStatus.Events, err = s.getEventsByServiceID(tx, serviceID, parameters.EventsPage, parameters.EventsPageSize); err != nil {
			log.Printf("[database][getServiceStatusByKey] Failed to retrieve events for key=%s: %s", key, err.Error())
		}
	}
	if parameters.ResultsPageSize > 0 {
		if serviceStatus.Results, err = s.getResultsByServiceID(tx, serviceID, parameters.ResultsPage, parameters.ResultsPageSize); err != nil {
			log.Printf("[database][getServiceStatusByKey] Failed to retrieve results for key=%s: %s", key, err.Error())
		}
	}
	if parameters.IncludeUptime {
		now := time.Now()
		serviceStatus.Uptime.LastHour, _, _ = s.getServiceUptime(tx, serviceID, now.Add(-time.Hour), now)
		serviceStatus.Uptime.LastTwentyFourHours, _, _ = s.getServiceUptime(tx, serviceID, now.Add(-24*time.Hour), now)
		serviceStatus.Uptime.LastSevenDays, _, _ = s.getServiceUptime(tx, serviceID, now.Add(-7*24*time.Hour), now)
	}
	return serviceStatus, nil
}

func (s *Store) getServiceIDGroupAndNameByKey(tx *sql.Tx, key string) (id int64, group, name string, err error) {
	rows, err := tx.Query("SELECT service_id, service_group, service_name FROM service WHERE service_key = $1 LIMIT 1", key)
	if err != nil {
		return 0, "", "", err
	}
	for rows.Next() {
		_ = rows.Scan(&id, &group, &name)
	}
	_ = rows.Close()
	if id == 0 {
		return 0, "", "", errServiceNotFoundInDatabase
	}
	return
}

func (s *Store) getEventsByServiceID(tx *sql.Tx, serviceID int64, page, pageSize int) (events []*core.Event, err error) {
	rows, err := tx.Query(
		`
			SELECT event_type, event_timestamp
			FROM service_event
			WHERE service_id = $1
			ORDER BY service_event_id DESC
			LIMIT $2 OFFSET $3
		`,
		serviceID,
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
	_ = rows.Close()
	return
}

func (s *Store) getResultsByServiceID(tx *sql.Tx, serviceID int64, page, pageSize int) (results []*core.Result, err error) {
	rows, err := tx.Query(
		`
			SELECT service_result_id, success, errors, connected, status, dns_rcode, certificate_expiration, hostname, ip, duration, timestamp
			FROM service_result
			WHERE service_id = $1
			ORDER BY timestamp DESC
			LIMIT $2 OFFSET $3
		`,
		serviceID,
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
		_ = rows.Scan(&id, &result.Success, &joinedErrors, &result.Connected, &result.HTTPStatus, &result.DNSRCode, &result.CertificateExpiration, &result.Hostname, &result.IP, &result.Duration, &result.Timestamp)
		result.Errors = strings.Split(joinedErrors, arraySeparator)
		results = append(results, result)
		idResultMap[id] = result
	}
	_ = rows.Close()
	// Get the conditionResults
	for serviceResultID, result := range idResultMap {
		rows, err = tx.Query(
			`
				SELECT service_result_id, condition, success
				FROM service_result_condition
				WHERE service_result_id = $1
			`,
			serviceResultID,
		)
		if err != nil {
			return
		}
		for rows.Next() {
			conditionResult := &core.ConditionResult{}
			_ = rows.Scan(&conditionResult.Condition, &conditionResult.Success)
			result.ConditionResults = append(result.ConditionResults, conditionResult)
		}
		_ = rows.Close()
	}
	return
}

func (s *Store) getServiceUptime(tx *sql.Tx, serviceID int64, from, to time.Time) (uptime float64, avgResponseTime time.Duration, err error) {
	rows, err := tx.Query(
		`
			SELECT SUM(total_executions), SUM(successful_executions), SUM(total_response_time)
			FROM service_uptime
			WHERE service_id = $1
				AND hour_unix_timestamp >= $2
				AND hour_unix_timestamp <= $3
		`,
		serviceID,
		from.Unix(),
		to.Unix(),
	)
	if err != nil {
		return 0, 0, err
	}
	var totalExecutions, totalSuccessfulExecutions, totalResponseTime int
	for rows.Next() {
		_ = rows.Scan(&totalExecutions, &totalSuccessfulExecutions, &totalResponseTime)
		break
	}
	_ = rows.Close()
	if totalExecutions > 0 {
		uptime = float64(totalSuccessfulExecutions) / float64(totalExecutions)
		avgResponseTime = time.Duration(float64(totalResponseTime)/float64(totalExecutions)) * time.Millisecond
	}
	return
}

func (s *Store) getServiceID(tx *sql.Tx, service *core.Service) (int64, error) {
	rows, err := tx.Query("SELECT service_id FROM service WHERE service_key = $1", service.Key())
	if err != nil {
		return 0, err
	}
	var id int64
	var found bool
	for rows.Next() {
		_ = rows.Scan(&id)
		found = true
		break
	}
	_ = rows.Close()
	if !found {
		return 0, errServiceNotFoundInDatabase
	}
	return id, nil
}

func (s *Store) getNumberOfEventsByServiceID(tx *sql.Tx, serviceID int64) (int64, error) {
	rows, err := tx.Query("SELECT COUNT(1) FROM service_event WHERE service_id = $1", serviceID)
	if err != nil {
		return 0, err
	}
	var numberOfEvents int64
	var found bool
	for rows.Next() {
		_ = rows.Scan(&numberOfEvents)
		found = true
		break
	}
	_ = rows.Close()
	if !found {
		return 0, errNoRowsReturned
	}
	return numberOfEvents, nil
}

func (s *Store) getNumberOfResultsByServiceID(tx *sql.Tx, serviceID int64) (int64, error) {
	rows, err := tx.Query("SELECT COUNT(1) FROM service_result WHERE service_id = $1", serviceID)
	if err != nil {
		return 0, err
	}
	var numberOfResults int64
	var found bool
	for rows.Next() {
		_ = rows.Scan(&numberOfResults)
		found = true
		break
	}
	_ = rows.Close()
	if !found {
		return 0, errNoRowsReturned
	}
	return numberOfResults, nil
}

func (s *Store) getAgeOfOldestServiceUptimeEntry(tx *sql.Tx, serviceID int64) (time.Duration, error) {
	rows, err := tx.Query(
		`
			SELECT hour_unix_timestamp 
			FROM service_uptime 
			WHERE service_id = $1 
			ORDER BY hour_unix_timestamp
			LIMIT 1
		`,
		serviceID,
	)
	if err != nil {
		return 0, err
	}
	var oldestServiceUptimeUnixTimestamp int64
	var found bool
	for rows.Next() {
		_ = rows.Scan(&oldestServiceUptimeUnixTimestamp)
		found = true
		break
	}
	_ = rows.Close()
	if !found {
		return 0, errNoRowsReturned
	}
	return time.Since(time.Unix(oldestServiceUptimeUnixTimestamp, 0)), nil
}

func (s *Store) getLastServiceResultSuccessValue(tx *sql.Tx, serviceID int64) (bool, error) {
	rows, err := tx.Query("SELECT success FROM service_result WHERE service_id = $1 ORDER BY service_result_id DESC LIMIT 1", serviceID)
	if err != nil {
		return false, err
	}
	var success bool
	var found bool
	for rows.Next() {
		_ = rows.Scan(&success)
		found = true
		break
	}
	_ = rows.Close()
	if !found {
		return false, errNoRowsReturned
	}
	return success, nil
}

// deleteOldServiceEvents deletes old service events that are no longer needed
func (s *Store) deleteOldServiceEvents(tx *sql.Tx, serviceID int64) error {
	_, err := tx.Exec(
		`
			DELETE FROM service_event 
			WHERE service_id = $1 
			  AND service_event_id NOT IN (
			      SELECT service_event_id 
			      FROM service_event
			      WHERE service_id = $1
			      ORDER BY service_event_id DESC
			      LIMIT $2
			  	)
		`,
		serviceID,
		core.MaximumNumberOfEvents,
	)
	if err != nil {
		return err
	}
	//rowsAffected, _ := result.RowsAffected()
	//log.Printf("deleted %d rows from service_event", rowsAffected)
	return nil
}

// deleteOldServiceResults deletes old service results that are no longer needed
func (s *Store) deleteOldServiceResults(tx *sql.Tx, serviceID int64) error {
	_, err := tx.Exec(
		`
			DELETE FROM service_result 
			WHERE service_id = $1 
			  AND service_result_id NOT IN (
			      SELECT service_result_id
			      FROM service_result
			      WHERE service_id = $1
			      ORDER BY service_result_id DESC
			      LIMIT $2
			  	)
		`,
		serviceID,
		core.MaximumNumberOfResults,
	)
	if err != nil {
		return err
	}
	//rowsAffected, _ := result.RowsAffected()
	//log.Printf("deleted %d rows from service_result", rowsAffected)
	return nil
}

func (s *Store) deleteOldUptimeEntries(tx *sql.Tx, serviceID int64, maxAge time.Time) error {
	_, err := tx.Exec("DELETE FROM service_uptime WHERE service_id = $1 AND hour_unix_timestamp < $2", serviceID, maxAge.Unix())
	//if err != nil {
	//	return err
	//}
	//rowsAffected, _ := result.RowsAffected()
	//log.Printf("deleted %d rows from service_uptime", rowsAffected)
	return err
}
