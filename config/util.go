package config

// toPtr returns a pointer to the given value
func toPtr[T any](value T) *T {
	return &value
}

// contains checks if a key exists in the slice
func contains[T comparable](slice []T, key T) bool {
	for _, item := range slice {
		if item == key {
			return true
		}
	}
	return false
}
