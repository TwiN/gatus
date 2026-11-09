package store

import (
	"strconv"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/gatus/v5/storage/store/memory"
	"github.com/TwiN/gatus/v5/storage/store/sql"
)

func BenchmarkStore_GetAllEndpointStatuses(b *testing.B) {
	memoryStore, err := memory.NewStore(storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sql.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetAllEndpointStatuses.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
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
		numberOfEndpoints := []int{10, 25, 50, 100}
		for _, numberOfEndpointsToCreate := range numberOfEndpoints {
			// Create endpoints and insert results
			for i := range numberOfEndpointsToCreate {
				ep := testEndpoint
				ep.Name = "endpoint" + strconv.Itoa(i)
				// InsertEndpointResult 20 results for each endpoint
				for range 20 {
					scenario.Store.InsertEndpointResult(&ep, &testSuccessfulResult)
				}
			}
			// Run the scenarios
			b.Run(scenario.Name+"-with-"+strconv.Itoa(numberOfEndpointsToCreate)+"-endpoints", func(b *testing.B) {
				if scenario.Parallel {
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							_, _ = scenario.Store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20))
						}
					})
				} else {
					for n := 0; n < b.N; n++ {
						_, _ = scenario.Store.GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(1, 20))
					}
				}
				b.ReportAllocs()
			})
			scenario.Store.Clear()
		}
	}
}

func BenchmarkStore_Insert(b *testing.B) {
	memoryStore, err := memory.NewStore(storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sql.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_Insert.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
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
						var result endpoint.Result
						if n%10 == 0 {
							result = testUnsuccessfulResult
						} else {
							result = testSuccessfulResult
						}
						result.Timestamp = time.Now()
						scenario.Store.InsertEndpointResult(&testEndpoint, &result)
						n++
					}
				})
			} else {
				for n := 0; n < b.N; n++ {
					var result endpoint.Result
					if n%10 == 0 {
						result = testUnsuccessfulResult
					} else {
						result = testSuccessfulResult
					}
					result.Timestamp = time.Now()
					scenario.Store.InsertEndpointResult(&testEndpoint, &result)
				}
			}
			b.ReportAllocs()
			scenario.Store.Clear()
		})
	}
}

func BenchmarkStore_GetEndpointStatusByKey(b *testing.B) {
	memoryStore, err := memory.NewStore(storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sql.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetEndpointStatusByKey.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
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
		for range 50 {
			scenario.Store.InsertEndpointResult(&testEndpoint, &testSuccessfulResult)
			scenario.Store.InsertEndpointResult(&testEndpoint, &testUnsuccessfulResult)
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

// BenchmarkStore_GetEndpointHistoryByKey benchmarks the retrieval of the dashboard history of a single endpoint
// checked every minute over a 24h window (1440 results), compared to the retrieval of its last 50 results, which is
// what the dashboard retrieves when no dashboard-history is configured.
func BenchmarkStore_GetEndpointHistoryByKey(b *testing.B) {
	const numberOfResults = 1440
	memoryStore, err := memory.NewStore(numberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	sqliteStore, err := sql.NewStore("sqlite", b.TempDir()+"/BenchmarkStore_GetEndpointHistoryByKey.db", false, numberOfResults, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	defer sqliteStore.Close()
	stores := []struct {
		Name  string
		Store Store
	}{
		{Name: "memory", Store: memoryStore},
		{Name: "sqlite", Store: sqliteStore},
	}
	start := time.Now().Add(-24 * time.Hour)
	for _, s := range stores {
		for i := range numberOfResults {
			result := testSuccessfulResult
			if i%100 == 0 {
				result = testUnsuccessfulResult
			}
			result.Timestamp = start.Add(time.Duration(i) * time.Minute)
			if err := s.Store.InsertEndpointResult(&testEndpoint, &result); err != nil {
				b.Fatal("failed to insert result:", err.Error())
			}
		}
		b.Run(s.Name+"-history-24h", func(b *testing.B) {
			for b.Loop() {
				from, to := endpoint.NewHistoryWindow(time.Now(), 24*time.Hour, endpoint.NumberOfHistoryBins)
				if _, err := s.Store.GetEndpointHistoryByKey(testEndpoint.Key(), from, to, endpoint.NumberOfHistoryBins, 2*time.Minute); err != nil {
					b.Fatal("failed to get history:", err.Error())
				}
			}
		})
		b.Run(s.Name+"-last-50-results", func(b *testing.B) {
			for b.Loop() {
				if _, err := s.Store.GetEndpointStatusByKey(testEndpoint.Key(), paging.NewEndpointStatusParams().WithResults(1, 50)); err != nil {
					b.Fatal("failed to get status:", err.Error())
				}
			}
		})
	}
}
