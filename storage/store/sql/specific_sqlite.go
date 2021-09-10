package sql

func (s *Store) createSQLiteSchema() error {
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
