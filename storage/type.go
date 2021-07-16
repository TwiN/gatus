package storage

// Type of the store.
type Type string

const (
	TypeInMemory Type = "inmemory" // In-memory store
	TypeSQLite   Type = "sqlite"   // SQLite store
)
