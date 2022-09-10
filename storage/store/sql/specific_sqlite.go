package sql

func (s *Store) createSQLiteSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoints (
			endpoint_id    INTEGER PRIMARY KEY,
			endpoint_key   TEXT UNIQUE,
			endpoint_name  TEXT NOT NULL,
			endpoint_group TEXT NOT NULL,
			UNIQUE(endpoint_name, endpoint_group)
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_events (
			endpoint_event_id  INTEGER PRIMARY KEY,
			endpoint_id        INTEGER   NOT NULL REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			event_type         TEXT      NOT NULL,
			event_timestamp    TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_results (
			endpoint_result_id     INTEGER PRIMARY KEY,
			endpoint_id            INTEGER   NOT NULL REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			success                INTEGER   NOT NULL,
			errors                 TEXT      NOT NULL,
			connected              INTEGER   NOT NULL,
			status                 INTEGER   NOT NULL,
			dns_rcode              TEXT      NOT NULL,
			certificate_expiration INTEGER   NOT NULL,
		    domain_expiration      INTEGER   NOT NULL,
			hostname               TEXT      NOT NULL,
			ip                     TEXT      NOT NULL,
			duration               INTEGER   NOT NULL,
			timestamp              TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_result_conditions (
			endpoint_result_condition_id  INTEGER PRIMARY KEY,
			endpoint_result_id            INTEGER NOT NULL REFERENCES endpoint_results(endpoint_result_id) ON DELETE CASCADE,
			condition                     TEXT    NOT NULL,
			success                       INTEGER NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_uptimes (
			endpoint_uptime_id    INTEGER PRIMARY KEY,
			endpoint_id           INTEGER NOT NULL REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			hour_unix_timestamp   INTEGER NOT NULL,
			total_executions      INTEGER NOT NULL,
			successful_executions INTEGER NOT NULL,
			total_response_time   INTEGER NOT NULL,
			UNIQUE(endpoint_id, hour_unix_timestamp)
		)
	`)
	// Silent table modifications TODO: Remove this
	_, _ = s.db.Exec(`ALTER TABLE endpoint_results ADD domain_expiration INTEGER NOT NULL DEFAULT 0`)
	return err
}
