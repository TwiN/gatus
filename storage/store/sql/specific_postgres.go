package sql

func (s *Store) createPostgresSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS service (
			service_id    BIGSERIAL PRIMARY KEY,
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
			service_event_id   BIGSERIAL PRIMARY KEY,
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
			service_result_id      BIGSERIAL PRIMARY KEY,
			service_id             BIGINT REFERENCES service(service_id) ON DELETE CASCADE,
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
		CREATE TABLE IF NOT EXISTS service_result_condition (
			service_result_condition_id  BIGSERIAL PRIMARY KEY,
			service_result_id            BIGINT REFERENCES service_result(service_result_id) ON DELETE CASCADE,
			condition                    TEXT,
			success                      BOOLEAN
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS service_uptime (
			service_uptime_id     BIGSERIAL PRIMARY KEY,
			service_id            BIGINT REFERENCES service(service_id) ON DELETE CASCADE,
			hour_unix_timestamp   BIGINT,
			total_executions      BIGINT,
			successful_executions BIGINT,
			total_response_time   BIGINT,
			UNIQUE(service_id, hour_unix_timestamp)
		)
	`)
	return err
}
