package jsonpath

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oliveagle/jsonpath"
)

// Eval just wraps oliveagle/jsonpath
func Eval(path string, b []byte) (string, int, error) {
	if len(path) == 0 && !(len(b) != 0 && b[0] == '[' && b[len(b)-1] == ']') {
		// if there's no path AND the value is not a JSON array, then there's nothing to walk
		s := string(b)
		return s, len(s), nil
	}

	if len(path) == 0 {
		path = "$"
	} else if !strings.HasPrefix(path, "$.") {
		path = "$." + path
	}

	var object interface{}
	if err := json.Unmarshal(b, &object); err != nil {
		return "", 0, err
	}

	obj, err := jsonpath.JsonPathLookup(object, path)
	if err != nil || obj == nil {
		return "", 0, err
	}

	switch value := obj.(type) {
	case map[string]interface{}:
		b, err := json.Marshal(value)
		s := string(b)
		return s, len(s), err
	case string:
		return value, len(value), nil
	case []interface{}:
		return fmt.Sprintf("%v", value), len(value), nil
	case interface{}:
		s := fmt.Sprintf("%v", value)
		return s, len(s), nil
	default:
		return "", 0, fmt.Errorf("jsonpath issue: unknown type returned %T", value)
	}
}
