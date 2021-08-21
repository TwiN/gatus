package chart

import (
	"fmt"
	"math"
)

// ContinuousRange represents a boundary for a set of numbers.
type ContinuousRange struct {
	Min        float64
	Max        float64
	Domain     int
	Descending bool
}

// IsDescending returns if the range is descending.
func (r ContinuousRange) IsDescending() bool {
	return r.Descending
}

// IsZero returns if the ContinuousRange has been set or not.
func (r ContinuousRange) IsZero() bool {
	return (r.Min == 0 || math.IsNaN(r.Min)) &&
		(r.Max == 0 || math.IsNaN(r.Max)) &&
		r.Domain == 0
}

// GetMin gets the min value for the continuous range.
func (r ContinuousRange) GetMin() float64 {
	return r.Min
}

// SetMin sets the min value for the continuous range.
func (r *ContinuousRange) SetMin(min float64) {
	r.Min = min
}

// GetMax returns the max value for the continuous range.
func (r ContinuousRange) GetMax() float64 {
	return r.Max
}

// SetMax sets the max value for the continuous range.
func (r *ContinuousRange) SetMax(max float64) {
	r.Max = max
}

// GetDelta returns the difference between the min and max value.
func (r ContinuousRange) GetDelta() float64 {
	return r.Max - r.Min
}

// GetDomain returns the range domain.
func (r ContinuousRange) GetDomain() int {
	return r.Domain
}

// SetDomain sets the range domain.
func (r *ContinuousRange) SetDomain(domain int) {
	r.Domain = domain
}

// String returns a simple string for the ContinuousRange.
func (r ContinuousRange) String() string {
	if r.GetDelta() == 0 {
		return "ContinuousRange [empty]"
	}
	return fmt.Sprintf("ContinuousRange [%.2f,%.2f] => %d", r.Min, r.Max, r.Domain)
}

// Translate maps a given value into the ContinuousRange space.
func (r ContinuousRange) Translate(value float64) int {
	normalized := value - r.Min
	ratio := normalized / r.GetDelta()

	if r.IsDescending() {
		return r.Domain - int(math.Ceil(ratio*float64(r.Domain)))
	}

	return int(math.Ceil(ratio * float64(r.Domain)))
}
