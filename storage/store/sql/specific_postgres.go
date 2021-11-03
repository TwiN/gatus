package sql

func (s *Store) createPostgresSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoints (
			endpoint_id    BIGSERIAL PRIMARY KEY,
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
			endpoint_event_id  BIGSERIAL PRIMARY KEY,
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
			endpoint_result_id     BIGSERIAL PRIMARY KEY,
			endpoint_id            BIGINT REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			success                BOOLEAN,
			errors                 TEXT,
			connected              BOOLEAN,
			status                 BIGINT,
			dns_rcode              TEXT,
			certificate_expiration BIGINT,
			hostname               TEXT,
			ip                     TEXT,
			duration               BIGINT,
			timestamp              TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_result_conditions (
			endpoint_result_condition_id  BIGSERIAL PRIMARY KEY,
			endpoint_result_id            BIGINT REFERENCES endpoint_results(endpoint_result_id) ON DELETE CASCADE,
			condition                     TEXT,
			success                       BOOLEAN
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_uptimes (
			endpoint_uptime_id     BIGSERIAL PRIMARY KEY,
			endpoint_id            BIGINT REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			hour_unix_timestamp    BIGINT,
			total_executions       BIGINT,
			successful_executions  BIGINT,
			total_response_time    BIGINT,
			UNIQUE(endpoint_id, hour_unix_timestamp)
		)
	`)
	return err
}
