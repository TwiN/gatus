package sql

func (s *Store) createPostgresSchema() error {
	// Create suite tables
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS suites (
			suite_id    BIGSERIAL PRIMARY KEY,
			suite_key   TEXT UNIQUE,
			suite_name  TEXT NOT NULL,
			suite_group TEXT NOT NULL,
			UNIQUE(suite_name, suite_group)
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS suite_results (
			suite_result_id  BIGSERIAL PRIMARY KEY,
			suite_id         BIGINT    NOT NULL REFERENCES suites(suite_id) ON DELETE CASCADE,
			success          BOOLEAN   NOT NULL,
			errors           TEXT      NOT NULL,
			duration         BIGINT    NOT NULL,
			timestamp        TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	// Create endpoint tables
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoints (
			endpoint_id    BIGSERIAL PRIMARY KEY,
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
			endpoint_event_id  BIGSERIAL PRIMARY KEY,
			endpoint_id        BIGINT    NOT NULL REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			event_type         TEXT      NOT NULL,
			event_timestamp    TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_results (
			endpoint_result_id     BIGSERIAL PRIMARY KEY,
			endpoint_id            BIGINT    NOT NULL REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			success                BOOLEAN   NOT NULL,
			errors                 TEXT      NOT NULL,
			connected              BOOLEAN   NOT NULL,
			status                 BIGINT    NOT NULL,
			dns_rcode              TEXT      NOT NULL,
			certificate_expiration BIGINT    NOT NULL,
			domain_expiration      BIGINT    NOT NULL,
			hostname               TEXT      NOT NULL,
			ip                     TEXT      NOT NULL,
			duration               BIGINT    NOT NULL,
			timestamp              TIMESTAMP NOT NULL,
			suite_result_id        BIGINT    REFERENCES suite_results(suite_result_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_result_conditions (
			endpoint_result_condition_id  BIGSERIAL PRIMARY KEY,
			endpoint_result_id            BIGINT  NOT NULL REFERENCES endpoint_results(endpoint_result_id) ON DELETE CASCADE,
			condition                     TEXT    NOT NULL,
			success                       BOOLEAN NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_uptimes (
			endpoint_uptime_id     BIGSERIAL PRIMARY KEY,
			endpoint_id            BIGINT NOT NULL REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
			hour_unix_timestamp    BIGINT NOT NULL,
			total_executions       BIGINT NOT NULL,
			successful_executions  BIGINT NOT NULL,
			total_response_time    BIGINT NOT NULL,
			UNIQUE(endpoint_id, hour_unix_timestamp)
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_alerts_triggered (
			endpoint_alert_trigger_id     BIGSERIAL PRIMARY KEY,
			endpoint_id                   BIGINT    NOT NULL REFERENCES endpoints(endpoint_id) ON DELETE CASCADE,
		    configuration_checksum        TEXT      NOT NULL,
		    resolve_key		              TEXT      NOT NULL,
			number_of_successes_in_a_row  INTEGER   NOT NULL,
			UNIQUE(endpoint_id, configuration_checksum)
		)
	`)
	if err != nil {
		return err
	}
	// Create index for suite_results
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS suite_results_suite_id_idx ON suite_results (suite_id);
	`)
	// Silent table modifications TODO: Remove this in v6.0.0
	_, _ = s.db.Exec(`ALTER TABLE endpoint_results ADD IF NOT EXISTS domain_expiration BIGINT NOT NULL DEFAULT 0`)
	// Add suite_result_id to endpoint_results table for suite endpoint linkage
	_, _ = s.db.Exec(`ALTER TABLE endpoint_results ADD COLUMN IF NOT EXISTS suite_result_id BIGINT REFERENCES suite_results(suite_result_id) ON DELETE CASCADE`)
	// Create index for suite_result_id
	_, _ = s.db.Exec(`CREATE INDEX IF NOT EXISTS endpoint_results_suite_result_id_idx ON endpoint_results(suite_result_id)`)
	return err
}
