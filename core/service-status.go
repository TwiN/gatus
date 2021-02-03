package core

import (
	"time"

	"github.com/TwinProduction/gatus/util"
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
		Key:     util.ConvertGroupAndServiceToKey(service.Group, service.Name),
		Results: make([]*Result, 0),
		Events: []*Event{{
			Type:      EventStart,
			Timestamp: time.Now(),
		}},
		Uptime: NewUptime(),
	}
}

// AddResult adds a Result to ServiceStatus.Results and makes sure that there are
// no more than 20 results in the Results slice
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
			if len(ss.Events) > 20 {
				ss.Events = ss.Events[1:]
			}
		}
	}
	ss.Results = append(ss.Results, result)
	if len(ss.Results) > 20 {
		ss.Results = ss.Results[1:]
	}
	ss.Uptime.ProcessResult(result)
}
