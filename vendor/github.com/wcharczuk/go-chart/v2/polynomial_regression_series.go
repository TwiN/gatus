package chart

import (
	"fmt"
	"math"

	"github.com/wcharczuk/go-chart/v2/matrix"
)

// Interface Assertions.
var (
	_ Series              = (*PolynomialRegressionSeries)(nil)
	_ FirstValuesProvider = (*PolynomialRegressionSeries)(nil)
	_ LastValuesProvider  = (*PolynomialRegressionSeries)(nil)
)

// PolynomialRegressionSeries implements a polynomial regression over a given
// inner series.
type PolynomialRegressionSeries struct {
	Name  string
	Style Style
	YAxis YAxisType

	Limit       int
	Offset      int
	Degree      int
	InnerSeries ValuesProvider

	coeffs []float64
}

// GetName returns the name of the time series.
func (prs PolynomialRegressionSeries) GetName() string {
	return prs.Name
}

// GetStyle returns the line style.
func (prs PolynomialRegressionSeries) GetStyle() Style {
	return prs.Style
}

// GetYAxis returns which YAxis the series draws on.
func (prs PolynomialRegressionSeries) GetYAxis() YAxisType {
	return prs.YAxis
}

// Len returns the number of elements in the series.
func (prs PolynomialRegressionSeries) Len() int {
	return MinInt(prs.GetLimit(), prs.InnerSeries.Len()-prs.GetOffset())
}

// GetLimit returns the window size.
func (prs PolynomialRegressionSeries) GetLimit() int {
	if prs.Limit == 0 {
		return prs.InnerSeries.Len()
	}
	return prs.Limit
}

// GetEndIndex returns the effective limit end.
func (prs PolynomialRegressionSeries) GetEndIndex() int {
	windowEnd := prs.GetOffset() + prs.GetLimit()
	innerSeriesLastIndex := prs.InnerSeries.Len() - 1
	return MinInt(windowEnd, innerSeriesLastIndex)
}

// GetOffset returns the data offset.
func (prs PolynomialRegressionSeries) GetOffset() int {
	if prs.Offset == 0 {
		return 0
	}
	return prs.Offset
}

// Validate validates the series.
func (prs *PolynomialRegressionSeries) Validate() error {
	if prs.InnerSeries == nil {
		return fmt.Errorf("linear regression series requires InnerSeries to be set")
	}

	endIndex := prs.GetEndIndex()
	if endIndex >= prs.InnerSeries.Len() {
		return fmt.Errorf("invalid window; inner series has length %d but end index is %d", prs.InnerSeries.Len(), endIndex)
	}

	return nil
}

// GetValues returns the series value for a given index.
func (prs *PolynomialRegressionSeries) GetValues(index int) (x, y float64) {
	if prs.InnerSeries == nil || prs.InnerSeries.Len() == 0 {
		return
	}

	if prs.coeffs == nil {
		coeffs, err := prs.computeCoefficients()
		if err != nil {
			panic(err)
		}
		prs.coeffs = coeffs
	}

	offset := prs.GetOffset()
	effectiveIndex := MinInt(index+offset, prs.InnerSeries.Len())
	x, y = prs.InnerSeries.GetValues(effectiveIndex)
	y = prs.apply(x)
	return
}

// GetFirstValues computes the first poly regression value.
func (prs *PolynomialRegressionSeries) GetFirstValues() (x, y float64) {
	if prs.InnerSeries == nil || prs.InnerSeries.Len() == 0 {
		return
	}
	if prs.coeffs == nil {
		coeffs, err := prs.computeCoefficients()
		if err != nil {
			panic(err)
		}
		prs.coeffs = coeffs
	}
	x, y = prs.InnerSeries.GetValues(0)
	y = prs.apply(x)
	return
}

// GetLastValues computes the last poly regression value.
func (prs *PolynomialRegressionSeries) GetLastValues() (x, y float64) {
	if prs.InnerSeries == nil || prs.InnerSeries.Len() == 0 {
		return
	}
	if prs.coeffs == nil {
		coeffs, err := prs.computeCoefficients()
		if err != nil {
			panic(err)
		}
		prs.coeffs = coeffs
	}
	endIndex := prs.GetEndIndex()
	x, y = prs.InnerSeries.GetValues(endIndex)
	y = prs.apply(x)
	return
}

func (prs *PolynomialRegressionSeries) apply(v float64) (out float64) {
	for index, coeff := range prs.coeffs {
		out = out + (coeff * math.Pow(v, float64(index)))
	}
	return
}

func (prs *PolynomialRegressionSeries) computeCoefficients() ([]float64, error) {
	xvalues, yvalues := prs.values()
	return matrix.Poly(xvalues, yvalues, prs.Degree)
}

func (prs *PolynomialRegressionSeries) values() (xvalues, yvalues []float64) {
	startIndex := prs.GetOffset()
	endIndex := prs.GetEndIndex()

	xvalues = make([]float64, endIndex-startIndex)
	yvalues = make([]float64, endIndex-startIndex)

	for index := startIndex; index < endIndex; index++ {
		x, y := prs.InnerSeries.GetValues(index)
		xvalues[index-startIndex] = x
		yvalues[index-startIndex] = y
	}

	return
}

// Render renders the series.
func (prs *PolynomialRegressionSeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	style := prs.Style.InheritFrom(defaults)
	Draw.LineSeries(r, canvasBox, xrange, yrange, style, prs)
}
