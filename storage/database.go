package storage

import (
	"fmt"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/sql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ResultGetter returns gatus Results from the persistence layer
type ResultGetter interface {
	GetAll() (map[string]*core.ServiceStatus, error)
}

// ResultStorer stores the given result in the persistence layer
type ResultStorer interface {
	Store(service *core.Service, res *core.Result) error
}

// Database represents the gatus database and provides CRUD operations
type Database struct {
	database    *gorm.DB
	memoryStore *InMemoryStore
}

// NewDatabase constructs and returns a new Database. It will also auto-migrate the database schema if applicable
func NewDatabase(config Config) (*Database, error) {
	inMemoryStore := newInMemoryStore()
	if config.InMemory {
		return &Database{
			database:    nil,
			memoryStore: &inMemoryStore,
		}, nil
	}

	db, err := sql.NewPostgresDatabase(config.ConnectionString)
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&service{}, &evaluationError{}, &conditionResult{}, &result{}); err != nil {
		return nil, err
	}
	return &Database{
		database:    db,
		memoryStore: nil,
	}, nil
}

// GetAll returns all gatus results from teh database
func (db *Database) GetAll() (map[string]*core.ServiceStatus, error) {
	if db.memoryStore != nil {
		return db.memoryStore.GetAll(), nil
	}

	services := []service{}

	queryResult := db.database.Preload("Results.ConditionResults").Preload("Results.Errors").Preload(clause.Associations).Find(&services)
	if queryResult.Error != nil {
		return nil, queryResult.Error
	}

	svcToResults := make(map[string]*core.ServiceStatus)

	for _, svc := range services {
		uptime := core.NewUptime()

		key := fmt.Sprintf("%s_%s", svc.Group, svc.Name)
		crs := []*core.Result{}
		for _, r := range svc.Results {
			cr := ConvertFromStorage(r)
			crs = append(crs, &cr)
			uptime.ProcessResult(&cr)
		}
		svcToResults[key] = &core.ServiceStatus{
			Name:    svc.Name,
			Group:   svc.Group,
			Results: crs,
			Uptime:  uptime,
		}
	}

	return svcToResults, nil
}

// Store inserts the provided result in the database
func (db *Database) Store(svc *core.Service, res *core.Result) error {
	if db.memoryStore != nil {
		db.memoryStore.Insert(svc, res)
		return nil
	}

	svcFromDb := &service{}

	queryResult := db.database.Where(service{Name: svc.Name, Group: svc.Group}).FirstOrCreate(&svcFromDb)
	if queryResult.Error != nil {
		return queryResult.Error
	}

	r := ConvertToStorage(*res)
	return db.database.Model(&svcFromDb).Association("Results").Append([]result{r})
}
