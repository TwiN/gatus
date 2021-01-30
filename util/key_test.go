package util

import "testing"

func TestConvertGroupAndServiceToKey(t *testing.T) {
	type Scenario struct {
		GroupName      string
		ServiceName    string
		ExpectedOutput string
	}
	scenarios := []Scenario{
		{
			GroupName:      "Core",
			ServiceName:    "Front End",
			ExpectedOutput: "core_front-end",
		},
		{
			GroupName:      "Load balancers",
			ServiceName:    "us-west-2",
			ExpectedOutput: "load-balancers_us-west-2",
		},
		{
			GroupName:      "a/b test",
			ServiceName:    "a",
			ExpectedOutput: "a-b-test_a",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.ExpectedOutput, func(t *testing.T) {
			output := ConvertGroupAndServiceToKey(scenario.GroupName, scenario.ServiceName)
			if output != scenario.ExpectedOutput {
				t.Errorf("Expected '%s', got '%s'", scenario.ExpectedOutput, output)
			}
		})
	}
}
