package chart

import "fmt"

const (
	// DefaultEMAPeriod is the default EMA period used in the sigma calculation.
	DefaultEMAPeriod = 12
)

// Interface Assertions.
var (
	_ Series              = (*EMASeries)(nil)
	_ FirstValuesProvider = (*EMASeries)(nil)
	_ LastValuesProvider  = (*EMASeries)(nil)
)

// EMASeries is a computed series.
type EMASeries struct {
	Name  string
	Style Style
	YAxis YAxisType

	Period      int
	InnerSeries ValuesProvider

	cache []float64
}

// GetName returns the name of the time series.
func (ema EMASeries) GetName() string {
	return ema.Name
}

// GetStyle returns the line style.
func (ema EMASeries) GetStyle() Style {
	return ema.Style
}

// GetYAxis returns which YAxis the series draws on.
func (ema EMASeries) GetYAxis() YAxisType {
	return ema.YAxis
}

// GetPeriod returns the window size.
func (ema EMASeries) GetPeriod() int {
	if ema.Period == 0 {
		return DefaultEMAPeriod
	}
	return ema.Period
}

// Len returns the number of elements in the series.
func (ema EMASeries) Len() int {
	return ema.InnerSeries.Len()
}

// GetSigma returns the smoothing factor for the serise.
func (ema EMASeries) GetSigma() float64 {
	return 2.0 / (float64(ema.GetPeriod()) + 1)
}

// GetValues gets a value at a given index.
func (ema *EMASeries) GetValues(index int) (x, y float64) {
	if ema.InnerSeries == nil {
		return
	}
	if len(ema.cache) == 0 {
		ema.ensureCachedValues()
	}
	vx, _ := ema.InnerSeries.GetValues(index)
	x = vx
	y = ema.cache[index]
	return
}

// GetFirstValues computes the first moving average value.
func (ema *EMASeries) GetFirstValues() (x, y float64) {
	if ema.InnerSeries == nil {
		return
	}
	if len(ema.cache) == 0 {
		ema.ensureCachedValues()
	}
	x, _ = ema.InnerSeries.GetValues(0)
	y = ema.cache[0]
	return
}

// GetLastValues computes the last moving average value but walking back window size samples,
// and recomputing the last moving average chunk.
func (ema *EMASeries) GetLastValues() (x, y float64) {
	if ema.InnerSeries == nil {
		return
	}
	if len(ema.cache) == 0 {
		ema.ensureCachedValues()
	}
	lastIndex := ema.InnerSeries.Len() - 1
	x, _ = ema.InnerSeries.GetValues(lastIndex)
	y = ema.cache[lastIndex]
	return
}

func (ema *EMASeries) ensureCachedValues() {
	seriesLength := ema.InnerSeries.Len()
	ema.cache = make([]float64, seriesLength)
	sigma := ema.GetSigma()
	for x := 0; x < seriesLength; x++ {
		_, y := ema.InnerSeries.GetValues(x)
		if x == 0 {
			ema.cache[x] = y
			continue
		}
		previousEMA := ema.cache[x-1]
		ema.cache[x] = ((y - previousEMA) * sigma) + previousEMA
	}
}

// Render renders the series.
func (ema *EMASeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	style := ema.Style.InheritFrom(defaults)
	Draw.LineSeries(r, canvasBox, xrange, yrange, style, ema)
}

// Validate validates the series.
func (ema *EMASeries) Validate() error {
	if ema.InnerSeries == nil {
		return fmt.Errorf("ema series requires InnerSeries to be set")
	}
	return nil
}
