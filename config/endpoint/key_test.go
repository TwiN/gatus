package endpoint

import "testing"

func TestConvertGroupAndEndpointNameToKey(t *testing.T) {
	type Scenario struct {
		GroupName      string
		EndpointName   string
		ExpectedOutput string
	}
	scenarios := []Scenario{
		{
			GroupName:      "Core",
			EndpointName:   "Front End",
			ExpectedOutput: "core_front-end",
		},
		{
			GroupName:      "Load balancers",
			EndpointName:   "us-west-2",
			ExpectedOutput: "load-balancers_us-west-2",
		},
		{
			GroupName:      "a/b test",
			EndpointName:   "a",
			ExpectedOutput: "a-b-test_a",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.ExpectedOutput, func(t *testing.T) {
			output := ConvertGroupAndEndpointNameToKey(scenario.GroupName, scenario.EndpointName)
			if output != scenario.ExpectedOutput {
				t.Errorf("expected '%s', got '%s'", scenario.ExpectedOutput, output)
			}
		})
	}
}
