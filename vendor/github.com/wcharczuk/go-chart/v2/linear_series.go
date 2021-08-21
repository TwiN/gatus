package chart

import (
	"fmt"
)

// Interface Assertions.
var (
	_ Series              = (*LinearSeries)(nil)
	_ FirstValuesProvider = (*LinearSeries)(nil)
	_ LastValuesProvider  = (*LinearSeries)(nil)
)

// LinearSeries is a series that plots a line in a given domain.
type LinearSeries struct {
	Name  string
	Style Style
	YAxis YAxisType

	XValues     []float64
	InnerSeries LinearCoefficientProvider

	m     float64
	b     float64
	stdev float64
	avg   float64
}

// GetName returns the name of the time series.
func (ls LinearSeries) GetName() string {
	return ls.Name
}

// GetStyle returns the line style.
func (ls LinearSeries) GetStyle() Style {
	return ls.Style
}

// GetYAxis returns which YAxis the series draws on.
func (ls LinearSeries) GetYAxis() YAxisType {
	return ls.YAxis
}

// Len returns the number of elements in the series.
func (ls LinearSeries) Len() int {
	return len(ls.XValues)
}

// GetEndIndex returns the effective limit end.
func (ls LinearSeries) GetEndIndex() int {
	return len(ls.XValues) - 1
}

// GetValues gets a value at a given index.
func (ls *LinearSeries) GetValues(index int) (x, y float64) {
	if ls.InnerSeries == nil || len(ls.XValues) == 0 {
		return
	}
	if ls.IsZero() {
		ls.computeCoefficients()
	}
	x = ls.XValues[index]
	y = (ls.m * ls.normalize(x)) + ls.b
	return
}

// GetFirstValues computes the first linear regression value.
func (ls *LinearSeries) GetFirstValues() (x, y float64) {
	if ls.InnerSeries == nil || len(ls.XValues) == 0 {
		return
	}
	if ls.IsZero() {
		ls.computeCoefficients()
	}
	x, y = ls.GetValues(0)
	return
}

// GetLastValues computes the last linear regression value.
func (ls *LinearSeries) GetLastValues() (x, y float64) {
	if ls.InnerSeries == nil || len(ls.XValues) == 0 {
		return
	}
	if ls.IsZero() {
		ls.computeCoefficients()
	}
	x, y = ls.GetValues(ls.GetEndIndex())
	return
}

// Render renders the series.
func (ls *LinearSeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	Draw.LineSeries(r, canvasBox, xrange, yrange, ls.Style.InheritFrom(defaults), ls)
}

// Validate validates the series.
func (ls LinearSeries) Validate() error {
	if ls.InnerSeries == nil {
		return fmt.Errorf("linear regression series requires InnerSeries to be set")
	}
	return nil
}

// IsZero returns if the linear series has computed coefficients or not.
func (ls LinearSeries) IsZero() bool {
	return ls.m == 0 && ls.b == 0
}

// computeCoefficients computes the `m` and `b` terms in the linear formula given by `y = mx+b`.
func (ls *LinearSeries) computeCoefficients() {
	ls.m, ls.b, ls.stdev, ls.avg = ls.InnerSeries.Coefficients()
}

func (ls *LinearSeries) normalize(xvalue float64) float64 {
	if ls.avg > 0 && ls.stdev > 0 {
		return (xvalue - ls.avg) / ls.stdev
	}
	return xvalue
}
