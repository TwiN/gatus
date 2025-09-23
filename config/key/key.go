package key

import "strings"

// ConvertGroupAndNameToKey converts a group and a name to a key
func ConvertGroupAndNameToKey(groupNames []string, name string) string {
	return sanitize(strings.Join(groupNames, "-")) + "_" + sanitize(name)
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