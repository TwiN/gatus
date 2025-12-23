package state

import (
	"errors"
	"math"
)

const (
	DefaultHealthyStateName     = "healthy"
	DefaultUnhealthyStateName   = "unhealthy"
	DefaultMaintenanceStateName = "maintenance"
)

var (
	ErrInvalidName     = errors.New("invalid name: must be non-empty and contain only lowercase letters, digits, hyphens, or underscores")
	ErrInvalidPriority = errors.New("invalid priority: must be non-negative")
)

type State struct {
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
	if err := ValidateName(cfg.Name); err != nil {
		return err
	}
	if cfg.Priority < 0 {
		return ErrInvalidPriority
	}
	return nil
}

func ValidateName(name string) error {
	if len(name) == 0 {
		return ErrInvalidName
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return ErrInvalidName
		}
	}
	return nil
}
