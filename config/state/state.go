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
	defaultHealthy   = true
	defaultUnhealthy = false

	ErrInvalidName     = errors.New("invalid name: must be non-empty")
	ErrInvalidPriority = errors.New("invalid priority: must be non-negative")
)

type State struct { // TODO#227 Add label or description fields? Derive label in frontend from name and only set desciption?
	Name     string `yaml:"name"`
	Priority int    `yaml:"priority"`
	Success  *bool  `yaml:"success,omitempty"`
}

func GetDefaultConfig() []*State {
	return []*State{
		{
			Name:     DefaultHealthyStateName,
			Priority: 0,
			Success:  &defaultHealthy,
		},
		{
			Name:     DefaultUnhealthyStateName,
			Priority: math.MaxInt - 1,
			Success:  &defaultUnhealthy,
		},
		{
			Name:     DefaultMaintenanceStateName,
			Priority: math.MaxInt,
			Success:  &defaultUnhealthy, // TODO#227 Maybe make maintenance success configurable?
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
	if cfg.Success == nil {
		cfg.Success = &defaultUnhealthy
	}
	return nil
}
