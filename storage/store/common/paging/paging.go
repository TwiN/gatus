package paging

// ServiceStatusParams represents all parameters that can be used for paging purposes
type ServiceStatusParams struct {
	EventsPage      int // Number of the event page
	EventsPageSize  int // Size of the event page
	ResultsPage     int // Number of the result page
	ResultsPageSize int // Size of the result page
}

// NewServiceStatusParams creates a new ServiceStatusParams
func NewServiceStatusParams() *ServiceStatusParams {
	return &ServiceStatusParams{}
}

// WithEvents sets the values for EventsPage and EventsPageSize
func (params *ServiceStatusParams) WithEvents(page, pageSize int) *ServiceStatusParams {
	params.EventsPage = page
	params.EventsPageSize = pageSize
	return params
}

// WithResults sets the values for ResultsPage and ResultsPageSize
func (params *ServiceStatusParams) WithResults(page, pageSize int) *ServiceStatusParams {
	params.ResultsPage = page
	params.ResultsPageSize = pageSize
	return params
}
