package jsonpath

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func Eval(path string, b []byte) (string, error) {
	var object interface{}
	err := json.Unmarshal(b, &object)
	if err != nil {
		// Try to unmarshall it into an array instead
		return "", err
	}
	return walk(path, object)
}

func walk(path string, object interface{}) (string, error) {
	keys := strings.Split(path, ".")
	currentKey := keys[0]
	// if there's more than one key, then walk deeper
	if len(keys) > 1 {
		switch value := extractValue(currentKey, object).(type) {
		case map[string]interface{}:
			return walk(strings.Replace(path, fmt.Sprintf("%s.", currentKey), "", 1), value)
		case interface{}:
			return fmt.Sprintf("%v", value), nil
		default:
			return "", fmt.Errorf("couldn't walk through '%s' because type was '%T', but expected 'map[string]interface{}'", currentKey, value)
		}
	}
	// if there's only one key and the target key is that key, then return its value
	return fmt.Sprintf("%v", extractValue(currentKey, object)), nil
}

func extractValue(currentKey string, value interface{}) interface{} {
	// Check if the current key ends with [#]
	if strings.HasSuffix(currentKey, "]") && strings.Contains(currentKey, "[") {
		tmp := strings.SplitN(currentKey, "[", 3)
		arrayIndex, err := strconv.Atoi(strings.Replace(tmp[1], "]", "", 1))
		if err != nil {
			return value
		}
		currentKey := tmp[0]
		// if currentKey contains only an index (i.e. [0] or 0)
		if len(currentKey) == 0 {
			array := value.([]interface{})
			if len(array) > arrayIndex {
				if len(tmp) > 2 {
					// Nested array? Go deeper.
					return extractValue(fmt.Sprintf("%s[%s", currentKey, tmp[2]), array[arrayIndex])
				}
				return array[arrayIndex]
			}
			return nil
		}
		// if currentKey contains both a key and an index (i.e. data[0])
		array := value.(map[string]interface{})[currentKey].([]interface{})
		if len(array) > arrayIndex {
			if len(tmp) > 2 {
				// Nested array? Go deeper.
				return extractValue(fmt.Sprintf("[%s", tmp[2]), array[arrayIndex])
			}
			return array[arrayIndex]
		}
		return nil
	}
	return value.(map[string]interface{})[currentKey]
}
