package pattern

import "path/filepath"

// Match checks whether a string matches a pattern
func Match(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	matched, _ := filepath.Match(pattern, s)
	return matched
}
