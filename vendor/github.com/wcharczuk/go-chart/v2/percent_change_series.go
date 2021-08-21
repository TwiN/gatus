package chart

// Interface Assertions.
var (
	_ Series                 = (*PercentChangeSeries)(nil)
	_ FirstValuesProvider    = (*PercentChangeSeries)(nil)
	_ LastValuesProvider     = (*PercentChangeSeries)(nil)
	_ ValueFormatterProvider = (*PercentChangeSeries)(nil)
)

// PercentChangeSeriesSource is a series that
// can be used with a PercentChangeSeries
type PercentChangeSeriesSource interface {
	Series
	FirstValuesProvider
	LastValuesProvider
	ValuesProvider
	ValueFormatterProvider
}

// PercentChangeSeries applies a
// percentage difference function to a given continuous series.
type PercentChangeSeries struct {
	Name        string
	Style       Style
	YAxis       YAxisType
	InnerSeries PercentChangeSeriesSource
}

// GetName returns the name of the time series.
func (pcs PercentChangeSeries) GetName() string {
	return pcs.Name
}

// GetStyle returns the line style.
func (pcs PercentChangeSeries) GetStyle() Style {
	return pcs.Style
}

// Len implements part of Series.
func (pcs PercentChangeSeries) Len() int {
	return pcs.InnerSeries.Len()
}

// GetFirstValues implements FirstValuesProvider.
func (pcs PercentChangeSeries) GetFirstValues() (x, y float64) {
	return pcs.InnerSeries.GetFirstValues()
}

// GetValues gets x, y values at a given index.
func (pcs PercentChangeSeries) GetValues(index int) (x, y float64) {
	_, fy := pcs.InnerSeries.GetFirstValues()
	x0, y0 := pcs.InnerSeries.GetValues(index)
	x = x0
	y = PercentDifference(fy, y0)
	return
}

// GetValueFormatters returns value formatter defaults for the series.
func (pcs PercentChangeSeries) GetValueFormatters() (x, y ValueFormatter) {
	x, _ = pcs.InnerSeries.GetValueFormatters()
	y = PercentValueFormatter
	return
}

// GetYAxis returns which YAxis the series draws on.
func (pcs PercentChangeSeries) GetYAxis() YAxisType {
	return pcs.YAxis
}

// GetLastValues gets the last values.
func (pcs PercentChangeSeries) GetLastValues() (x, y float64) {
	_, fy := pcs.InnerSeries.GetFirstValues()
	x0, y0 := pcs.InnerSeries.GetLastValues()
	x = x0
	y = PercentDifference(fy, y0)
	return
}

// Render renders the series.
func (pcs PercentChangeSeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	style := pcs.Style.InheritFrom(defaults)
	Draw.LineSeries(r, canvasBox, xrange, yrange, style, pcs)
}

// Validate validates the series.
func (pcs PercentChangeSeries) Validate() error {
	return pcs.InnerSeries.Validate()
}
