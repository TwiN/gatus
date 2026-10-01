package endpoint

import (
	"testing"
)

func TestNumberOfResultsInARow(t *testing.T) {
	scenarios := []struct {
		name                string
		results             []*Result
		expectedFailures    int
		expectedSuccesses   int
	}{
		{
			name:              "nil results",
			results:           nil,
			expectedFailures:  0,
			expectedSuccesses: 0,
		},
		{
			name:              "empty results",
			results:           []*Result{},
			expectedFailures:  0,
			expectedSuccesses: 0,
		},
		{
			name:              "single success",
			results:           []*Result{{Success: true}},
			expectedFailures:  0,
			expectedSuccesses: 1,
		},
		{
			name:              "single failure",
			results:           []*Result{{Success: false}},
			expectedFailures:  1,
			expectedSuccesses: 0,
		},
		{
			name: "trailing successes after older failures",
			results: []*Result{
				{Success: false},
				{Success: false},
				{Success: true},
				{Success: true},
			},
			expectedFailures:  0,
			expectedSuccesses: 2,
		},
		{
			name: "trailing failures after older successes",
			results: []*Result{
				{Success: true},
				{Success: true},
				{Success: false},
				{Success: false},
				{Success: false},
			},
			expectedFailures:  3,
			expectedSuccesses: 0,
		},
		{
			name: "all successes",
			results: []*Result{
				{Success: true},
				{Success: true},
				{Success: true},
			},
			expectedFailures:  0,
			expectedSuccesses: 3,
		},
		{
			name: "all failures",
			results: []*Result{
				{Success: false},
				{Success: false},
			},
			expectedFailures:  2,
			expectedSuccesses: 0,
		},
		{
			name: "alternating ending in success",
			results: []*Result{
				{Success: false},
				{Success: true},
			},
			expectedFailures:  0,
			expectedSuccesses: 1,
		},
		{
			name: "alternating ending in failure",
			results: []*Result{
				{Success: true},
				{Success: false},
			},
			expectedFailures:  1,
			expectedSuccesses: 0,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			failures, successes := NumberOfResultsInARow(scenario.results)
			if failures != scenario.expectedFailures {
				t.Errorf("expected %d failures in a row, got %d", scenario.expectedFailures, failures)
			}
			if successes != scenario.expectedSuccesses {
				t.Errorf("expected %d successes in a row, got %d", scenario.expectedSuccesses, successes)
			}
		})
	}
}
