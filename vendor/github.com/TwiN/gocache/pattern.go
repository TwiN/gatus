package gocache

import "path/filepath"

// MatchPattern checks whether a string matches a pattern
func MatchPattern(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	matched, _ := filepath.Match(pattern, s)
	return matched
}
