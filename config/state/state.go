package state

import (
	"errors"
	"math"
)

// TODO#227 Add tests

const (
	DefaultHealthyStateName     = "healthy"
	DefaultUnhealthyStateName   = "unhealthy"
	DefaultMaintenanceStateName = "maintenance"
)

var (
	ErrInvalidName     = errors.New("invalid name: must be non-empty")
	ErrInvalidPriority = errors.New("invalid priority: must be non-negative")
)

type State struct { // TODO#227 Add label or description fields? Derive label in frontend from name and only set desciption?
	Name     string `yaml:"name"`
	Priority int    `yaml:"priority"`
}

func GetDefaultConfig() []*State {
	return []*State{
		{
			Name:     DefaultHealthyStateName,
			Priority: 0,
		},
		{
			Name:     DefaultUnhealthyStateName,
			Priority: math.MaxInt - 1,
		},
		{
			Name:     DefaultMaintenanceStateName,
			Priority: math.MaxInt,
		},
	}
}

func (cfg *State) ValidateAndSetDefaults() error {
	if len(cfg.Name) == 0 { // TODO#227 more robust name validation
		return ErrInvalidName
	}
	if cfg.Priority < 0 {
		return ErrInvalidPriority
	}
	return nil
}
