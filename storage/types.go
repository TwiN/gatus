package storage

import (
	"time"

	"github.com/TwinProduction/gatus/core"
	"gorm.io/gorm"
)

type service struct {
	gorm.Model
	Name    string
	Group   string
	Results []result `gorm:"foreignKey:ServiceID"`
}

type result struct {
	gorm.Model
	HTTPStatus       int
	Hostname         string
	Duration         time.Duration
	Errors           []evaluationError `gorm:"foreignKey:ResultID"`
	ConditionResults []conditionResult `gorm:"foreignKey:ResultID"`
	Success          bool
	Timestamp        time.Time
	ServiceID        uint
}

type evaluationError struct {
	gorm.Model
	Message  string
	ResultID uint
}

type conditionResult struct {
	gorm.Model
	Condition string
	Success   bool
	ResultID  uint
}

// ConvertFromStorage converts storage structs to core structs
func ConvertFromStorage(res result) core.Result {
	errors := make([]string, len(res.Errors))
	for i, err := range res.Errors {
		errors[i] = err.Message
	}

	crs := make([]*core.ConditionResult, len(res.ConditionResults))
	for i, cr := range res.ConditionResults {
		crs[i] = &core.ConditionResult{
			Condition: cr.Condition,
			Success:   cr.Success,
		}
	}

	return core.Result{
		HTTPStatus:       res.HTTPStatus,
		Body:             nil,
		Hostname:         res.Hostname,
		IP:               "",
		Connected:        false,
		Duration:         res.Duration,
		Errors:           errors,
		ConditionResults: crs,
		Success:          res.Success,
		Timestamp:        res.Timestamp,
	}
}

// ConvertToStorage converts core structs to storage structs
func ConvertToStorage(res core.Result) result {
	errors := make([]evaluationError, len(res.Errors))
	for i, err := range res.Errors {
		errors[i] = evaluationError{
			Message: err,
		}
	}

	crs := make([]conditionResult, len(res.ConditionResults))
	for i, cr := range res.ConditionResults {
		crs[i] = conditionResult{
			Condition: cr.Condition,
			Success:   cr.Success,
		}
	}

	return result{
		HTTPStatus:       res.HTTPStatus,
		Hostname:         res.Hostname,
		Duration:         res.Duration,
		Errors:           errors,
		ConditionResults: crs,
		Success:          res.Success,
		Timestamp:        res.Timestamp,
	}
}
