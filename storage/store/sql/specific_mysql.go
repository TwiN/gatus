package sql

func (s *Store) createMySQLSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoints (
			endpoint_id    BIGINT AUTO_INCREMENT PRIMARY KEY,
			endpoint_key   VARCHAR(255) UNIQUE,
			endpoint_name  VARCHAR(255) NOT NULL,
			endpoint_group VARCHAR(255) NOT NULL,
			UNIQUE(endpoint_name, endpoint_group)
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_events (
			endpoint_event_id  BIGINT AUTO_INCREMENT PRIMARY KEY,
			endpoint_id        BIGINT       NOT NULL,
			event_type         VARCHAR(255) NOT NULL,
			event_timestamp    DATETIME     NOT NULL,
			FOREIGN KEY (endpoint_id) REFERENCES endpoints(endpoint_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_results (
			endpoint_result_id     BIGINT AUTO_INCREMENT PRIMARY KEY,
			endpoint_id            BIGINT       NOT NULL,
			success                BOOLEAN      NOT NULL,
			errors                 TEXT         NOT NULL,
			connected              BOOLEAN      NOT NULL,
			status                 INT          NOT NULL,
			dns_rcode              VARCHAR(255) NOT NULL,
			certificate_expiration BIGINT       NOT NULL,
			domain_expiration      BIGINT       NOT NULL,
			hostname               VARCHAR(255) NOT NULL,
			ip                     VARCHAR(255) NOT NULL,
			duration               BIGINT       NOT NULL,
			timestamp              DATETIME     NOT NULL,
			FOREIGN KEY (endpoint_id) REFERENCES endpoints(endpoint_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_result_conditions (
			endpoint_result_condition_id  BIGINT AUTO_INCREMENT PRIMARY KEY,
			endpoint_result_id            BIGINT NOT NULL,
			condition                     TEXT   NOT NULL,
			success                       BOOLEAN NOT NULL,
			FOREIGN KEY (endpoint_result_id) REFERENCES endpoint_results(endpoint_result_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_uptimes (
			endpoint_uptime_id     BIGINT AUTO_INCREMENT PRIMARY KEY,
			endpoint_id            BIGINT NOT NULL,
			hour_unix_timestamp    BIGINT NOT NULL,
			total_executions       BIGINT NOT NULL,
			successful_executions  BIGINT NOT NULL,
			total_response_time    BIGINT NOT NULL,
			UNIQUE(endpoint_id, hour_unix_timestamp),
			FOREIGN KEY (endpoint_id) REFERENCES endpoints(endpoint_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoint_alerts_triggered (
			endpoint_alert_trigger_id BIGINT AUTO_INCREMENT PRIMARY KEY,
			endpoint_id                  BIGINT       NOT NULL,
			configuration_checksum       VARCHAR(255) NOT NULL,
			resolve_key                  VARCHAR(255) NOT NULL,
			number_of_successes_in_a_row INT          NOT NULL,
			UNIQUE(endpoint_id, configuration_checksum),
			FOREIGN KEY (endpoint_id) REFERENCES endpoints(endpoint_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}
	return nil
}
