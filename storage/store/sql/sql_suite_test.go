package sql

import (
	"fmt"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

func TestStore_GetSuiteStatusesWithEndpointResultsAndConditions(t *testing.T) {
	store, _ := NewStore("sqlite", t.TempDir()+"/TestStore_GetSuiteStatusesWithEndpointResultsAndConditions.db", false, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	defer store.Close()
	suites := []*suite.Suite{{Name: "first", Group: "group"}, {Name: "second", Group: "group"}}
	start := time.Now().Add(-time.Hour)
	const numberOfResults = 3
	for i := 0; i < numberOfResults; i++ {
		for _, su := range suites {
			result := &suite.Result{
				Success:   i%2 == 0,
				Timestamp: start.Add(time.Duration(i) * time.Minute),
				Duration:  time.Duration(i+1) * time.Second,
			}
			for _, epName := range []string{"login", "logout"} {
				result.EndpointResults = append(result.EndpointResults, &endpoint.Result{
					Name:      fmt.Sprintf("%s-%s", su.Name, epName),
					Success:   true,
					Timestamp: result.Timestamp,
					Duration:  time.Duration(i+1) * time.Millisecond,
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: fmt.Sprintf("%s-%s-%d-a", su.Name, epName, i), Success: true},
						{Condition: fmt.Sprintf("%s-%s-%d-b", su.Name, epName, i), Success: false},
					},
				})
			}
			if err := store.InsertSuiteResult(su, result); err != nil {
				t.Fatal("expected no error, got", err.Error())
			}
		}
	}
	validate := func(t *testing.T, status *suite.Status, su *suite.Suite, firstIndex int) {
		for j, result := range status.Results {
			i := firstIndex + j
			if result.Name != su.Name || result.Group != su.Group {
				t.Errorf("expected result for %s/%s, got %s/%s", su.Group, su.Name, result.Group, result.Name)
			}
			if !result.Timestamp.Equal(start.Add(time.Duration(i) * time.Minute)) {
				t.Errorf("result %d of %s: unexpected timestamp %s", i, su.Name, result.Timestamp)
			}
			if len(result.EndpointResults) != 2 {
				t.Fatalf("result %d of %s: expected 2 endpoint results, got %d", i, su.Name, len(result.EndpointResults))
			}
			for k, epName := range []string{"login", "logout"} {
				epResult := result.EndpointResults[k]
				if expected := fmt.Sprintf("%s-%s", su.Name, epName); epResult.Name != expected {
					t.Errorf("result %d of %s: expected endpoint result %s, got %s", i, su.Name, expected, epResult.Name)
				}
				if len(epResult.ConditionResults) != 2 {
					t.Fatalf("result %d of %s: expected 2 condition results for %s, got %d", i, su.Name, epResult.Name, len(epResult.ConditionResults))
				}
				for l, suffix := range []string{"a", "b"} {
					if expected := fmt.Sprintf("%s-%s-%d-%s", su.Name, epName, i, suffix); epResult.ConditionResults[l].Condition != expected {
						t.Errorf("result %d of %s: expected condition %s, got %s", i, su.Name, expected, epResult.ConditionResults[l].Condition)
					}
				}
			}
		}
	}
	t.Run("GetAllSuiteStatuses", func(t *testing.T) {
		statuses, err := store.GetAllSuiteStatuses(paging.NewSuiteStatusParams().WithPagination(1, 20))
		if err != nil {
			t.Fatal("expected no error, got", err.Error())
		}
		if len(statuses) != len(suites) {
			t.Fatalf("expected %d suite statuses, got %d", len(suites), len(statuses))
		}
		for i, status := range statuses {
			if len(status.Results) != numberOfResults {
				t.Fatalf("expected %d results for %s, got %d", numberOfResults, status.Name, len(status.Results))
			}
			validate(t, status, suites[i], 0)
		}
	})
	t.Run("GetSuiteStatusByKey-paginated", func(t *testing.T) {
		status, err := store.GetSuiteStatusByKey(suites[1].Key(), paging.NewSuiteStatusParams().WithPagination(1, 2))
		if err != nil {
			t.Fatal("expected no error, got", err.Error())
		}
		if len(status.Results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(status.Results))
		}
		validate(t, status, suites[1], numberOfResults-2)
	})
	t.Run("GetSuiteStatusByKey-empty-page", func(t *testing.T) {
		status, err := store.GetSuiteStatusByKey(suites[0].Key(), paging.NewSuiteStatusParams().WithPagination(5, 20))
		if err != nil {
			t.Fatal("expected no error, got", err.Error())
		}
		if len(status.Results) != 0 {
			t.Errorf("expected no results, got %d", len(status.Results))
		}
	})
}
