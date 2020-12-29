package storage

import (
	"time"

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
