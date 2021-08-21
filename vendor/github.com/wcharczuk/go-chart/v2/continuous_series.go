package chart

import "fmt"

// Interface Assertions.
var (
	_ Series              = (*ContinuousSeries)(nil)
	_ FirstValuesProvider = (*ContinuousSeries)(nil)
	_ LastValuesProvider  = (*ContinuousSeries)(nil)
)

// ContinuousSeries represents a line on a chart.
type ContinuousSeries struct {
	Name  string
	Style Style

	YAxis YAxisType

	XValueFormatter ValueFormatter
	YValueFormatter ValueFormatter

	XValues []float64
	YValues []float64
}

// GetName returns the name of the time series.
func (cs ContinuousSeries) GetName() string {
	return cs.Name
}

// GetStyle returns the line style.
func (cs ContinuousSeries) GetStyle() Style {
	return cs.Style
}

// Len returns the number of elements in the series.
func (cs ContinuousSeries) Len() int {
	return len(cs.XValues)
}

// GetValues gets the x,y values at a given index.
func (cs ContinuousSeries) GetValues(index int) (float64, float64) {
	return cs.XValues[index], cs.YValues[index]
}

// GetFirstValues gets the first x,y values.
func (cs ContinuousSeries) GetFirstValues() (float64, float64) {
	return cs.XValues[0], cs.YValues[0]
}

// GetLastValues gets the last x,y values.
func (cs ContinuousSeries) GetLastValues() (float64, float64) {
	return cs.XValues[len(cs.XValues)-1], cs.YValues[len(cs.YValues)-1]
}

// GetValueFormatters returns value formatter defaults for the series.
func (cs ContinuousSeries) GetValueFormatters() (x, y ValueFormatter) {
	if cs.XValueFormatter != nil {
		x = cs.XValueFormatter
	} else {
		x = FloatValueFormatter
	}
	if cs.YValueFormatter != nil {
		y = cs.YValueFormatter
	} else {
		y = FloatValueFormatter
	}
	return
}

// GetYAxis returns which YAxis the series draws on.
func (cs ContinuousSeries) GetYAxis() YAxisType {
	return cs.YAxis
}

// Render renders the series.
func (cs ContinuousSeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	style := cs.Style.InheritFrom(defaults)
	Draw.LineSeries(r, canvasBox, xrange, yrange, style, cs)
}

// Validate validates the series.
func (cs ContinuousSeries) Validate() error {
	if len(cs.XValues) == 0 {
		return fmt.Errorf("continuous series; must have xvalues set")
	}

	if len(cs.YValues) == 0 {
		return fmt.Errorf("continuous series; must have yvalues set")
	}

	if len(cs.XValues) != len(cs.YValues) {
		return fmt.Errorf("continuous series; must have same length xvalues as yvalues")
	}
	return nil
}
