package memory

import (
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/paging"
)

// ShallowCopyServiceStatus returns a shallow copy of a ServiceStatus with only the results
// within the range defined by the page and pageSize parameters
func ShallowCopyServiceStatus(ss *core.ServiceStatus, params *paging.ServiceStatusParams) *core.ServiceStatus {
	shallowCopy := &core.ServiceStatus{
		Name:   ss.Name,
		Group:  ss.Group,
		Key:    ss.Key,
		Uptime: core.NewUptime(),
	}
	numberOfResults := len(ss.Results)
	resultsStart, resultsEnd := getStartAndEndIndex(numberOfResults, params.ResultsPage, params.ResultsPageSize)
	if resultsStart < 0 || resultsEnd < 0 {
		shallowCopy.Results = []*core.Result{}
	} else {
		shallowCopy.Results = ss.Results[resultsStart:resultsEnd]
	}
	numberOfEvents := len(ss.Events)
	eventsStart, eventsEnd := getStartAndEndIndex(numberOfEvents, params.EventsPage, params.EventsPageSize)
	if eventsStart < 0 || eventsEnd < 0 {
		shallowCopy.Events = []*core.Event{}
	} else {
		shallowCopy.Events = ss.Events[eventsStart:eventsEnd]
	}
	if params.IncludeUptime {
		shallowCopy.Uptime.LastHour = ss.Uptime.LastHour
		shallowCopy.Uptime.LastTwentyFourHours = ss.Uptime.LastTwentyFourHours
		shallowCopy.Uptime.LastSevenDays = ss.Uptime.LastSevenDays
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

// AddResult adds a Result to ServiceStatus.Results and makes sure that there are
// no more than MaximumNumberOfResults results in the Results slice
func AddResult(ss *core.ServiceStatus, result *core.Result) {
	if ss == nil {
		return
	}
	if len(ss.Results) > 0 {
		// Check if there's any change since the last result
		if ss.Results[len(ss.Results)-1].Success != result.Success {
			ss.Events = append(ss.Events, core.NewEventFromResult(result))
			if len(ss.Events) > core.MaximumNumberOfEvents {
				// Doing ss.Events[1:] would usually be sufficient, but in the case where for some reason, the slice has
				// more than one extra element, we can get rid of all of them at once and thus returning the slice to a
				// length of MaximumNumberOfEvents by using ss.Events[len(ss.Events)-MaximumNumberOfEvents:] instead
				ss.Events = ss.Events[len(ss.Events)-core.MaximumNumberOfEvents:]
			}
		}
	} else {
		// This is the first result, so we need to add the first healthy/unhealthy event
		ss.Events = append(ss.Events, core.NewEventFromResult(result))
	}
	ss.Results = append(ss.Results, result)
	if len(ss.Results) > core.MaximumNumberOfResults {
		// Doing ss.Results[1:] would usually be sufficient, but in the case where for some reason, the slice has more
		// than one extra element, we can get rid of all of them at once and thus returning the slice to a length of
		// MaximumNumberOfResults by using ss.Results[len(ss.Results)-MaximumNumberOfResults:] instead
		ss.Results = ss.Results[len(ss.Results)-core.MaximumNumberOfResults:]
	}
	processUptimeAfterResult(ss.Uptime, result)
}
