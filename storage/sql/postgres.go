package sql

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgresDatabase returns a new database via Postgres
func NewPostgresDatabase(connectionString string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
