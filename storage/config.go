package storage

// Config is the configuration for alerting providers
type Config struct {
	// ConnectionString is the postgres connection string to use for the database connection. The connection
	// string must be as defined here: https://gorm.io/docs/connecting_to_the_database.html#PostgreSQL
	ConnectionString string `yaml:"connectionString"`

	// InMemory indicates whether Gatus should use an in-memory datastore or not. If this is set to false
	// then ConnectionString must be set to something valid.
	InMemory bool `yaml:"inMemory"`
}
