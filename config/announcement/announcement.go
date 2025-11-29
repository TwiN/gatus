package announcement

import (
	"errors"
	"sort"
	"time"
)

const (
	// TypeOutage represents a service outage
	TypeOutage = "outage"

	// TypeWarning represents a warning or potential issue
	TypeWarning = "warning"

	// TypeInformation represents general information
	TypeInformation = "information"

	// TypeOperational represents operational status or resolved issues
	TypeOperational = "operational"

	// TypeNone represents no specific type (default)
	TypeNone = "none"
)

var (
	// ErrInvalidAnnouncementType is returned when an invalid announcement type is specified
	ErrInvalidAnnouncementType = errors.New("invalid announcement type")

	// ErrEmptyMessage is returned when an announcement has an empty message
	ErrEmptyMessage = errors.New("announcement message cannot be empty")

	// ErrMissingTimestamp is returned when an announcement has an empty timestamp
	ErrMissingTimestamp = errors.New("announcement timestamp must be set")

	// validTypes contains all valid announcement types
	validTypes = map[string]bool{
		TypeOutage:      true,
		TypeWarning:     true,
		TypeInformation: true,
		TypeOperational: true,
		TypeNone:        true,
	}
)

// Announcement represents a system-wide announcement
type Announcement struct {
	// Timestamp is the UTC timestamp when the announcement was made
	Timestamp time.Time `yaml:"timestamp" json:"timestamp"`

	// Type is the type of announcement (outage, warning, information, operational, none)
	Type string `yaml:"type" json:"type"`

	// Message is the user-facing text describing the announcement
	Message string `yaml:"message" json:"message"`

	// Archived indicates whether the announcement should be displayed in the historical section
	// instead of at the top of the status page
	Archived bool `yaml:"archived,omitempty" json:"archived,omitempty"`
}

// ValidateAndSetDefaults validates the announcement and sets default values if necessary
func (a *Announcement) ValidateAndSetDefaults() error {
	// Validate message
	if a.Message == "" {
		return ErrEmptyMessage
	}
	// Set default type if empty
	if a.Type == "" {
		a.Type = TypeNone
	}
	// Validate type
	if !validTypes[a.Type] {
		return ErrInvalidAnnouncementType
	}
	// If timestamp is zero, return an error
	if a.Timestamp.IsZero() {
		return ErrMissingTimestamp
	}
	return nil
}

// SortByTimestamp sorts a slice of announcements by timestamp in descending order (newest first)
func SortByTimestamp(announcements []*Announcement) {
	sort.Slice(announcements, func(i, j int) bool {
		return announcements[i].Timestamp.After(announcements[j].Timestamp)
	})
}

// ValidateAndSetDefaults validates a slice of announcements and sets defaults
func ValidateAndSetDefaults(announcements []*Announcement) error {
	for _, announcement := range announcements {
		if err := announcement.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}
