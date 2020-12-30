package storage

import (
	"testing"
)

func TestStorage_WhenInMemoryConfigIsEnabledThenInMemoryDatabaseIsUsed(t *testing.T) {
	config := Config{InMemory: true, ConnectionString: "ConnectionString"}
	db, err := NewDatabase(config)
	if err != nil {
		t.Fatal(err)
	}

	if db.database != nil {
		t.Error("Database should've been using in memory store, but db.database was not nil")
	}
	if db.memoryStore == nil {
		t.Error("Database should've been using in memory store, but db.memoryStore was nil")
	}
}
