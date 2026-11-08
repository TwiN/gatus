package jsonpath

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Eval is a half-baked json path implementation that needs some love
//
// Keys are separated by dots (i.e. data.name). Array elements are addressed by their index
// between brackets (i.e. data[0].name). Object keys that contain dots must be wrapped in
// single or double quotes between brackets (i.e. data["name.with.dots"]), because an unquoted
// dot is always treated as a key separator.
func Eval(path string, b []byte) (string, int, error) {
	if len(path) == 0 && !(len(b) != 0 && b[0] == '[' && b[len(b)-1] == ']') {
		// if there's no path AND the value is not a JSON array, then there's nothing to walk
		return string(b), len(b), nil
	}
	var object interface{}
	if err := json.Unmarshal(b, &object); err != nil {
		return "", 0, err
	}
	return walk(path, object)
}

// walk traverses the object and returns the value as a string as well as its length
func walk(path string, object interface{}) (string, int, error) {
	var keys []string
	startOfCurrentKey, bracketDepth := 0, 0
	var quoteCharacter byte
	for i := range path {
		// If we're inside a quoted accessor, everything up to the closing quote is part of the key
		if quoteCharacter != 0 {
			if path[i] == quoteCharacter {
				quoteCharacter = 0
			}
			continue
		}
		if (path[i] == '"' || path[i] == '\'') && bracketDepth > 0 {
			quoteCharacter = path[i]
			continue
		}
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
		newPath := strings.Replace(path, fmt.Sprintf("%s.", currentKey), "", 1)
		if path == newPath {
			// If the path hasn't changed, it means we're at the end of the path
			// So we'll treat it as a string by re-marshaling it to JSON since it's a map.
			// Note that the output JSON will be minified.
			b, err := json.Marshal(value)
			return string(b), len(b), err
		}
		return walk(newPath, value)
	case string:
		if len(keys) > 1 {
			return "", 0, fmt.Errorf("couldn't walk through '%s', because '%s' was a string instead of an object", keys[1], currentKey)
		}
		return value, len(value), nil
	case []interface{}:
		return fmt.Sprintf("%v", value), len(value), nil
	case interface{}:
		newValue := fmt.Sprintf("%v", value)
		return newValue, len(newValue), nil
	default:
		return "", 0, fmt.Errorf("couldn't walk through '%s' because type was '%T', but expected 'map[string]interface{}'", currentKey, value)
	}
}

func extractValue(currentKey string, value interface{}) interface{} {
	// Check if the current key has an accessor, i.e. [#] for an array index or ["key"] for an object key
	if strings.HasSuffix(currentKey, "]") && strings.Contains(currentKey, "[") {
		var hasMoreAccessors bool
		var accessor string
		startOfBracket, endOfBracket, bracketDepth := 0, 0, 0
		var quoteCharacter byte
		for i := range currentKey {
			// If we're inside a quoted accessor, everything up to the closing quote is part of the key
			if quoteCharacter != 0 {
				if currentKey[i] == quoteCharacter {
					quoteCharacter = 0
				}
				continue
			}
			if (currentKey[i] == '"' || currentKey[i] == '\'') && bracketDepth == 1 {
				quoteCharacter = currentKey[i]
			} else if currentKey[i] == '[' {
				startOfBracket = i
				bracketDepth++
			} else if currentKey[i] == ']' && bracketDepth == 1 {
				bracketDepth--
				endOfBracket = i
				accessor = currentKey[startOfBracket+1 : i]
				if len(currentKey) > i+1 && currentKey[i+1] == '[' {
					hasMoreAccessors = true // there's more accessors.
				}
				break
			}
		}
		// Resolve the value that the accessor applies to. If there's a key before the accessor
		// (i.e. data[0]), then the accessor applies to the value behind that key.
		target := value
		if keyBeforeAccessor := currentKey[:startOfBracket]; len(keyBeforeAccessor) > 0 {
			valueAsMap, ok := value.(map[string]interface{})
			if !ok || valueAsMap[keyBeforeAccessor] == nil {
				return nil
			}
			target = valueAsMap[keyBeforeAccessor]
		}
		var extractedValue interface{}
		if key, isQuoted := unquote(accessor); isQuoted {
			// A quoted accessor (i.e. ["key"]) is an object key rather than an array index.
			// This is the only way to address keys that contain dots, because an unquoted dot
			// is always treated as a key separator.
			valueAsMap, ok := target.(map[string]interface{})
			if !ok {
				return nil
			}
			extractedValue = valueAsMap[key]
		} else {
			arrayIndex, err := strconv.Atoi(accessor)
			if err != nil || arrayIndex < 0 {
				return nil
			}
			array, _ := target.([]interface{})
			if len(array) <= arrayIndex {
				return nil
			}
			extractedValue = array[arrayIndex]
		}
		if hasMoreAccessors {
			return extractValue(currentKey[endOfBracket+1:], extractedValue)
		}
		return extractedValue
	}
	if valueAsSlice, ok := value.([]interface{}); ok {
		// If the type is a slice, return it
		// This happens when the body (value) is a JSON array
		return valueAsSlice
	}
	if valueAsMap, ok := value.(map[string]interface{}); ok {
		// If the value is a map, then we get the currentKey from that map
		// This happens when the body (value) is a JSON object
		return valueAsMap[currentKey]
	}
	// If the value is neither a map, nor a slice, nor an index, then we cannot retrieve the currentKey
	// from said value. This usually happens when the body (value) is null.
	return value
}

// unquote removes the surrounding single or double quotes from s, if there are any.
// The second return value indicates whether s was quoted.
func unquote(s string) (string, bool) {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1], true
	}
	return s, false
}
