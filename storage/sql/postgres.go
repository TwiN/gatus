package sql

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgresDatabase returns a new database via Postgres
func NewPostgresDatabase() (*gorm.DB, error) {
	dsn := "host=localhost user=gatus dbname=gatus password=gatus port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
