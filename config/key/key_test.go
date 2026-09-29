package key

import "testing"

func TestConvertGroupAndNameToKey(t *testing.T) {
	type Scenario struct {
		GroupName      string
		Name           string
		ExpectedOutput string
	}
	scenarios := []Scenario{
		{
			GroupName:      "Core",
			Name:           "Front End",
			ExpectedOutput: "core_front-end",
		},
		{
			GroupName:      "Load balancers",
			Name:           "us-west-2",
			ExpectedOutput: "load-balancers_us-west-2",
		},
		{
			GroupName:      "a/b test",
			Name:           "a",
			ExpectedOutput: "a-b-test_a",
		},
		{
			GroupName:      "",
			Name:           "name",
			ExpectedOutput: "_name",
		},
		{
			GroupName:      "API (v1)",
			Name:           "endpoint",
			ExpectedOutput: "api-(v1)_endpoint",
		},
		{
			GroupName:      "website (admin)",
			Name:           "test",
			ExpectedOutput: "website-(admin)_test",
		},
		{
			GroupName:      "search",
			Name:           "query&filter",
			ExpectedOutput: "search_query-filter",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.ExpectedOutput, func(t *testing.T) {
			output := ConvertGroupAndNameToKey(scenario.GroupName, scenario.Name)
			if output != scenario.ExpectedOutput {
				t.Errorf("expected '%s', got '%s'", scenario.ExpectedOutput, output)
			}
		})
	}
}
