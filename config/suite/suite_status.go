package suite

// Status represents the status of a suite
type Status struct {
	// Name of the suite
	Name string `json:"name,omitempty"`

	// Groups the suite is a part of. Used for grouping multiple suites together on the front end.
	Groups []string `json:"groups,omitempty"`

	// Key of the Suite
	Key string `json:"key"`

	// Results is the list of suite execution results
	Results []*Result `json:"results"`
}

// NewStatus creates a new Status for a given Suite
func NewStatus(s *Suite) *Status {
	return &Status{
		Name:    s.Name,
		Groups:  s.Groups,
		Key:     s.Key(),
		Results: []*Result{},
	}
}