package memory

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

func TestAddResult(t *testing.T) {
	ep := &endpoint.Endpoint{Name: "name", Group: "group"}
	endpointStatus := endpoint.NewStatus(ep.Group, ep.Name)
	for i := 0; i < (storage.DefaultMaximumNumberOfResults+storage.DefaultMaximumNumberOfEvents)*2; i++ {
		AddResult(endpointStatus, &endpoint.Result{Success: i%2 == 0, Timestamp: time.Now()}, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
	}
	if len(endpointStatus.Results) != storage.DefaultMaximumNumberOfResults {
		t.Errorf("expected endpointStatus.Results to not exceed a length of %d", storage.DefaultMaximumNumberOfResults)
	}
	if len(endpointStatus.Events) != storage.DefaultMaximumNumberOfEvents {
		t.Errorf("expected endpointStatus.Events to not exceed a length of %d", storage.DefaultMaximumNumberOfEvents)
	}
	// Try to add nil endpointStatus
	AddResult(nil, &endpoint.Result{Timestamp: time.Now()}, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
}

func TestShallowCopyEndpointStatus(t *testing.T) {
	ep := &endpoint.Endpoint{Name: "name", Group: "group"}
	endpointStatus := endpoint.NewStatus(ep.Group, ep.Name)
	ts := time.Now().Add(-25 * time.Hour)
	for i := 0; i < 25; i++ {
		AddResult(endpointStatus, &endpoint.Result{Success: i%2 == 0, Timestamp: ts}, storage.DefaultMaximumNumberOfResults, storage.DefaultMaximumNumberOfEvents)
		ts = ts.Add(time.Hour)
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(-1, -1)).Results) != 0 {
		t.Error("expected to have 0 result")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(1, 1)).Results) != 1 {
		t.Error("expected to have 1 result")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(5, 0)).Results) != 0 {
		t.Error("expected to have 0 results")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(-1, 20)).Results) != 0 {
		t.Error("expected to have 0 result, because the page was invalid")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(1, -1)).Results) != 0 {
		t.Error("expected to have 0 result, because the page size was invalid")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(1, 10)).Results) != 10 {
		t.Error("expected to have 10 results, because given a page size of 10, page 1 should have 10 elements")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(2, 10)).Results) != 10 {
		t.Error("expected to have 10 results, because given a page size of 10, page 2 should have 10 elements")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(3, 10)).Results) != 5 {
		t.Error("expected to have 5 results, because given a page size of 10, page 3 should have 5 elements")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(4, 10)).Results) != 0 {
		t.Error("expected to have 0 results, because given a page size of 10, page 4 should have 0 elements")
	}
	if len(ShallowCopyEndpointStatus(endpointStatus, paging.NewEndpointStatusParams().WithResults(1, 50)).Results) != 25 {
		t.Error("expected to have 25 results, because there's only 25 results")
	}
}

func TestShallowCopySuiteStatus(t *testing.T) {
	testSuite := &suite.Suite{Name: "test-suite", Group: "test-group"}
	suiteStatus := &suite.Status{
		Name:    testSuite.Name,
		Group:   testSuite.Group,
		Key:     testSuite.Key(),
		Results: []*suite.Result{},
	}
	
	ts := time.Now().Add(-25 * time.Hour)
	for i := 0; i < 25; i++ {
		result := &suite.Result{
			Name:      testSuite.Name,
			Group:     testSuite.Group,
			Success:   i%2 == 0,
			Timestamp: ts,
			Duration:  time.Duration(i*10) * time.Millisecond,
		}
		suiteStatus.Results = append(suiteStatus.Results, result)
		ts = ts.Add(time.Hour)
	}

	t.Run("invalid-page-negative", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(-1, 10))
		if len(result.Results) != 0 {
			t.Errorf("expected 0 results for negative page, got %d", len(result.Results))
		}
	})

	t.Run("invalid-page-zero", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(0, 10))
		if len(result.Results) != 0 {
			t.Errorf("expected 0 results for zero page, got %d", len(result.Results))
		}
	})

	t.Run("invalid-pagesize-negative", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(1, -1))
		if len(result.Results) != 0 {
			t.Errorf("expected 0 results for negative page size, got %d", len(result.Results))
		}
	})

	t.Run("zero-pagesize", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(1, 0))
		if len(result.Results) != 0 {
			t.Errorf("expected 0 results for zero page size, got %d", len(result.Results))
		}
	})

	t.Run("nil-params", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, nil)
		if len(result.Results) != 25 {
			t.Errorf("expected 25 results for nil params, got %d", len(result.Results))
		}
	})

	t.Run("zero-params", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, &paging.SuiteStatusParams{Page: 0, PageSize: 0})
		if len(result.Results) != 25 {
			t.Errorf("expected 25 results for zero-value params, got %d", len(result.Results))
		}
	})

	t.Run("first-page", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(1, 10))
		if len(result.Results) != 10 {
			t.Errorf("expected 10 results for page 1, size 10, got %d", len(result.Results))
		}
		// Verify newest results are returned (reverse pagination)
		if len(result.Results) > 0 && !result.Results[len(result.Results)-1].Timestamp.After(result.Results[0].Timestamp) {
			t.Error("expected newest result to be at the end")
		}
	})

	t.Run("second-page", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(2, 10))
		if len(result.Results) != 10 {
			t.Errorf("expected 10 results for page 2, size 10, got %d", len(result.Results))
		}
	})

	t.Run("last-partial-page", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(3, 10))
		if len(result.Results) != 5 {
			t.Errorf("expected 5 results for page 3, size 10, got %d", len(result.Results))
		}
	})

	t.Run("beyond-available-pages", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(4, 10))
		if len(result.Results) != 0 {
			t.Errorf("expected 0 results for page beyond available data, got %d", len(result.Results))
		}
	})

	t.Run("large-page-size", func(t *testing.T) {
		result := ShallowCopySuiteStatus(suiteStatus, paging.NewSuiteStatusParams().WithPagination(1, 100))
		if len(result.Results) != 25 {
			t.Errorf("expected 25 results for large page size, got %d", len(result.Results))
		}
	})
}

