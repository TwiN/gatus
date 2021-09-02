package core

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
	Events []*Event `json:"events"`

	// Uptime information on the service's uptime
	//
	// Used by the memory store.
	//
	// To retrieve the uptime between two time, use store.GetUptimeByKey.
	Uptime *Uptime `json:"-"`
}

// NewServiceStatus creates a new ServiceStatus
func NewServiceStatus(serviceKey, serviceGroup, serviceName string) *ServiceStatus {
	return &ServiceStatus{
		Name:    serviceName,
		Group:   serviceGroup,
		Key:     serviceKey,
		Results: make([]*Result, 0),
		Events:  make([]*Event, 0),
		Uptime:  NewUptime(),
	}
}
