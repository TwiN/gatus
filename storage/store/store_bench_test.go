package store

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v3/core"
	"github.com/TwiN/gatus/v3/storage/store/common/paging"
	"github.com/TwiN/gatus/v3/storage/store/memory"
	"github.com/TwiN/gatus/v3/storage/store/sql"
)

func BenchmarkStore_GetAllEndpointStatuses(b *testing.B) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sql.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetAllEndpointStatuses.db")
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
		scenario.Store.Insert(&testEndpoint, &testSuccessfulResult)
		scenario.Store.Insert(&testEndpoint, &testUnsuccessfulResult)
		b.Run(scenario.Name, func(b *testing.B) {
			if scenario.Parallel {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						scenario.Store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20))
					}
				})
			} else {
				for n := 0; n < b.N; n++ {
					scenario.Store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20))
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
	sqliteStore, err := sql.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_Insert.db")
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
						scenario.Store.Insert(&testEndpoint, &result)
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
					scenario.Store.Insert(&testEndpoint, &result)
				}
			}
			b.ReportAllocs()
			scenario.Store.Clear()
		})
	}
}

func BenchmarkStore_GetEndpointStatusByKey(b *testing.B) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sql.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetEndpointStatusByKey.db")
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
			scenario.Store.Insert(&testEndpoint, &testSuccessfulResult)
			scenario.Store.Insert(&testEndpoint, &testUnsuccessfulResult)
		}
		b.Run(scenario.Name, func(b *testing.B) {
			if scenario.Parallel {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						scenario.Store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithResults(1, 20))
					}
				})
			} else {
				for n := 0; n < b.N; n++ {
					scenario.Store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithResults(1, 20))
				}
			}
			b.ReportAllocs()
		})
		scenario.Store.Clear()
	}
}
