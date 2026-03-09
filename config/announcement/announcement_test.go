package announcement

import (
	"errors"
	"testing"
	"time"
)

func TestAnnouncement_ValidateAndSetDefaults(t *testing.T) {
	now := time.Now()
	scenarios := []struct {
		name          string
		announcement  *Announcement
		expectedError error
		expectedType  string
	}{
		{
			name: "valid-announcement-with-all-fields",
			announcement: &Announcement{
				Timestamp: now,
				Type:      TypeWarning,
				Message:   "This is a test announcement",
				Archived:  false,
			},
			expectedError: nil,
			expectedType:  TypeWarning,
		},
		{
			name: "valid-announcement-with-archived-true",
			announcement: &Announcement{
				Timestamp: now,
				Type:      TypeOperational,
				Message:   "This is an archived announcement",
				Archived:  true,
			},
			expectedError: nil,
			expectedType:  TypeOperational,
		},
		{
			name: "valid-announcement-with-empty-type-should-default-to-none",
			announcement: &Announcement{
				Timestamp: now,
				Message:   "This announcement has no type",
			},
			expectedError: nil,
			expectedType:  TypeNone,
		},
		{
			name: "invalid-announcement-with-empty-message",
			announcement: &Announcement{
				Timestamp: now,
				Type:      TypeWarning,
				Message:   "",
			},
			expectedError: ErrEmptyMessage,
		},
		{
			name: "invalid-announcement-with-zero-timestamp",
			announcement: &Announcement{
				Timestamp: time.Time{},
				Type:      TypeWarning,
				Message:   "Test message",
			},
			expectedError: ErrMissingTimestamp,
		},
		{
			name: "invalid-announcement-with-invalid-type",
			announcement: &Announcement{
				Timestamp: now,
				Type:      "invalid-type",
				Message:   "Test message",
			},
			expectedError: ErrInvalidAnnouncementType,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.announcement.ValidateAndSetDefaults()
			if !errors.Is(err, scenario.expectedError) {
				t.Errorf("expected error %v, got %v", scenario.expectedError, err)
			}
			if scenario.expectedError == nil && scenario.announcement.Type != scenario.expectedType {
				t.Errorf("expected type %s, got %s", scenario.expectedType, scenario.announcement.Type)
			}
		})
	}
}

func TestAnnouncement_ValidateAndSetDefaults_AllTypes(t *testing.T) {
	now := time.Now()
	validTypes := []string{TypeOutage, TypeWarning, TypeInformation, TypeOperational, TypeNone}
	for _, validType := range validTypes {
		t.Run("type-"+validType, func(t *testing.T) {
			announcement := &Announcement{
				Timestamp: now,
				Type:      validType,
				Message:   "Test message",
			}
			if err := announcement.ValidateAndSetDefaults(); err != nil {
				t.Errorf("expected no error for type %s, got %v", validType, err)
			}
			if announcement.Type != validType {
				t.Errorf("expected type %s, got %s", validType, announcement.Type)
			}
		})
	}
}

func TestSortByTimestamp(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)
	announcements := []*Announcement{
		{Timestamp: now, Message: "now"},
		{Timestamp: later, Message: "later"},
		{Timestamp: earlier, Message: "earlier"},
	}
	SortByTimestamp(announcements)
	if announcements[0].Timestamp != later {
		t.Error("expected first announcement to be the latest")
	}
	if announcements[1].Timestamp != now {
		t.Error("expected second announcement to be the middle one")
	}
	if announcements[2].Timestamp != earlier {
		t.Error("expected third announcement to be the earliest")
	}
}

func TestSortByTimestamp_WithArchivedField(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)
	announcements := []*Announcement{
		{Timestamp: now, Message: "now", Archived: false},
		{Timestamp: later, Message: "later", Archived: true},
		{Timestamp: earlier, Message: "earlier", Archived: false},
	}
	SortByTimestamp(announcements)
	// Sorting should be by timestamp only, not affected by archived status
	if announcements[0].Timestamp != later {
		t.Error("expected first announcement to be the latest, regardless of archived status")
	}
	if !announcements[0].Archived {
		t.Error("expected first announcement to be archived")
	}
	if announcements[1].Timestamp != now {
		t.Error("expected second announcement to be the middle one")
	}
	if announcements[2].Timestamp != earlier {
		t.Error("expected third announcement to be the earliest")
	}
}

func TestValidateAndSetDefaults_Slice(t *testing.T) {
	now := time.Now()
	scenarios := []struct {
		name           string
		announcements  []*Announcement
		expectedError  error
		shouldValidate bool
	}{
		{
			name: "all-valid-announcements",
			announcements: []*Announcement{
				{Timestamp: now, Type: TypeWarning, Message: "First announcement"},
				{Timestamp: now, Type: TypeOperational, Message: "Second announcement"},
			},
			expectedError:  nil,
			shouldValidate: true,
		},
		{
			name: "mixed-archived-announcements",
			announcements: []*Announcement{
				{Timestamp: now, Type: TypeWarning, Message: "Active announcement", Archived: false},
				{Timestamp: now, Type: TypeOperational, Message: "Archived announcement", Archived: true},
			},
			expectedError:  nil,
			shouldValidate: true,
		},
		{
			name: "one-invalid-announcement-in-slice",
			announcements: []*Announcement{
				{Timestamp: now, Type: TypeWarning, Message: "Valid announcement"},
				{Timestamp: now, Type: TypeOperational, Message: ""},
			},
			expectedError:  ErrEmptyMessage,
			shouldValidate: false,
		},
		{
			name: "announcement-with-missing-timestamp",
			announcements: []*Announcement{
				{Timestamp: now, Type: TypeWarning, Message: "Valid announcement"},
				{Timestamp: time.Time{}, Type: TypeOperational, Message: "Invalid announcement"},
			},
			expectedError:  ErrMissingTimestamp,
			shouldValidate: false,
		},
		{
			name: "announcement-with-invalid-type",
			announcements: []*Announcement{
				{Timestamp: now, Type: TypeWarning, Message: "Valid announcement"},
				{Timestamp: now, Type: "bad-type", Message: "Invalid announcement"},
			},
			expectedError:  ErrInvalidAnnouncementType,
			shouldValidate: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := ValidateAndSetDefaults(scenario.announcements)
			if !errors.Is(err, scenario.expectedError) {
				t.Errorf("expected error %v, got %v", scenario.expectedError, err)
			}
		})
	}
}

func TestAnnouncement_ArchivedFieldDefaults(t *testing.T) {
	now := time.Now()
	announcement := &Announcement{
		Timestamp: now,
		Type:      TypeWarning,
		Message:   "Test announcement",
		// Archived not set, should default to false
	}
	if err := announcement.ValidateAndSetDefaults(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// Zero value for bool is false
	if announcement.Archived {
		t.Error("expected Archived to default to false")
	}
}

func TestValidateAndSetDefaults_EmptySlice(t *testing.T) {
	announcements := []*Announcement{}
	if err := ValidateAndSetDefaults(announcements); err != nil {
		t.Errorf("expected no error for empty slice, got %v", err)
	}
}
