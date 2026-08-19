package jsonpath

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Eval is a half-baked json path implementation that needs some love
//
// Beyond plain dotted keys and [N] array indices, it supports a single
// gjson-style query segment for selecting one element out of an array of
// objects by a field's value: arrayField.#(key=="value").remainingPath.
// See queryArray for the exact syntax and its limitations.
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
	startOfCurrentKey, depth := 0, 0
	for i := range path {
		// [ and ( both open a nested segment (an array index or a #(...) query);
		// a dot inside either must not be treated as a key separator.
		switch path[i] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		}
		// If we encounter a dot, we've reached the end of a key unless we're inside a bracket
		if path[i] == '.' && depth == 0 {
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
		// If the next segment is a #(key=="value") query, use it to select a
		// single element out of the array and keep walking from there.
		// Otherwise, preserve the existing behavior of returning the whole
		// array formatted as a string.
		if len(keys) > 1 && isQuerySegment(keys[1]) {
			matched, err := queryArray(keys[1], value)
			if err != nil {
				return "", 0, err
			}
			newPath := strings.Replace(path, fmt.Sprintf("%s.%s.", currentKey, keys[1]), "", 1)
			if newPath == path {
				// The query was the last segment of the path, so treat the
				// matched element like the end-of-path map case above.
				b, err := json.Marshal(matched)
				return string(b), len(b), err
			}
			return walk(newPath, matched)
		}
		return fmt.Sprintf("%v", value), len(value), nil
	case interface{}:
		newValue := fmt.Sprintf("%v", value)
		return newValue, len(newValue), nil
	default:
		return "", 0, fmt.Errorf("couldn't walk through '%s' because type was '%T', but expected 'map[string]interface{}'", currentKey, value)
	}
}

// isQuerySegment returns whether a path segment is a #(key=="value") query,
// as opposed to a plain key.
func isQuerySegment(segment string) bool {
	return strings.HasPrefix(segment, "#(") && strings.HasSuffix(segment, ")")
}

// queryArray evaluates a #(key=="value") segment against an array of objects
// and returns the first element for which key equals value.
//
// This intentionally supports only exact-match string equality against a
// double-quoted literal (no !=, no wildcards, no unquoted/numeric values, no
// selecting all matches) -- just enough to pick a single named element (e.g.
// a component) out of the kind of array returned by any Atlassian-
// Statuspage-compatible components.json endpoint, which is the motivating
// use case. Root-level array queries (a query as the very first path
// segment, with no preceding field name) aren't supported either -- in
// practice a query always follows the array field it selects from.
func queryArray(segment string, array []interface{}) (map[string]interface{}, error) {
	inner := segment[2 : len(segment)-1] // strip the leading "#(" and trailing ")"
	key, value, found := strings.Cut(inner, "==")
	if !found {
		return nil, fmt.Errorf("invalid query '%s': expected the form #(key==\"value\")", segment)
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return nil, fmt.Errorf("invalid query '%s': value must be a quoted string, e.g. #(%s==\"value\")", segment, key)
	}
	value = value[1 : len(value)-1]
	for _, element := range array {
		if object, ok := element.(map[string]interface{}); ok {
			if fmt.Sprintf("%v", object[key]) == value {
				return object, nil
			}
		}
	}
	return nil, fmt.Errorf("no element found matching query '%s'", segment)
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
		if err != nil || arrayIndex < 0 {
			return nil
		}
		currentKeyWithoutIndex := currentKey[:startOfBracket]
		// if currentKeyWithoutIndex contains only an index (i.e. [0] or 0)
		if len(currentKeyWithoutIndex) == 0 {
			array, _ := value.([]interface{})
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
		array, _ := value.(map[string]interface{})[currentKeyWithoutIndex].([]interface{})
		if len(array) > arrayIndex {
			if isNestedArray {
				return extractValue(currentKey[endOfBracket+1:], array[arrayIndex])
			}
			return array[arrayIndex]
		}
		return nil
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
