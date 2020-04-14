package core

import (
	"time"
)

type HealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Result struct {
	HttpStatus       int                `json:"status"`
	Body             []byte             `json:"-"`
	Hostname         string             `json:"hostname"`
	Ip               string             `json:"-"`
	Duration         time.Duration      `json:"duration"`
	Errors           []string           `json:"errors"`
	ConditionResults []*ConditionResult `json:"condition-results"`
	Success          bool               `json:"success"`
	Timestamp        time.Time          `json:"timestamp"`
}

type ConditionResult struct {
	Condition string `json:"condition"`
	Success   bool   `json:"success"`
}
