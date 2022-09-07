package sql

func (s *Store) createSQLiteSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoints (
			endpoint_id    INTEGER PRIMARY KEY,
			endpoint_key   TEXT UNIQUE,
			endpoint_name  TEXT,
			endpoint_group TEXT,
			UNIQUE(endpoint_name, endpoint_group)
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_events (
			endpoint_event_id  INTEGER PRIMARY KEY,
			endpoint_id        INTEGER REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			event_type         TEXT,
			event_timestamp    TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_results (
			endpoint_result_id     INTEGER PRIMARY KEY,
			endpoint_id            INTEGER REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			success                INTEGER,
			errors                 TEXT,
			connected              INTEGER,
			status                 INTEGER,
			dns_rcode              TEXT,
			certificate_expiration INTEGER,
		    domain_expiration      INTEGER,
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
		CREATE TABLE IF NOT EXISTS endpoint_result_conditions (
			endpoint_result_condition_id  INTEGER PRIMARY KEY,
			endpoint_result_id            INTEGER REFERENCES endpoint_results(endpoint_result_id) ON DELETE CASCADE,
			condition                     TEXT,
			success                       INTEGER
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_uptimes (
			endpoint_uptime_id    INTEGER PRIMARY KEY,
			endpoint_id           INTEGER REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			hour_unix_timestamp   INTEGER,
			total_executions      INTEGER,
			successful_executions INTEGER,
			total_response_time   INTEGER,
			UNIQUE(endpoint_id, hour_unix_timestamp)
		)
	`)
	// Silent table modifications
	_, _ = s.db.Exec(`ALTER TABLE endpoint_results ADD domain_expiration INTEGER`)
	return err
}
