package memory

import (
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
)

// ShallowCopyEndpointStatus returns a shallow copy of a Status with only the results
// within the range defined by the page and pageSize parameters
func ShallowCopyEndpointStatus(ss *endpoint.Status, params *paging.EndpointStatusParams) *endpoint.Status {
	shallowCopy := &endpoint.Status{
		Name:   ss.Name,
		Group:  ss.Group,
		Key:    ss.Key,
		Uptime: endpoint.NewUptime(),
	}
	if params == nil || (params.ResultsPage == 0 && params.ResultsPageSize == 0 && params.EventsPage == 0 && params.EventsPageSize == 0) {
		shallowCopy.Results = ss.Results
		shallowCopy.Events = ss.Events
	} else {
		numberOfResults := len(ss.Results)
		resultsStart, resultsEnd := getStartAndEndIndex(numberOfResults, params.ResultsPage, params.ResultsPageSize)
		if resultsStart < 0 || resultsEnd < 0 {
			shallowCopy.Results = []*endpoint.Result{}
		} else {
			shallowCopy.Results = ss.Results[resultsStart:resultsEnd]
		}
		numberOfEvents := len(ss.Events)
		eventsStart, eventsEnd := getStartAndEndIndex(numberOfEvents, params.EventsPage, params.EventsPageSize)
		if eventsStart < 0 || eventsEnd < 0 {
			shallowCopy.Events = []*endpoint.Event{}
		} else {
			shallowCopy.Events = ss.Events[eventsStart:eventsEnd]
		}
	}
	return shallowCopy
}

// ShallowCopySuiteStatus returns a shallow copy of a suite Status with only the results
// within the range defined by the page and pageSize parameters
func ShallowCopySuiteStatus(ss *suite.Status, params *paging.SuiteStatusParams) *suite.Status {
	shallowCopy := &suite.Status{
		Name:  ss.Name,
		Group: ss.Group,
		Key:   ss.Key,
	}
	if params == nil || (params.Page == 0 && params.PageSize == 0) {
		shallowCopy.Results = ss.Results
	} else {
		numberOfResults := len(ss.Results)
		resultsStart, resultsEnd := getStartAndEndIndex(numberOfResults, params.Page, params.PageSize)
		if resultsStart < 0 || resultsEnd < 0 {
			shallowCopy.Results = []*suite.Result{}
		} else {
			shallowCopy.Results = ss.Results[resultsStart:resultsEnd]
		}
	}
	return shallowCopy
}

func getStartAndEndIndex(numberOfResults int, page, pageSize int) (int, int) {
	if page < 1 || pageSize < 0 {
		return -1, -1
	}
	start := numberOfResults - (page * pageSize)
	end := numberOfResults - ((page - 1) * pageSize)
	if start > numberOfResults {
		start = -1
	} else if start < 0 {
		start = 0
	}
	if end > numberOfResults {
		end = numberOfResults
	}
	return start, end
}

// AddResult adds a Result to Status.Results and makes sure that there are
// no more than MaximumNumberOfResults results in the Results slice
func AddResult(ss *endpoint.Status, result *endpoint.Result, maximumNumberOfResults, maximumNumberOfEvents int) {
	if ss == nil {
		return
	}
	if len(ss.Results) > 0 {
		// Check if there's any change since the last result
		if ss.Results[len(ss.Results)-1].Success != result.Success {
			ss.Events = append(ss.Events, endpoint.NewEventFromResult(result))
			if len(ss.Events) > maximumNumberOfEvents {
				// Doing ss.Events[1:] would usually be sufficient, but in the case where for some reason, the slice has
				// more than one extra element, we can get rid of all of them at once and thus returning the slice to a
				// length of MaximumNumberOfEvents by using ss.Events[len(ss.Events)-MaximumNumberOfEvents:] instead
				ss.Events = ss.Events[len(ss.Events)-maximumNumberOfEvents:]
			}
		}
	} else {
		// This is the first result, so we need to add the first healthy/unhealthy event
		ss.Events = append(ss.Events, endpoint.NewEventFromResult(result))
	}
	ss.Results = append(ss.Results, result)
	if len(ss.Results) > maximumNumberOfResults {
		// Doing ss.Results[1:] would usually be sufficient, but in the case where for some reason, the slice has more
		// than one extra element, we can get rid of all of them at once and thus returning the slice to a length of
		// MaximumNumberOfResults by using ss.Results[len(ss.Results)-MaximumNumberOfResults:] instead
		ss.Results = ss.Results[len(ss.Results)-maximumNumberOfResults:]
	}
	processUptimeAfterResult(ss.Uptime, result)
}
