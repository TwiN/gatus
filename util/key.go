package util

import "strings"

// ConvertGroupAndEndpointNameToKey converts a group and an endpoint to a key
func ConvertGroupAndEndpointNameToKey(group, endpoint string) string {
	return sanitize(group) + "_" + sanitize(endpoint)
}

func sanitize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ",", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
