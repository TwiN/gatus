package threemagateway

import (
	"encoding"
	"errors"
	"fmt"
)

// TODO#1464: Add tests

const (
	defaultMode = "basic" // TODO#1464: Should the default be e2ee event though it is not implemented yet to avoid future breaking or bad default behavior?
)

var (
	ErrModeTypeInvalid    = fmt.Errorf("invalid mode, must be one of: %s", joinKeys(validModeTypes, ", "))
	ErrNotImplementedMode = errors.New("configured mode is not implemented")
	validModeTypes        = map[string]ModeType{
		"basic":     ModeTypeBasic,
		"e2ee":      ModeTypeE2EE,
		"e2ee-bulk": ModeTypeE2EEBulk,
	}
)

type ModeType int

const (
	ModeTypeInvalid ModeType = iota
	ModeTypeBasic
	ModeTypeE2EE
	ModeTypeE2EEBulk
)

type SendMode struct {
	Value string   `yaml:"-"`
	Type  ModeType `yaml:"-"`
}

var _ encoding.TextUnmarshaler = (*SendMode)(nil)
var _ encoding.TextMarshaler = (*SendMode)(nil)

func (m *SendMode) UnmarshalText(text []byte) error {
	t := string(text)
	if len(t) == 0 {
		t = defaultMode
	}
	m.Value = t
	if val, ok := validModeTypes[t]; ok {
		m.Type = val
		return nil
	}
	m.Type = ModeTypeInvalid
	return ErrModeTypeInvalid
}

func (m SendMode) MarshalText() ([]byte, error) {
	return []byte(m.Value), nil
}
