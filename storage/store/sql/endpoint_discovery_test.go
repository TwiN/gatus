package sql

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
)

func newEndpointDiscoveryStore(tb testing.TB, history int) *Store {
	tb.Helper()
	s, err := NewStore("sqlite", tb.TempDir()+"/discovery.db", false, history, storage.DefaultMaximumNumberOfEvents)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(s.Close)
	tx, err := s.db.Begin()
	if err != nil {
		tb.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO suites (suite_id, suite_key, suite_name, suite_group) VALUES (1, 'suite', 'suite', '')`); err != nil {
		tb.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO suite_results (suite_result_id, suite_id, success, errors, duration, timestamp) VALUES (1, 1, 1, '', 0, CURRENT_TIMESTAMP)`); err != nil {
		tb.Fatal(err)
	}
	suiteResultID := int64(1)
	for _, name := range []string{"z-standalone", "a-mixed", "suite-only", "empty"} {
		ep := &endpoint.Endpoint{Name: name}
		id, err := s.insertEndpoint(tx, ep)
		if err != nil {
			tb.Fatal(err)
		}
		if name == "empty" {
			continue
		}
		if name != "z-standalone" {
			if err = s.insertEndpointResultWithSuiteID(tx, id, &testSuccessfulResult, &suiteResultID); err != nil {
				tb.Fatal(err)
			}
		}
		if name != "suite-only" {
			for i := 0; i < history; i++ {
				if err = s.insertEndpointResult(tx, id, &testSuccessfulResult); err != nil {
					tb.Fatal(err)
				}
			}
		}
	}
	if err = tx.Commit(); err != nil {
		tb.Fatal(err)
	}
	return s
}

func TestStore_GetAllEndpointKeysStandaloneResults(t *testing.T) {
	t.Parallel()
	s := newEndpointDiscoveryStore(t, 3)
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	keys, err := s.getAllEndpointKeys(tx)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{(&endpoint.Endpoint{Name: "a-mixed"}).Key(), (&endpoint.Endpoint{Name: "z-standalone"}).Key()}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("expected sorted, unique standalone endpoints %v, got %v", want, keys)
	}
}

func BenchmarkStore_GetAllEndpointKeysHistory(b *testing.B) {
	for _, history := range []int{100, 10000, 70000} {
		b.Run(fmt.Sprintf("results=%d", history), func(b *testing.B) {
			s := newEndpointDiscoveryStore(b, history)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tx, err := s.db.Begin()
				if err != nil {
					b.Fatal(err)
				}
				keys, err := s.getAllEndpointKeys(tx)
				_ = tx.Rollback()
				if err != nil {
					b.Fatal(err)
				}
				if len(keys) != 2 {
					b.Fatalf("expected 2 endpoints, got %d", len(keys))
				}
			}
		})
	}
}
