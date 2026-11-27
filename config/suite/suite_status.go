package suite

// Status represents the status of a suite
type Status struct {
	// Name of the suite
	Name string `json:"name,omitempty"`

	// Group the suite is a part of. Used for grouping multiple suites together on the front end.
	Group string `json:"group,omitempty"`

	// Key of the Suite
	Key string `json:"key"`

	// Results is the list of suite execution results
	Results []*Result `json:"results"`

	Public bool `json:"public,omitempty"`
}

// NewStatus creates a new Status for a given Suite
func NewStatus(s *Suite) *Status {
	return &Status{
		Name:    s.Name,
		Group:   s.Group,
		Key:     s.Key(),
		Results: []*Result{},
		Public:  s.Visibility.Public,
	}
}

// SuiteStatusVisibility is a DTO used to manage a SuiteStatus visibility
type SuiteStatusVisibility struct {
	Key    string
	Public bool
}

func NewSuiteStatusVisibility(key string, public bool) *SuiteStatusVisibility {
	return &SuiteStatusVisibility{
		Key:    key,
		Public: public,
	}
}