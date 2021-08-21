package chart

var (
	_ Sequence = (*Array)(nil)
)

// NewArray returns a new array from a given set of values.
// Array implements Sequence, which allows it to be used with the sequence helpers.
func NewArray(values ...float64) Array {
	return Array(values)
}

// Array is a wrapper for an array of floats that implements `ValuesProvider`.
type Array []float64

// Len returns the value provider length.
func (a Array) Len() int {
	return len(a)
}

// GetValue returns the value at a given index.
func (a Array) GetValue(index int) float64 {
	return a[index]
}
