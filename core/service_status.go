package core

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
