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
// single or double quotes, either on their own (i.e. data."name.with.dots") or between brackets
// (i.e. data["name.with.dots"]), because an unquoted dot is always treated as a key separator.
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
	keys := splitPath(path)
	currentKey := keys[0]
	switch value := extractValue(currentKey, object).(type) {
	case map[string]interface{}:
		if len(keys) == 1 {
			// If there are no more keys, it means we're at the end of the path
			// So we'll treat it as a string by re-marshaling it to JSON since it's a map.
			// Note that the output JSON will be minified.
			b, err := json.Marshal(value)
			return string(b), len(b), err
		}
		return walk(path[len(currentKey)+1:], value)
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

// splitPath splits the path into keys on every dot that is neither inside brackets nor inside quotes
func splitPath(path string) []string {
	var keys []string
	var quote byte
	startOfCurrentKey, bracketDepth := 0, 0
	for i := 0; i < len(path); i++ {
		switch {
		case quote != 0:
			if path[i] == quote {
				quote = 0
			}
		case (path[i] == '"' || path[i] == '\'') && (i == startOfCurrentKey || path[i-1] == '['):
			// Quotes are only meaningful at the start of a key (i.e. "m.server") or of a bracket (i.e. ["m.server"])
			quote = path[i]
		case path[i] == '[':
			bracketDepth++
		case path[i] == ']':
			bracketDepth--
		case path[i] == '.' && bracketDepth == 0:
			// If we encounter a dot, we've reached the end of a key unless we're inside a bracket or quotes
			keys = append(keys, path[startOfCurrentKey:i])
			startOfCurrentKey = i + 1
		}
	}
	return append(keys, path[startOfCurrentKey:])
}

// extractValue retrieves the value referenced by currentKey from value.
//
// currentKey is made of an optional key, which may be quoted (e.g. data, "m.server" or 'm.server'), followed by any
// number of brackets, each containing either an array index (e.g. [0]) or a quoted key (e.g. ["m.server"])
func extractValue(currentKey string, value interface{}) interface{} {
	key, brackets := currentKey, ""
	isQuoted := len(currentKey) > 0 && (currentKey[0] == '"' || currentKey[0] == '\'')
	if isQuoted {
		endOfQuote := strings.IndexByte(currentKey[1:], currentKey[0]) + 1
		if endOfQuote == 0 {
			return nil
		}
		key, brackets = currentKey[1:endOfQuote], currentKey[endOfQuote+1:]
	} else if startOfBracket := strings.IndexByte(currentKey, '['); startOfBracket != -1 {
		key, brackets = currentKey[:startOfBracket], currentKey[startOfBracket:]
	}
	if len(brackets) == 0 {
		return extractKey(key, value)
	}
	// If there's a key before the brackets (i.e. data[0]), we retrieve it before handling the brackets
	if isQuoted || len(key) > 0 {
		valueAsMap, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}
		value = valueAsMap[key]
	}
	for len(brackets) > 0 {
		if brackets[0] != '[' || len(brackets) < 2 {
			return nil
		}
		if brackets[1] == '"' || brackets[1] == '\'' {
			// The bracket contains a quoted key (i.e. ["m.server"])
			endOfQuote := strings.IndexByte(brackets[2:], brackets[1]) + 2
			if endOfQuote == 1 || len(brackets) <= endOfQuote+1 || brackets[endOfQuote+1] != ']' {
				return nil
			}
			valueAsMap, ok := value.(map[string]interface{})
			if !ok {
				return nil
			}
			value = valueAsMap[brackets[2:endOfQuote]]
			brackets = brackets[endOfQuote+2:]
			continue
		}
		// The bracket contains an array index (i.e. [0])
		endOfBracket := strings.IndexByte(brackets, ']')
		if endOfBracket == -1 {
			return nil
		}
		arrayIndex, err := strconv.Atoi(brackets[1:endOfBracket])
		if err != nil || arrayIndex < 0 {
			return nil
		}
		array, _ := value.([]interface{})
		if len(array) <= arrayIndex {
			return nil
		}
		value = array[arrayIndex]
		brackets = brackets[endOfBracket+1:]
	}
	return value
}

func extractKey(key string, value interface{}) interface{} {
	if valueAsSlice, ok := value.([]interface{}); ok {
		// If the type is a slice, return it
		// This happens when the body (value) is a JSON array
		return valueAsSlice
	}
	if valueAsMap, ok := value.(map[string]interface{}); ok {
		// If the value is a map, then we get the key from that map
		// This happens when the body (value) is a JSON object
		return valueAsMap[key]
	}
	// If the value is neither a map nor a slice, then we cannot retrieve the key from said value.
	// This usually happens when the body (value) is null.
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
