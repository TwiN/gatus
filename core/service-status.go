package core

import "github.com/TwinProduction/gatus/util"

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

	// Uptime information on the service's uptime
	Uptime *Uptime `json:"uptime"`
}

// NewServiceStatus creates a new ServiceStatus
func NewServiceStatus(service *Service) *ServiceStatus {
	return &ServiceStatus{
		Name:    service.Name,
		Group:   service.Group,
		Key:     util.ConvertGroupAndServiceToKey(service.Group, service.Name),
		Results: make([]*Result, 0),
		Uptime:  NewUptime(),
	}
}

// AddResult adds a Result to ServiceStatus.Results and makes sure that there are
// no more than 20 results in the Results slice
func (ss *ServiceStatus) AddResult(result *Result) {
	ss.Results = append(ss.Results, result)
	if len(ss.Results) > 20 {
		ss.Results = ss.Results[1:]
	}
	ss.Uptime.ProcessResult(result)
}
