package endpoint

import "strings"

// ConvertGroupAndEndpointNameToKey converts a group and an endpoint to a key
func ConvertGroupAndEndpointNameToKey(groupName, endpointName string) string {
	return sanitize(groupName) + "_" + sanitize(endpointName)
}

func sanitize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ",", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "#", "-")
	return s
}
