package core

import (
	"time"
)

const (
	// MaximumNumberOfResults is the maximum number of results that ServiceStatus.Results can have
	MaximumNumberOfResults = 100

	// MaximumNumberOfEvents is the maximum number of events that ServiceStatus.Events can have
	MaximumNumberOfEvents = 50
)

// ServiceStatus contains the evaluation Results of a Service
type ServiceStatus struct {
	// Name of the service
	Name string `json:"name,omitempty"`

	// Group the service is a part of. Used for grouping multiple services together on the front end.
	Group string `json:"group,omitempty"`

	// Key is the key representing the ServiceStatus
	Key string `json:"key"`

	// Results is the list of service evaluation results
	Results []*Result `json:"results"`

	// Events is a list of events
	//
	// We don't expose this through JSON, because the main dashboard doesn't need to have this data.
	// However, the detailed service page does leverage this by including it to a map that will be
	// marshalled alongside the ServiceStatus.
	Events []*Event `json:"-"`

	// Uptime information on the service's uptime
	//
	// We don't expose this through JSON, because the main dashboard doesn't need to have this data.
	// However, the detailed service page does leverage this by including it to a map that will be
	// marshalled alongside the ServiceStatus.
	Uptime *Uptime `json:"-"`
}

// NewServiceStatus creates a new ServiceStatus
func NewServiceStatus(service *Service) *ServiceStatus {
	return &ServiceStatus{
		Name:    service.Name,
		Group:   service.Group,
		Key:     service.Key(),
		Results: make([]*Result, 0),
		Events: []*Event{{
			Type:      EventStart,
			Timestamp: time.Now(),
		}},
		Uptime: NewUptime(),
	}
}

// WithResultPagination returns a shallow copy of the ServiceStatus with only the results
// within the range defined by the page and pageSize parameters
func (ss ServiceStatus) WithResultPagination(page, pageSize int) *ServiceStatus {
	shallowCopy := ss
	numberOfResults := len(shallowCopy.Results)
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
	if start < 0 || end < 0 {
		shallowCopy.Results = []*Result{}
	} else {
		shallowCopy.Results = shallowCopy.Results[start:end]
	}
	return &shallowCopy
}

// AddResult adds a Result to ServiceStatus.Results and makes sure that there are
// no more than MaximumNumberOfResults results in the Results slice
func (ss *ServiceStatus) AddResult(result *Result) {
	if len(ss.Results) > 0 {
		// Check if there's any change since the last result
		// OR there's only 1 event, which only happens when there's a start event
		if ss.Results[len(ss.Results)-1].Success != result.Success || len(ss.Events) == 1 {
			event := &Event{Timestamp: result.Timestamp}
			if result.Success {
				event.Type = EventHealthy
			} else {
				event.Type = EventUnhealthy
			}
			ss.Events = append(ss.Events, event)
			if len(ss.Events) > MaximumNumberOfEvents {
				// Doing ss.Events[1:] would usually be sufficient, but in the case where for some reason, the slice has
				// more than one extra element, we can get rid of all of them at once and thus returning the slice to a
				// length of MaximumNumberOfEvents by using ss.Events[len(ss.Events)-MaximumNumberOfEvents:] instead
				ss.Events = ss.Events[len(ss.Events)-MaximumNumberOfEvents:]
			}
		}
	}
	ss.Results = append(ss.Results, result)
	if len(ss.Results) > MaximumNumberOfResults {
		// Doing ss.Results[1:] would usually be sufficient, but in the case where for some reason, the slice has more
		// than one extra element, we can get rid of all of them at once and thus returning the slice to a length of
		// MaximumNumberOfResults by using ss.Results[len(ss.Results)-MaximumNumberOfResults:] instead
		ss.Results = ss.Results[len(ss.Results)-MaximumNumberOfResults:]
	}
	ss.Uptime.ProcessResult(result)
}
