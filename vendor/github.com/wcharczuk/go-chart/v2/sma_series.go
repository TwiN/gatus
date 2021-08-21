package chart

import (
	"fmt"
)

const (
	// DefaultSimpleMovingAveragePeriod is the default number of values to average.
	DefaultSimpleMovingAveragePeriod = 16
)

// Interface Assertions.
var (
	_ Series              = (*SMASeries)(nil)
	_ FirstValuesProvider = (*SMASeries)(nil)
	_ LastValuesProvider  = (*SMASeries)(nil)
)

// SMASeries is a computed series.
type SMASeries struct {
	Name  string
	Style Style
	YAxis YAxisType

	Period      int
	InnerSeries ValuesProvider
}

// GetName returns the name of the time series.
func (sma SMASeries) GetName() string {
	return sma.Name
}

// GetStyle returns the line style.
func (sma SMASeries) GetStyle() Style {
	return sma.Style
}

// GetYAxis returns which YAxis the series draws on.
func (sma SMASeries) GetYAxis() YAxisType {
	return sma.YAxis
}

// Len returns the number of elements in the series.
func (sma SMASeries) Len() int {
	return sma.InnerSeries.Len()
}

// GetPeriod returns the window size.
func (sma SMASeries) GetPeriod(defaults ...int) int {
	if sma.Period == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultSimpleMovingAveragePeriod
	}
	return sma.Period
}

// GetValues gets a value at a given index.
func (sma SMASeries) GetValues(index int) (x, y float64) {
	if sma.InnerSeries == nil || sma.InnerSeries.Len() == 0 {
		return
	}
	px, _ := sma.InnerSeries.GetValues(index)
	x = px
	y = sma.getAverage(index)
	return
}

// GetFirstValues computes the first moving average value.
func (sma SMASeries) GetFirstValues() (x, y float64) {
	if sma.InnerSeries == nil || sma.InnerSeries.Len() == 0 {
		return
	}
	px, _ := sma.InnerSeries.GetValues(0)
	x = px
	y = sma.getAverage(0)
	return
}

// GetLastValues computes the last moving average value but walking back window size samples,
// and recomputing the last moving average chunk.
func (sma SMASeries) GetLastValues() (x, y float64) {
	if sma.InnerSeries == nil || sma.InnerSeries.Len() == 0 {
		return
	}
	seriesLen := sma.InnerSeries.Len()
	px, _ := sma.InnerSeries.GetValues(seriesLen - 1)
	x = px
	y = sma.getAverage(seriesLen - 1)
	return
}

func (sma SMASeries) getAverage(index int) float64 {
	period := sma.GetPeriod()
	floor := MaxInt(0, index-period)
	var accum float64
	var count float64
	for x := index; x >= floor; x-- {
		_, vy := sma.InnerSeries.GetValues(x)
		accum += vy
		count += 1.0
	}
	return accum / count
}

// Render renders the series.
func (sma SMASeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	style := sma.Style.InheritFrom(defaults)
	Draw.LineSeries(r, canvasBox, xrange, yrange, style, sma)
}

// Validate validates the series.
func (sma SMASeries) Validate() error {
	if sma.InnerSeries == nil {
		return fmt.Errorf("sma series requires InnerSeries to be set")
	}
	return nil
}
