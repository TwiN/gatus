package database

import (
	"database/sql"
	"errors"
	"log"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/util"
	_ "modernc.org/sqlite"
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
		    service_id         INTEGER REFERENCES service(id),
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
		    service_id             INTEGER REFERENCES service(id),
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
		    service_result_id            INTEGER REFERENCES service_result(service_result_id),
		    condition                    TEXT,
		    success                      INTEGER
		)
	`)
	return err
}

// TODO: add parameter event and uptime & only fetch them if necessary

func (s *Store) GetAllServiceStatusesWithResultPagination(page, pageSize int) map[string]*core.ServiceStatus {
	panic("implement me")
}

// GetServiceStatus returns the service status for a given service name in the given group
func (s *Store) GetServiceStatus(groupName, serviceName string) *core.ServiceStatus {
	return s.GetServiceStatusByKey(util.ConvertGroupAndServiceToKey(groupName, serviceName))
}

// GetServiceStatusByKey returns the service status for a given key
func (s *Store) GetServiceStatusByKey(key string) *core.ServiceStatus {
	serviceID, serviceName, serviceGroup, err := s.getServiceIDNameAndGroupByKey(key)
	if err != nil {
		return nil
	}
	serviceStatus := &core.ServiceStatus{
		Name:    serviceName,
		Group:   serviceGroup,
		Key:     key,
		Results: nil,
		Events:  nil,
		Uptime:  nil,
	}
	if serviceStatus.Events, err = s.getEventsByServiceID(serviceID); err != nil {
		log.Printf("[database][GetServiceStatusByKey] Failed to retrieve events for key=%s: %s", key, err.Error())
	}
	serviceStatus.Results, err = s.getResultsByServiceID(serviceID)
	return serviceStatus
}

func (s *Store) getServiceIDNameAndGroupByKey(key string) (id int64, group, name string, err error) {
	rows, err := s.db.Query("SELECT service_id, service_name, service_group FROM service WHERE service_key = $1 LIMIT 1", key)
	if err != nil {
		return 0, "", "", err
	}
	for rows.Next() {
		_ = rows.Scan(&id, &name, &group)
	}
	if id == 0 {
		return 0, "", "", errServiceNotFoundInDatabase
	}
	return
}

func (s *Store) getEventsByServiceID(serviceID int64) (events []*core.Event, err error) {
	rows, err := s.db.Query("SELECT event_type, event_timestamp FROM service_event WHERE service_id = $1 ORDER BY event_timestamp DESC LIMIT $2", serviceID, core.MaximumNumberOfEvents)
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

func (s *Store) getResultsByServiceID(serviceID int64) (results []*core.Result, err error) {
	rows, err := s.db.Query(`
			SELECT service_result_id, success, connected, status, dns_rcode, certificate_expiration, hostname, ip, duration, timestamp
			FROM service_result
			WHERE service_id = $1
			ORDER BY timestamp DESC
			LIMIT $2`,
		serviceID,
		core.MaximumNumberOfResults,
	)
	if err != nil {
		return nil, err
	}
	idResultMap := make(map[int64]*core.Result)
	for rows.Next() {
		result := &core.Result{}
		var id int64
		_ = rows.Scan(&id, &result.Success, &result.Connected, &result.HTTPStatus, &result.DNSRCode, &result.CertificateExpiration, &result.Hostname, &result.IP, &result.Duration, &result.Timestamp)
		results = append(results, result)
		idResultMap[id] = result
	}
	_ = rows.Close()
	// Get the conditionResults
	transaction, err := s.db.Begin()
	if err != nil {
		return
	}
	for serviceResultID, result := range idResultMap {
		rows, err = transaction.Query(`
			SELECT service_result_id, condition, success
			FROM service_result_condition
			WHERE service_result_id = $1`,
			serviceResultID,
		)
		if err != nil {
			_ = transaction.Rollback()
			return
		}
		for rows.Next() {
			conditionResult := &core.ConditionResult{}
			//var id int64
			_ = rows.Scan(&conditionResult.Condition, &conditionResult.Success)
			result.ConditionResults = append(result.ConditionResults, conditionResult)
		}
		_ = rows.Close()
	}
	err = transaction.Commit()
	if err != nil {
		_ = transaction.Rollback()
		return
	}
	return
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
				return // failed to insert service
			}
		} else {
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
		return
	}
	//log.Printf("there are currently %d events", numberOfEvents)
	if numberOfEvents == 0 {
		// There's no events yet, which means we need to add the EventStart and the first healthy/unhealthy event
		err = s.insertEvent(tx, serviceID, &core.Event{
			Type:      core.EventStart,
			Timestamp: result.Timestamp.Add(-result.Duration),
		})
		if err != nil {
			// Silently fail
			log.Printf("[database][Insert] Failed to insert event=%s for group=%s; service=%s: %s", core.EventStart, service.Name, service.Group, err.Error())
		}
		event := generateEventBasedOnResult(result)
		err = s.insertEvent(tx, serviceID, event)
		if err != nil {
			// Silently fail
			log.Printf("[database][Insert] Failed to insert event=%s for group=%s; service=%s: %s", event.Type, service.Name, service.Group, err.Error())
		}
	} else {
		// Get the success value of the previous result
		var lastResultSuccess bool
		lastResultSuccess, err = s.getLastServiceResultSuccessValue(tx, serviceID)
		if err != nil {
			log.Printf("[database][Insert] Failed to retrieve outcome of previous result for group=%s; service=%s: %s", service.Name, service.Group, err.Error())
		} else {
			// If we managed to retrieve the outcome of the previous result, we'll compare it with the new result.
			// If the final outcome (success or failure) of the previous and the new result aren't the same, it means
			// that the service either went from Healthy to Unhealthy or Unhealthy -> Healthy, therefore, we'll add
			// an event to mark the change in state
			if lastResultSuccess != result.Success {
				event := generateEventBasedOnResult(result)
				err = s.insertEvent(tx, serviceID, event)
				if err != nil {
					// Silently fail
					log.Printf("[database][Insert] Failed to insert event=%s for group=%s; service=%s: %s", event.Type, service.Name, service.Group, err.Error())
				}
			}
		}
	}
	// Second, we need to insert the result.
	err = s.insertResult(tx, serviceID, result)
	if err != nil {
		// Silently fail
		log.Printf("[database][Insert] Failed to insert result for group=%s; service=%s: %s", service.Name, service.Group, err.Error())
	}
	//log.Printf("[database][Insert] Successfully inserted result in duration=%dns", time.Since(start).Nanoseconds())
	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
	}
	return
}

func (s *Store) getServiceID(tx *sql.Tx, service *core.Service) (int64, error) {
	rows, err := tx.Query(
		"SELECT service_id FROM service WHERE service_key = $1",
		service.Key(),
		service.Group,
	)
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

func (s *Store) getLastServiceResultSuccessValue(tx *sql.Tx, serviceID int64) (bool, error) {
	rows, err := tx.Query("SELECT success FROM service_result WHERE service_id = $1 ORDER BY timestamp LIMIT 1", serviceID)
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

// insertResult inserts a result in the store
func (s *Store) insertResult(tx *sql.Tx, serviceID int64, result *core.Result) error {
	res, err := tx.Exec(`
			INSERT INTO service_result (service_id, success, connected, status, dns_rcode, certificate_expiration, hostname, ip, duration, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		serviceID,
		result.Success,
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
			_ = tx.Rollback()
			return err
		}
	}
	return nil
}

// DeleteAllServiceStatusesNotInKeys removes all rows owned by a service whose key is not within the keys provided
func (s *Store) DeleteAllServiceStatusesNotInKeys(keys []string) int {
	panic("implement me")
}

// Clear deletes everything from the store
func (s *Store) Clear() {
	panic("implement me")
}

// Save does nothing, because this store is immediately persistent.
func (s *Store) Save() error {
	return nil
}

func (s *Store) Close() {
	_ = s.db.Close()
}
