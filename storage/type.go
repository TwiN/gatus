package storage

// Type of the store.
type Type string

const (
	TypeMemory   Type = "memory"   // In-memory store
	TypeSQLite   Type = "sqlite"   // SQLite store
	TypePostgres Type = "postgres" // Postgres store
)
