package gontext

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrGontextPathNotFound is returned when a gontext path doesn't exist
	ErrGontextPathNotFound = errors.New("gontext path not found")
)

// Gontext holds values that can be shared between endpoints in a suite
type Gontext struct {
	mu     sync.RWMutex
	values map[string]interface{}
}

// New creates a new gontext with initial values
func New(initial map[string]interface{}) *Gontext {
	if initial == nil {
		initial = make(map[string]interface{})
	}
	// Create a deep copy to avoid external modifications
	values := make(map[string]interface{})
	for k, v := range initial {
		values[k] = deepCopyValue(v)
	}
	return &Gontext{
		values: values,
	}
}

// Get retrieves a value from the gontext using dot notation
func (g *Gontext) Get(path string) (interface{}, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	parts := strings.Split(path, ".")
	current := interface{}(g.values)
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, exists := v[part]
			if !exists {
				return nil, fmt.Errorf("%w: %s", ErrGontextPathNotFound, path)
			}
			current = val
		default:
			return nil, fmt.Errorf("%w: %s", ErrGontextPathNotFound, path)
		}
	}
	return current, nil
}

// Set stores a value in the gontext using dot notation
func (g *Gontext) Set(path string, value interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return errors.New("empty path")
	}
	// Navigate to the parent of the target
	current := g.values
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// Path exists but is not a map, create a new map
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			}
		} else {
			// Create intermediate maps
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}
	// Set the final value
	current[parts[len(parts)-1]] = value
	return nil
}

// GetAll returns a copy of all gontext values
func (g *Gontext) GetAll() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range g.values {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue creates a deep copy of a value
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, v := range val {
			newMap[k] = deepCopyValue(v)
		}
		return newMap
	case []interface{}:
		newSlice := make([]interface{}, len(val))
		for i, v := range val {
			newSlice[i] = deepCopyValue(v)
		}
		return newSlice
	default:
		// For primitive types, return as-is (they're passed by value anyway)
		return val
	}
}
