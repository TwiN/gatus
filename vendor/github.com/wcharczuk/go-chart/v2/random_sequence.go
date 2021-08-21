package chart

import (
	"math"
	"math/rand"
	"time"
)

var (
	_ Sequence = (*RandomSeq)(nil)
)

// RandomValues returns an array of random values.
func RandomValues(count int) []float64 {
	return Seq{NewRandomSequence().WithLen(count)}.Values()
}

// RandomValuesWithMax returns an array of random values with a given average.
func RandomValuesWithMax(count int, max float64) []float64 {
	return Seq{NewRandomSequence().WithMax(max).WithLen(count)}.Values()
}

// NewRandomSequence creates a new random seq.
func NewRandomSequence() *RandomSeq {
	return &RandomSeq{
		rnd: rand.New(rand.NewSource(time.Now().Unix())),
	}
}

// RandomSeq is a random number seq generator.
type RandomSeq struct {
	rnd *rand.Rand
	max *float64
	min *float64
	len *int
}

// Len returns the number of elements that will be generated.
func (r *RandomSeq) Len() int {
	if r.len != nil {
		return *r.len
	}
	return math.MaxInt32
}

// GetValue returns the value.
func (r *RandomSeq) GetValue(_ int) float64 {
	if r.min != nil && r.max != nil {
		var delta float64

		if *r.max > *r.min {
			delta = *r.max - *r.min
		} else {
			delta = *r.min - *r.max
		}

		return *r.min + (r.rnd.Float64() * delta)
	} else if r.max != nil {
		return r.rnd.Float64() * *r.max
	} else if r.min != nil {
		return *r.min + (r.rnd.Float64())
	}
	return r.rnd.Float64()
}

// WithLen sets a maximum len
func (r *RandomSeq) WithLen(length int) *RandomSeq {
	r.len = &length
	return r
}

// Min returns the minimum value.
func (r RandomSeq) Min() *float64 {
	return r.min
}

// WithMin sets the scale and returns the Random.
func (r *RandomSeq) WithMin(min float64) *RandomSeq {
	r.min = &min
	return r
}

// Max returns the maximum value.
func (r RandomSeq) Max() *float64 {
	return r.max
}

// WithMax sets the average and returns the Random.
func (r *RandomSeq) WithMax(max float64) *RandomSeq {
	r.max = &max
	return r
}
