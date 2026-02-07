package config

// toPtr returns a pointer to the given value
func toPtr[T any](value T) *T {
	return &value
}
