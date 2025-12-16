package endpoint

// ConditionResult result of a Condition
type ConditionResult struct {
	// Condition that was evaluated
	Condition string `json:"condition"`

	// Success whether the condition was met (successful) or not (failed)
	Success bool `json:"success"`

	// The state the condition is linked to
	LinkedState string `json:"-"`
}
