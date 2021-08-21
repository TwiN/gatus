package chart

import (
	"fmt"
)

// Interface Assertions.
var (
	_ Series = (*BollingerBandsSeries)(nil)
)

// BollingerBandsSeries draws bollinger bands for an inner series.
// Bollinger bands are defined by two lines, one at SMA+k*stddev, one at SMA-k*stdev.
type BollingerBandsSeries struct {
	Name  string
	Style Style
	YAxis YAxisType

	Period      int
	K           float64
	InnerSeries ValuesProvider

	valueBuffer *ValueBuffer
}

// GetName returns the name of the time series.
func (bbs BollingerBandsSeries) GetName() string {
	return bbs.Name
}

// GetStyle returns the line style.
func (bbs BollingerBandsSeries) GetStyle() Style {
	return bbs.Style
}

// GetYAxis returns which YAxis the series draws on.
func (bbs BollingerBandsSeries) GetYAxis() YAxisType {
	return bbs.YAxis
}

// GetPeriod returns the window size.
func (bbs BollingerBandsSeries) GetPeriod() int {
	if bbs.Period == 0 {
		return DefaultSimpleMovingAveragePeriod
	}
	return bbs.Period
}

// GetK returns the K value, or the number of standard deviations above and below
// to band the simple moving average with.
// Typical K value is 2.0.
func (bbs BollingerBandsSeries) GetK(defaults ...float64) float64 {
	if bbs.K == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return 2.0
	}
	return bbs.K
}

// Len returns the number of elements in the series.
func (bbs BollingerBandsSeries) Len() int {
	return bbs.InnerSeries.Len()
}

// GetBoundedValues gets the bounded value for the series.
func (bbs *BollingerBandsSeries) GetBoundedValues(index int) (x, y1, y2 float64) {
	if bbs.InnerSeries == nil {
		return
	}
	if bbs.valueBuffer == nil || index == 0 {
		bbs.valueBuffer = NewValueBufferWithCapacity(bbs.GetPeriod())
	}
	if bbs.valueBuffer.Len() >= bbs.GetPeriod() {
		bbs.valueBuffer.Dequeue()
	}
	px, py := bbs.InnerSeries.GetValues(index)
	bbs.valueBuffer.Enqueue(py)
	x = px

	ay := Seq{bbs.valueBuffer}.Average()
	std := Seq{bbs.valueBuffer}.StdDev()

	y1 = ay + (bbs.GetK() * std)
	y2 = ay - (bbs.GetK() * std)
	return
}

// GetBoundedLastValues returns the last bounded value for the series.
func (bbs *BollingerBandsSeries) GetBoundedLastValues() (x, y1, y2 float64) {
	if bbs.InnerSeries == nil {
		return
	}
	period := bbs.GetPeriod()
	seriesLength := bbs.InnerSeries.Len()
	startAt := seriesLength - period
	if startAt < 0 {
		startAt = 0
	}

	vb := NewValueBufferWithCapacity(period)
	for index := startAt; index < seriesLength; index++ {
		xn, yn := bbs.InnerSeries.GetValues(index)
		vb.Enqueue(yn)
		x = xn
	}

	ay := Seq{vb}.Average()
	std := Seq{vb}.StdDev()

	y1 = ay + (bbs.GetK() * std)
	y2 = ay - (bbs.GetK() * std)

	return
}

// Render renders the series.
func (bbs *BollingerBandsSeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	s := bbs.Style.InheritFrom(defaults.InheritFrom(Style{
		StrokeWidth: 1.0,
		StrokeColor: DefaultAxisColor.WithAlpha(64),
		FillColor:   DefaultAxisColor.WithAlpha(32),
	}))

	Draw.BoundedSeries(r, canvasBox, xrange, yrange, s, bbs, bbs.GetPeriod())
}

// Validate validates the series.
func (bbs BollingerBandsSeries) Validate() error {
	if bbs.InnerSeries == nil {
		return fmt.Errorf("bollinger bands series requires InnerSeries to be set")
	}
	return nil
}
