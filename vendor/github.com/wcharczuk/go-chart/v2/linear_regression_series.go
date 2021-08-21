package chart

import (
	"fmt"
)

// Interface Assertions.
var (
	_ Series                    = (*LinearRegressionSeries)(nil)
	_ FirstValuesProvider       = (*LinearRegressionSeries)(nil)
	_ LastValuesProvider        = (*LinearRegressionSeries)(nil)
	_ LinearCoefficientProvider = (*LinearRegressionSeries)(nil)
)

// LinearRegressionSeries is a series that plots the n-nearest neighbors
// linear regression for the values.
type LinearRegressionSeries struct {
	Name  string
	Style Style
	YAxis YAxisType

	Limit       int
	Offset      int
	InnerSeries ValuesProvider

	m       float64
	b       float64
	avgx    float64
	stddevx float64
}

// Coefficients returns the linear coefficients for the series.
func (lrs LinearRegressionSeries) Coefficients() (m, b, stdev, avg float64) {
	if lrs.IsZero() {
		lrs.computeCoefficients()
	}

	m = lrs.m
	b = lrs.b
	stdev = lrs.stddevx
	avg = lrs.avgx
	return
}

// GetName returns the name of the time series.
func (lrs LinearRegressionSeries) GetName() string {
	return lrs.Name
}

// GetStyle returns the line style.
func (lrs LinearRegressionSeries) GetStyle() Style {
	return lrs.Style
}

// GetYAxis returns which YAxis the series draws on.
func (lrs LinearRegressionSeries) GetYAxis() YAxisType {
	return lrs.YAxis
}

// Len returns the number of elements in the series.
func (lrs LinearRegressionSeries) Len() int {
	return MinInt(lrs.GetLimit(), lrs.InnerSeries.Len()-lrs.GetOffset())
}

// GetLimit returns the window size.
func (lrs LinearRegressionSeries) GetLimit() int {
	if lrs.Limit == 0 {
		return lrs.InnerSeries.Len()
	}
	return lrs.Limit
}

// GetEndIndex returns the effective limit end.
func (lrs LinearRegressionSeries) GetEndIndex() int {
	windowEnd := lrs.GetOffset() + lrs.GetLimit()
	innerSeriesLastIndex := lrs.InnerSeries.Len() - 1
	return MinInt(windowEnd, innerSeriesLastIndex)
}

// GetOffset returns the data offset.
func (lrs LinearRegressionSeries) GetOffset() int {
	if lrs.Offset == 0 {
		return 0
	}
	return lrs.Offset
}

// GetValues gets a value at a given index.
func (lrs *LinearRegressionSeries) GetValues(index int) (x, y float64) {
	if lrs.InnerSeries == nil || lrs.InnerSeries.Len() == 0 {
		return
	}
	if lrs.IsZero() {
		lrs.computeCoefficients()
	}
	offset := lrs.GetOffset()
	effectiveIndex := MinInt(index+offset, lrs.InnerSeries.Len())
	x, y = lrs.InnerSeries.GetValues(effectiveIndex)
	y = (lrs.m * lrs.normalize(x)) + lrs.b
	return
}

// GetFirstValues computes the first linear regression value.
func (lrs *LinearRegressionSeries) GetFirstValues() (x, y float64) {
	if lrs.InnerSeries == nil || lrs.InnerSeries.Len() == 0 {
		return
	}
	if lrs.IsZero() {
		lrs.computeCoefficients()
	}
	x, y = lrs.InnerSeries.GetValues(0)
	y = (lrs.m * lrs.normalize(x)) + lrs.b
	return
}

// GetLastValues computes the last linear regression value.
func (lrs *LinearRegressionSeries) GetLastValues() (x, y float64) {
	if lrs.InnerSeries == nil || lrs.InnerSeries.Len() == 0 {
		return
	}
	if lrs.IsZero() {
		lrs.computeCoefficients()
	}
	endIndex := lrs.GetEndIndex()
	x, y = lrs.InnerSeries.GetValues(endIndex)
	y = (lrs.m * lrs.normalize(x)) + lrs.b
	return
}

// Render renders the series.
func (lrs *LinearRegressionSeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	style := lrs.Style.InheritFrom(defaults)
	Draw.LineSeries(r, canvasBox, xrange, yrange, style, lrs)
}

// Validate validates the series.
func (lrs *LinearRegressionSeries) Validate() error {
	if lrs.InnerSeries == nil {
		return fmt.Errorf("linear regression series requires InnerSeries to be set")
	}
	return nil
}

// IsZero returns if we've computed the coefficients or not.
func (lrs *LinearRegressionSeries) IsZero() bool {
	return lrs.m == 0 && lrs.b == 0
}

//
// internal helpers
//

func (lrs *LinearRegressionSeries) normalize(xvalue float64) float64 {
	return (xvalue - lrs.avgx) / lrs.stddevx
}

// computeCoefficients computes the `m` and `b` terms in the linear formula given by `y = mx+b`.
func (lrs *LinearRegressionSeries) computeCoefficients() {
	startIndex := lrs.GetOffset()
	endIndex := lrs.GetEndIndex()

	p := float64(endIndex - startIndex)

	xvalues := NewValueBufferWithCapacity(lrs.Len())
	for index := startIndex; index < endIndex; index++ {
		x, _ := lrs.InnerSeries.GetValues(index)
		xvalues.Enqueue(x)
	}

	lrs.avgx = Seq{xvalues}.Average()
	lrs.stddevx = Seq{xvalues}.StdDev()

	var sumx, sumy, sumxx, sumxy float64
	for index := startIndex; index < endIndex; index++ {
		x, y := lrs.InnerSeries.GetValues(index)

		x = lrs.normalize(x)

		sumx += x
		sumy += y
		sumxx += x * x
		sumxy += x * y
	}

	lrs.m = (p*sumxy - sumx*sumy) / (p*sumxx - sumx*sumx)
	lrs.b = (sumy / p) - (lrs.m * sumx / p)
}
