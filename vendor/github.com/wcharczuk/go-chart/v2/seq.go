package chart

import (
	"math"
	"sort"
)

// ValueSequence returns a sequence for a given values set.
func ValueSequence(values ...float64) Seq {
	return Seq{NewArray(values...)}
}

// Sequence is a provider for values for a seq.
type Sequence interface {
	Len() int
	GetValue(int) float64
}

// Seq is a utility wrapper for seq providers.
type Seq struct {
	Sequence
}

// Values enumerates the seq into a slice.
func (s Seq) Values() (output []float64) {
	if s.Len() == 0 {
		return
	}

	output = make([]float64, s.Len())
	for i := 0; i < s.Len(); i++ {
		output[i] = s.GetValue(i)
	}
	return
}

// Each applies the `mapfn` to all values in the value provider.
func (s Seq) Each(mapfn func(int, float64)) {
	for i := 0; i < s.Len(); i++ {
		mapfn(i, s.GetValue(i))
	}
}

// Map applies the `mapfn` to all values in the value provider,
// returning a new seq.
func (s Seq) Map(mapfn func(i int, v float64) float64) Seq {
	output := make([]float64, s.Len())
	for i := 0; i < s.Len(); i++ {
		mapfn(i, s.GetValue(i))
	}
	return Seq{Array(output)}
}

// FoldLeft collapses a seq from left to right.
func (s Seq) FoldLeft(mapfn func(i int, v0, v float64) float64) (v0 float64) {
	if s.Len() == 0 {
		return 0
	}

	if s.Len() == 1 {
		return s.GetValue(0)
	}

	v0 = s.GetValue(0)
	for i := 1; i < s.Len(); i++ {
		v0 = mapfn(i, v0, s.GetValue(i))
	}
	return
}

// FoldRight collapses a seq from right to left.
func (s Seq) FoldRight(mapfn func(i int, v0, v float64) float64) (v0 float64) {
	if s.Len() == 0 {
		return 0
	}

	if s.Len() == 1 {
		return s.GetValue(0)
	}

	v0 = s.GetValue(s.Len() - 1)
	for i := s.Len() - 2; i >= 0; i-- {
		v0 = mapfn(i, v0, s.GetValue(i))
	}
	return
}

// Min returns the minimum value in the seq.
func (s Seq) Min() float64 {
	if s.Len() == 0 {
		return 0
	}
	min := s.GetValue(0)
	var value float64
	for i := 1; i < s.Len(); i++ {
		value = s.GetValue(i)
		if value < min {
			min = value
		}
	}
	return min
}

// Max returns the maximum value in the seq.
func (s Seq) Max() float64 {
	if s.Len() == 0 {
		return 0
	}
	max := s.GetValue(0)
	var value float64
	for i := 1; i < s.Len(); i++ {
		value = s.GetValue(i)
		if value > max {
			max = value
		}
	}
	return max
}

// MinMax returns the minimum and the maximum in one pass.
func (s Seq) MinMax() (min, max float64) {
	if s.Len() == 0 {
		return
	}
	min = s.GetValue(0)
	max = min
	var value float64
	for i := 1; i < s.Len(); i++ {
		value = s.GetValue(i)
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	return
}

// Sort returns the seq sorted in ascending order.
// This fully enumerates the seq.
func (s Seq) Sort() Seq {
	if s.Len() == 0 {
		return s
	}
	values := s.Values()
	sort.Float64s(values)
	return Seq{Array(values)}
}

// Reverse reverses the sequence
func (s Seq) Reverse() Seq {
	if s.Len() == 0 {
		return s
	}

	values := s.Values()
	valuesLen := len(values)
	valuesLen1 := len(values) - 1
	valuesLen2 := valuesLen >> 1
	var i, j float64
	for index := 0; index < valuesLen2; index++ {
		i = values[index]
		j = values[valuesLen1-index]
		values[index] = j
		values[valuesLen1-index] = i
	}

	return Seq{Array(values)}
}

// Median returns the median or middle value in the sorted seq.
func (s Seq) Median() (median float64) {
	l := s.Len()
	if l == 0 {
		return
	}

	sorted := s.Sort()
	if l%2 == 0 {
		v0 := sorted.GetValue(l/2 - 1)
		v1 := sorted.GetValue(l/2 + 1)
		median = (v0 + v1) / 2
	} else {
		median = float64(sorted.GetValue(l << 1))
	}

	return
}

// Sum adds all the elements of a series together.
func (s Seq) Sum() (accum float64) {
	if s.Len() == 0 {
		return 0
	}

	for i := 0; i < s.Len(); i++ {
		accum += s.GetValue(i)
	}
	return
}

// Average returns the float average of the values in the buffer.
func (s Seq) Average() float64 {
	if s.Len() == 0 {
		return 0
	}

	return s.Sum() / float64(s.Len())
}

// Variance computes the variance of the buffer.
func (s Seq) Variance() float64 {
	if s.Len() == 0 {
		return 0
	}

	m := s.Average()
	var variance, v float64
	for i := 0; i < s.Len(); i++ {
		v = s.GetValue(i)
		variance += (v - m) * (v - m)
	}

	return variance / float64(s.Len())
}

// StdDev returns the standard deviation.
func (s Seq) StdDev() float64 {
	if s.Len() == 0 {
		return 0
	}

	return math.Pow(s.Variance(), 0.5)
}

//Percentile finds the relative standing in a slice of floats.
// `percent` should be given on the interval [0,1.0).
func (s Seq) Percentile(percent float64) (percentile float64) {
	l := s.Len()
	if l == 0 {
		return 0
	}

	if percent < 0 || percent > 1.0 {
		panic("percent out of range [0.0, 1.0)")
	}

	sorted := s.Sort()
	index := percent * float64(l)
	if index == float64(int64(index)) {
		i := f64i(index)
		ci := sorted.GetValue(i - 1)
		c := sorted.GetValue(i)
		percentile = (ci + c) / 2.0
	} else {
		i := f64i(index)
		percentile = sorted.GetValue(i)
	}

	return percentile
}

// Normalize maps every value to the interval [0, 1.0].
func (s Seq) Normalize() Seq {
	min, max := s.MinMax()

	delta := max - min
	output := make([]float64, s.Len())
	for i := 0; i < s.Len(); i++ {
		output[i] = (s.GetValue(i) - min) / delta
	}

	return Seq{Array(output)}
}
