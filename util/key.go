package util

import "strings"

// ConvertGroupAndServiceToKey converts a group and a service to a key
func ConvertGroupAndServiceToKey(group, service string) string {
	return sanitize(group) + "_" + sanitize(service)
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
