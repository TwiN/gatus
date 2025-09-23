package key

import "testing"

func TestConvertGroupAndNameToKey(t *testing.T) {
	type Scenario struct {
		GroupNames     []string
		Name           string
		ExpectedOutput string
	}
	scenarios := []Scenario{
		{
			GroupNames:     []string{"Core"},
			Name:           "Front End",
			ExpectedOutput: "core_front-end",
		},
		{
			GroupNames:     []string{"Load balancers"},
			Name:           "us-west-2",
			ExpectedOutput: "load-balancers_us-west-2",
		},
		{
			GroupNames:     []string{"a/b test"},
			Name:           "a",
			ExpectedOutput: "a-b-test_a",
		},
		{
			GroupNames:     []string{""},
			Name:           "name",
			ExpectedOutput: "_name",
		},
		{
			GroupNames:     []string{"multiple", "groups"},
			Name:           "name",
			ExpectedOutput: "multiple-groups_name",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.ExpectedOutput, func(t *testing.T) {
			output := ConvertGroupAndNameToKey(scenario.GroupNames, scenario.Name)
			if output != scenario.ExpectedOutput {
				t.Errorf("expected '%s', got '%s'", scenario.ExpectedOutput, output)
			}
		})
	}
}
