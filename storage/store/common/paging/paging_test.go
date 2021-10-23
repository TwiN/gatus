package paging

import "testing"

func TestNewEndpointStatusParams(t *testing.T) {
	type Scenario struct {
		Name                    string
		Params                  *EndpointStatusParams
		ExpectedEventsPage      int
		ExpectedEventsPageSize  int
		ExpectedResultsPage     int
		ExpectedResultsPageSize int
	}
	scenarios := []Scenario{
		{
			Name:                    "empty-params",
			Params:                  NewEndpointStatusParams(),
			ExpectedEventsPage:      0,
			ExpectedEventsPageSize:  0,
			ExpectedResultsPage:     0,
			ExpectedResultsPageSize: 0,
		},
		{
			Name:                    "with-events-page-2-size-7",
			Params:                  NewEndpointStatusParams().WithEvents(2, 7),
			ExpectedEventsPage:      2,
			ExpectedEventsPageSize:  7,
			ExpectedResultsPage:     0,
			ExpectedResultsPageSize: 0,
		},
		{
			Name:                    "with-events-page-4-size-3-uptime",
			Params:                  NewEndpointStatusParams().WithEvents(4, 3),
			ExpectedEventsPage:      4,
			ExpectedEventsPageSize:  3,
			ExpectedResultsPage:     0,
			ExpectedResultsPageSize: 0,
		},
		{
			Name:                    "with-results-page-1-size-20-uptime",
			Params:                  NewEndpointStatusParams().WithResults(1, 20),
			ExpectedEventsPage:      0,
			ExpectedEventsPageSize:  0,
			ExpectedResultsPage:     1,
			ExpectedResultsPageSize: 20,
		},
		{
			Name:                    "with-results-page-2-size-10-events-page-3-size-50",
			Params:                  NewEndpointStatusParams().WithResults(2, 10).WithEvents(3, 50),
			ExpectedEventsPage:      3,
			ExpectedEventsPageSize:  50,
			ExpectedResultsPage:     2,
			ExpectedResultsPageSize: 10,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if scenario.Params.EventsPage != scenario.ExpectedEventsPage {
				t.Errorf("expected ExpectedEventsPage to be %d, was %d", scenario.ExpectedEventsPageSize, scenario.Params.EventsPage)
			}
			if scenario.Params.EventsPageSize != scenario.ExpectedEventsPageSize {
				t.Errorf("expected EventsPageSize to be %d, was %d", scenario.ExpectedEventsPageSize, scenario.Params.EventsPageSize)
			}
			if scenario.Params.ResultsPage != scenario.ExpectedResultsPage {
				t.Errorf("expected ResultsPage to be %d, was %d", scenario.ExpectedResultsPage, scenario.Params.ResultsPage)
			}
			if scenario.Params.ResultsPageSize != scenario.ExpectedResultsPageSize {
				t.Errorf("expected ResultsPageSize to be %d, was %d", scenario.ExpectedResultsPageSize, scenario.Params.ResultsPageSize)
			}
		})
	}
}
