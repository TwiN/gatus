package threemagateway

import (
	"errors"
	"maps"
	"slices"
	"strings"
)

var (
	ErrInvalidThreemaId = errors.New("Must be 8 characters long and alphabetic characters must be uppercase")
)

func joinKeys[V any](m map[string]V, separator string) string {
	return strings.Join(slices.Collect(maps.Keys(m)), separator)
}

func validateThreemaId(id string) error {
	if len(id) != 8 || strings.ToUpper(id) != id {
		return ErrInvalidThreemaId
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
