package core

// ConditionResult result of a Condition
type ConditionResult struct {
	// Condition that was evaluated
	Condition string `json:"condition"`

	// Success whether the condition was met (successful) or not (failed)
	Success bool `json:"success"`

	// Severity of condition, evaluated during failure case only
	SeverityStatus SeverityStatus `json:"-"`
}
