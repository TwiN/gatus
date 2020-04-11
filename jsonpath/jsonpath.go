package jsonpath

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Eval(path string, b []byte) string {
	var object map[string]interface{}
	err := json.Unmarshal(b, &object)
	if err != nil {
		return ""
	}
	return walk(path, object)
}

func walk(path string, object map[string]interface{}) string {
	keys := strings.Split(path, ".")
	targetKey := keys[0]
	// if there's only one key and the target key is that key, then return its value
	if len(keys) == 1 {
		return fmt.Sprintf("%v", object[targetKey])
	}
	// if there's more than one key, then walk deeper
	if len(keys) > 0 {
		return walk(strings.Replace(path, fmt.Sprintf("%s.", targetKey), "", 1), object[targetKey].(map[string]interface{}))
	}
	return ""
}
