package store

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/common/paging"
	"github.com/TwinProduction/gatus/storage/store/memory"
	"github.com/TwinProduction/gatus/storage/store/sqlite"
)

func BenchmarkStore_GetAllServiceStatuses(b *testing.B) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sqlite.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetAllServiceStatuses.db")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	defer sqliteStore.Close()
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
			Name:     "sqlite",
			Store:    sqliteStore,
			Parallel: false,
		},
		{
			Name:     "sqlite-parallel",
			Store:    sqliteStore,
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
	sqliteStore, err := sqlite.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_Insert.db")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	defer sqliteStore.Close()
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
			Name:     "sqlite",
			Store:    sqliteStore,
			Parallel: false,
		},
		{
			Name:     "sqlite-parallel",
			Store:    sqliteStore,
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
	sqliteStore, err := sqlite.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetServiceStatusByKey.db")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	defer sqliteStore.Close()
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
			Name:     "sqlite",
			Store:    sqliteStore,
			Parallel: false,
		},
		{
			Name:     "sqlite-parallel",
			Store:    sqliteStore,
			Parallel: true,
		},
	}
	for _, scenario := range scenarios {
		for i := 0; i < 50; i++ {
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
