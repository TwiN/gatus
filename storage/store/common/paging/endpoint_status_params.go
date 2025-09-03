package paging

// EndpointStatusParams represents all parameters that can be used for paging purposes
type EndpointStatusParams struct {
	EventsPage      int // Number of the event page
	EventsPageSize  int // Size of the event page
	ResultsPage     int // Number of the result page
	ResultsPageSize int // Size of the result page
}

// NewEndpointStatusParams creates a new EndpointStatusParams
func NewEndpointStatusParams() *EndpointStatusParams {
	return &EndpointStatusParams{}
}

// WithEvents sets the values for EventsPage and EventsPageSize
func (params *EndpointStatusParams) WithEvents(page, pageSize int) *EndpointStatusParams {
	params.EventsPage = page
	params.EventsPageSize = pageSize
	return params
}

// WithResults sets the values for ResultsPage and ResultsPageSize
func (params *EndpointStatusParams) WithResults(page, pageSize int) *EndpointStatusParams {
	params.ResultsPage = page
	params.ResultsPageSize = pageSize
	return params
}
