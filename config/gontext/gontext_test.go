package gontext

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		initial  map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil-input",
			initial:  nil,
			expected: make(map[string]interface{}),
		},
		{
			name:     "empty-input",
			initial:  make(map[string]interface{}),
			expected: make(map[string]interface{}),
		},
		{
			name: "simple-values",
			initial: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		},
		{
			name: "nested-values",
			initial: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   123,
					"name": "John Doe",
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   123,
					"name": "John Doe",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := New(tt.initial)
			if ctx == nil {
				t.Error("Expected non-nil gontext")
			}
			if ctx.values == nil {
				t.Error("Expected non-nil values map")
			}

			// Verify deep copy by modifying original
			if tt.initial != nil {
				tt.initial["modified"] = "should not appear"
				if _, exists := ctx.values["modified"]; exists {
					t.Error("Deep copy failed - original map modification affected gontext")
				}
			}
		})
	}
}

func TestGontext_Get(t *testing.T) {
	ctx := New(map[string]interface{}{
		"simple":  "value",
		"number":  42,
		"boolean": true,
		"nested": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": "deep_value",
			},
		},
		"user": map[string]interface{}{
			"id":   123,
			"name": "John",
			"profile": map[string]interface{}{
				"email": "john@example.com",
			},
		},
	})

	tests := []struct {
		name        string
		path        string
		expected    interface{}
		shouldError bool
		errorType   error
	}{
		{
			name:        "simple-value",
			path:        "simple",
			expected:    "value",
			shouldError: false,
		},
		{
			name:        "number-value",
			path:        "number",
			expected:    42,
			shouldError: false,
		},
		{
			name:        "boolean-value",
			path:        "boolean",
			expected:    true,
			shouldError: false,
		},
		{
			name:        "nested-value",
			path:        "nested.level1.level2",
			expected:    "deep_value",
			shouldError: false,
		},
		{
			name:        "user-id",
			path:        "user.id",
			expected:    123,
			shouldError: false,
		},
		{
			name:        "deep-nested-value",
			path:        "user.profile.email",
			expected:    "john@example.com",
			shouldError: false,
		},
		{
			name:        "non-existent-key",
			path:        "nonexistent",
			expected:    nil,
			shouldError: true,
			errorType:   ErrGontextPathNotFound,
		},
		{
			name:        "non-existent-nested-key",
			path:        "user.nonexistent",
			expected:    nil,
			shouldError: true,
			errorType:   ErrGontextPathNotFound,
		},
		{
			name:        "invalid-nested-path",
			path:        "simple.invalid",
			expected:    nil,
			shouldError: true,
			errorType:   ErrGontextPathNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ctx.Get(tt.path)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("Expected error type %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestGontext_Set(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "simple-set",
			path:    "key",
			value:   "value",
			wantErr: false,
		},
		{
			name:    "nested-set",
			path:    "user.name",
			value:   "John Doe",
			wantErr: false,
		},
		{
			name:    "deep-nested-set",
			path:    "user.profile.email",
			value:   "john@example.com",
			wantErr: false,
		},
		{
			name:    "override-primitive-with-nested",
			path:    "existing.new",
			value:   "nested_value",
			wantErr: false,
		},
		{
			name:    "empty-path",
			path:    "",
			value:   "value",
			wantErr: false, // Actually, empty string creates a single part [""], which is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := New(map[string]interface{}{
				"existing": "primitive",
			})

			err := ctx.Set(tt.path, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify the value was set correctly
			result, getErr := ctx.Get(tt.path)
			if getErr != nil {
				t.Errorf("Error retrieving set value: %v", getErr)
				return
			}

			if result != tt.value {
				t.Errorf("Expected %v, got %v", tt.value, result)
			}
		})
	}
}

func TestGontext_SetOverrideBehavior(t *testing.T) {
	ctx := New(map[string]interface{}{
		"primitive": "value",
		"nested": map[string]interface{}{
			"key": "existing",
		},
	})

	// Test overriding primitive with nested structure
	err := ctx.Set("primitive.new", "nested_value")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify the primitive was replaced with a nested structure
	result, err := ctx.Get("primitive.new")
	if err != nil {
		t.Errorf("Error getting nested value: %v", err)
	}
	if result != "nested_value" {
		t.Errorf("Expected 'nested_value', got %v", result)
	}

	// Test overriding existing nested value
	err = ctx.Set("nested.key", "modified")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	result, err = ctx.Get("nested.key")
	if err != nil {
		t.Errorf("Error getting modified value: %v", err)
	}
	if result != "modified" {
		t.Errorf("Expected 'modified', got %v", result)
	}
}

func TestGontext_GetAll(t *testing.T) {
	initial := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"nested": map[string]interface{}{
			"inner": "value",
		},
	}

	ctx := New(initial)

	// Add another value after creation
	ctx.Set("key3", "value3")

	result := ctx.GetAll()

	// Verify all values are present
	if result["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", result["key1"])
	}
	if result["key2"] != 42 {
		t.Errorf("Expected key2=42, got %v", result["key2"])
	}
	if result["key3"] != "value3" {
		t.Errorf("Expected key3=value3, got %v", result["key3"])
	}

	// Verify nested values
	nested, ok := result["nested"].(map[string]interface{})
	if !ok {
		t.Error("Expected nested to be map[string]interface{}")
	} else if nested["inner"] != "value" {
		t.Errorf("Expected nested.inner=value, got %v", nested["inner"])
	}

	// Verify deep copy - modifying returned map shouldn't affect gontext
	result["key1"] = "modified"
	original, _ := ctx.Get("key1")
	if original != "value1" {
		t.Error("GetAll did not return a deep copy - modification affected original")
	}
}

func TestGontext_ConcurrentAccess(t *testing.T) {
	ctx := New(map[string]interface{}{
		"counter": 0,
	})

	done := make(chan bool, 10)

	// Start 5 goroutines that read values
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				_, err := ctx.Get("counter")
				if err != nil {
					t.Errorf("Reader %d error: %v", id, err)
				}
			}
			done <- true
		}(i)
	}

	// Start 5 goroutines that write values
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				err := ctx.Set("counter", id*1000+j)
				if err != nil {
					t.Errorf("Writer %d error: %v", id, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDeepCopyValue(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "primitive-string",
			input: "test",
		},
		{
			name:  "primitive-int",
			input: 42,
		},
		{
			name:  "primitive-bool",
			input: true,
		},
		{
			name: "simple-map",
			input: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "nested-map",
			input: map[string]interface{}{
				"nested": map[string]interface{}{
					"deep": "value",
				},
			},
		},
		{
			name:  "simple-slice",
			input: []interface{}{"a", "b", "c"},
		},
		{
			name: "mixed-slice",
			input: []interface{}{
				"string",
				42,
				map[string]interface{}{"nested": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepCopyValue(tt.input)

			// For maps and slices, verify it's a different object
			switch v := tt.input.(type) {
			case map[string]interface{}:
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Error("Deep copy didn't preserve map type")
					return
				}
				// Modify original to ensure independence
				v["modified"] = "test"
				if _, exists := resultMap["modified"]; exists {
					t.Error("Deep copy failed - maps are not independent")
				}
			case []interface{}:
				resultSlice, ok := result.([]interface{})
				if !ok {
					t.Error("Deep copy didn't preserve slice type")
					return
				}
				if len(resultSlice) != len(v) {
					t.Error("Deep copy didn't preserve slice length")
				}
			}
		})
	}
}
