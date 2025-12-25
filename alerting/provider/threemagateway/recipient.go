package threemagateway

import (
	"encoding"
	"errors"
	"fmt"
	"strings"
)

const (
	defaultRecipientType = RecipientTypeId
)

var (
	ErrInvalidRecipientFormat = errors.New("recipient must be in the format '[<type>:]<value>'")
	ErrInvalidRecipientType   = fmt.Errorf("invalid recipient type, must be one of: %v", joinKeys(validRecipientTypes, ", "))
	validRecipientTypes       = map[string]RecipientType{
		"id":    RecipientTypeId,
		"phone": RecipientTypePhone,
		"email": RecipientTypeEmail,
	}

	ErrInvalidPhoneNumberFormat  = errors.New("invalid phone number: must contain only digits and may start with '+'")
	ErrInvalidEmailAddressFormat = errors.New("invalid email address: must contain '@'")
)

type RecipientType int

const (
	RecipientTypeInvalid RecipientType = iota
	RecipientTypeId
	RecipientTypePhone
	RecipientTypeEmail
)

func parseRecipientType(s string) RecipientType {
	if val, ok := validRecipientTypes[s]; ok {
		return val
	}
	return RecipientTypeInvalid
}

type Recipient struct {
	Value string        `yaml:"-"`
	Type  RecipientType `yaml:"-"`
}

var _ encoding.TextUnmarshaler = (*Recipient)(nil)
var _ encoding.TextMarshaler = (*Recipient)(nil)

func (r *Recipient) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), ":")
	switch {
	case len(parts) > 2:
		return ErrInvalidRecipientFormat
	case len(parts) == 2:
		if r.Type = parseRecipientType(parts[0]); r.Type == RecipientTypeInvalid {
			return ErrInvalidRecipientType
		}
		r.Value = parts[1]
	default:
		r.Type = defaultRecipientType
		r.Value = parts[0]
	}
	return nil
}

func (r Recipient) MarshalText() ([]byte, error) {
	if r.Type == RecipientTypeInvalid {
		return []byte("invalid" + ":" + r.Value), nil
	}
	for key, val := range validRecipientTypes {
		if val == r.Type {
			return []byte(key + ":" + r.Value), nil
		}
	}
	return nil, ErrInvalidRecipientType
}

func (r *Recipient) Validate() error {
	if len(r.Value) == 0 {
		return ErrInvalidRecipientFormat
	}
	switch r.Type {
	case RecipientTypeId:
		if err := validateThreemaId(r.Value); err != nil {
			return err
		}
	case RecipientTypePhone:
		r.Value = strings.TrimPrefix(r.Value, "+")
		if !isValidPhoneNumber(r.Value) {
			return ErrInvalidPhoneNumberFormat
		}
	case RecipientTypeEmail:
		// Basic validation for email address // TODO#1464: improve email validation
		if !strings.Contains(r.Value, "@") {
			return ErrInvalidEmailAddressFormat
		}
	default:
		return ErrInvalidRecipientType
	}
	return nil
}

func isValidPhoneNumber(number string) bool {
	if len(number) == 0 {
		return false
	}
	for _, ch := range number {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
