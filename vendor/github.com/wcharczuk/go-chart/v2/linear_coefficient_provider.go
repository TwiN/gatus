package chart

// LinearCoefficientProvider is a type that returns linear cofficients.
type LinearCoefficientProvider interface {
	Coefficients() (m, b, stdev, avg float64)
}

// LinearCoefficients returns a fixed linear coefficient pair.
func LinearCoefficients(m, b float64) LinearCoefficientSet {
	return LinearCoefficientSet{
		M: m,
		B: b,
	}
}

// NormalizedLinearCoefficients returns a fixed linear coefficient pair.
func NormalizedLinearCoefficients(m, b, stdev, avg float64) LinearCoefficientSet {
	return LinearCoefficientSet{
		M:      m,
		B:      b,
		StdDev: stdev,
		Avg:    avg,
	}
}

// LinearCoefficientSet is the m and b values for the linear equation in the form:
// y = (m*x) + b
type LinearCoefficientSet struct {
	M      float64
	B      float64
	StdDev float64
	Avg    float64
}

// Coefficients returns the coefficients.
func (lcs LinearCoefficientSet) Coefficients() (m, b, stdev, avg float64) {
	m = lcs.M
	b = lcs.B
	stdev = lcs.StdDev
	avg = lcs.Avg
	return
}
