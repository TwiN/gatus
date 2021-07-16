package store

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/database"
	"github.com/TwinProduction/gatus/storage/store/memory"
	"github.com/TwinProduction/gatus/storage/store/paging"
)

func BenchmarkStore_GetAllServiceStatuses(b *testing.B) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetAllServiceStatuses.db")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name     string
		Store    Store
		Parallel bool
	}
	scenarios := []Scenario{
		{
			Name:     "memory",
			Store:    memoryStore,
			Parallel: false,
		},
		{
			Name:     "memory-parallel",
			Store:    memoryStore,
			Parallel: true,
		},
		{
			Name:     "database",
			Store:    databaseStore,
			Parallel: false,
		},
		{
			Name:     "database-parallel",
			Store:    databaseStore,
			Parallel: true,
		},
	}
	for _, scenario := range scenarios {
		scenario.Store.Insert(&testService, &testSuccessfulResult)
		scenario.Store.Insert(&testService, &testUnsuccessfulResult)
		b.Run(scenario.Name, func(b *testing.B) {
			if scenario.Parallel {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						scenario.Store.GetAllServiceStatuses(paging.NewServiceStatusParams().WithResults(1, 20))
					}
				})
			} else {
				for n := 0; n < b.N; n++ {
					scenario.Store.GetAllServiceStatuses(paging.NewServiceStatusParams().WithResults(1, 20))
				}
			}
			b.ReportAllocs()
		})
		scenario.Store.Clear()
	}
}

func BenchmarkStore_Insert(b *testing.B) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_Insert.db")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name     string
		Store    Store
		Parallel bool
	}
	scenarios := []Scenario{
		{
			Name:     "memory",
			Store:    memoryStore,
			Parallel: false,
		},
		{
			Name:     "memory-parallel",
			Store:    memoryStore,
			Parallel: true,
		},
		{
			Name:     "database",
			Store:    databaseStore,
			Parallel: false,
		},
		{
			Name:     "database-parallel",
			Store:    databaseStore,
			Parallel: false,
		},
	}
	for _, scenario := range scenarios {
		b.Run(scenario.Name, func(b *testing.B) {
			if scenario.Parallel {
				b.RunParallel(func(pb *testing.PB) {
					n := 0
					for pb.Next() {
						var result core.Result
						if n%10 == 0 {
							result = testUnsuccessfulResult
						} else {
							result = testSuccessfulResult
						}
						result.Timestamp = time.Now()
						scenario.Store.Insert(&testService, &result)
						n++
					}
				})
			} else {
				for n := 0; n < b.N; n++ {
					var result core.Result
					if n%10 == 0 {
						result = testUnsuccessfulResult
					} else {
						result = testSuccessfulResult
					}
					result.Timestamp = time.Now()
					scenario.Store.Insert(&testService, &result)
				}
			}
			b.ReportAllocs()
			scenario.Store.Clear()
		})
	}
}

func BenchmarkStore_GetServiceStatusByKey(b *testing.B) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	databaseStore, err := database.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetServiceStatusByKey.db")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	defer databaseStore.Close()
	type Scenario struct {
		Name     string
		Store    Store
		Parallel bool
	}
	scenarios := []Scenario{
		{
			Name:     "memory",
			Store:    memoryStore,
			Parallel: false,
		},
		//{
		//	Name:     "memory-parallel",
		//	Store:    memoryStore,
		//	Parallel: true,
		//},
		{
			Name:     "database",
			Store:    databaseStore,
			Parallel: false,
		},
		//{
		//	Name:     "database-parallel",
		//	Store:    databaseStore,
		//	Parallel: true,
		//},
	}
	for _, scenario := range scenarios {
		for i := 0; i < 10; i++ {
			scenario.Store.Insert(&testService, &testSuccessfulResult)
			scenario.Store.Insert(&testService, &testUnsuccessfulResult)
		}
		b.Run(scenario.Name, func(b *testing.B) {
			if scenario.Parallel {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						scenario.Store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithResults(1, 20))
					}
				})
			} else {
				for n := 0; n < b.N; n++ {
					scenario.Store.GetServiceStatusByKey(testService.Key(), paging.NewServiceStatusParams().WithResults(1, 20))
				}
			}
			b.ReportAllocs()
		})
		scenario.Store.Clear()
	}
}
