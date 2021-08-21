package matrix

// Vector is just an array of values.
type Vector []float64

// DotProduct returns the dot product of two vectors.
func (v Vector) DotProduct(v2 Vector) (result float64, err error) {
	if len(v) != len(v2) {
		err = ErrDimensionMismatch
		return
	}

	for i := 0; i < len(v); i++ {
		result = result + (v[i] * v2[i])
	}
	return
}
