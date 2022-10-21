package jsonpath

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Eval is a half-baked json path implementation that needs some love
func Eval(path string, b []byte) (string, int, error) {
	var object interface{}
	err := json.Unmarshal(b, &object)
	if err != nil {
		// Try to unmarshal it into an array instead
		return "", 0, err
	}
	return walk(path, object)
}

func walk(path string, object interface{}) (string, int, error) {
	var keys []string
	startOfCurrentKey, bracketDepth := 0, 0
	for i := range path {
		if path[i] == '[' {
			bracketDepth++
		} else if path[i] == ']' {
			bracketDepth--
		}
		// If we encounter a dot, we've reached the end of a key unless we're inside a bracket
		if path[i] == '.' && bracketDepth == 0 {
			keys = append(keys, path[startOfCurrentKey:i])
			startOfCurrentKey = i + 1
		}
	}
	if startOfCurrentKey <= len(path) {
		keys = append(keys, path[startOfCurrentKey:])
	}
	currentKey := keys[0]
	switch value := extractValue(currentKey, object).(type) {
	case map[string]interface{}:
		return walk(strings.Replace(path, fmt.Sprintf("%s.", currentKey), "", 1), value)
	case string:
		if len(keys) > 1 {
			return "", 0, fmt.Errorf("couldn't walk through '%s', because '%s' was a string instead of an object", keys[1], currentKey)
		}
		return value, len(value), nil
	case []interface{}:
		return fmt.Sprintf("%v", value), len(value), nil
	case interface{}:
		return fmt.Sprintf("%v", value), 1, nil
	default:
		return "", 0, fmt.Errorf("couldn't walk through '%s' because type was '%T', but expected 'map[string]interface{}'", currentKey, value)
	}
}

func extractValue(currentKey string, value interface{}) interface{} {
	// Check if the current key ends with [#]
	if strings.HasSuffix(currentKey, "]") && strings.Contains(currentKey, "[") {
		var isNestedArray bool
		var index string
		startOfBracket, endOfBracket, bracketDepth := 0, 0, 0
		for i := range currentKey {
			if currentKey[i] == '[' {
				startOfBracket = i
				bracketDepth++
			} else if currentKey[i] == ']' && bracketDepth == 1 {
				bracketDepth--
				endOfBracket = i
				index = currentKey[startOfBracket+1 : i]
				if len(currentKey) > i+1 && currentKey[i+1] == '[' {
					isNestedArray = true // there's more keys.
				}
				break
			}
		}
		arrayIndex, err := strconv.Atoi(index)
		if err != nil {
			return nil
		}
		currentKeyWithoutIndex := currentKey[:startOfBracket]
		// if currentKeyWithoutIndex contains only an index (i.e. [0] or 0)
		if len(currentKeyWithoutIndex) == 0 {
			array := value.([]interface{})
			if len(array) > arrayIndex {
				if isNestedArray {
					return extractValue(currentKey[endOfBracket+1:], array[arrayIndex])
				}
				return array[arrayIndex]
			}
			return nil
		}
		if value == nil || value.(map[string]interface{})[currentKeyWithoutIndex] == nil {
			return nil
		}
		// if currentKeyWithoutIndex contains both a key and an index (i.e. data[0])
		array := value.(map[string]interface{})[currentKeyWithoutIndex].([]interface{})
		if len(array) > arrayIndex {
			if isNestedArray {
				return extractValue(currentKey[endOfBracket+1:], array[arrayIndex])
			}
			return array[arrayIndex]
		}
		return nil
	}
	return value.(map[string]interface{})[currentKey]
}
