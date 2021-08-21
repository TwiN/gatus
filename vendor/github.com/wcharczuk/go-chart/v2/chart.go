package chart

import (
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/golang/freetype/truetype"
)

// Chart is what we're drawing.
type Chart struct {
	Title      string
	TitleStyle Style

	ColorPalette ColorPalette

	Width  int
	Height int
	DPI    float64

	Background Style
	Canvas     Style

	XAxis          XAxis
	YAxis          YAxis
	YAxisSecondary YAxis

	Font        *truetype.Font
	defaultFont *truetype.Font

	Series   []Series
	Elements []Renderable

	Log Logger
}

// GetDPI returns the dpi for the chart.
func (c Chart) GetDPI(defaults ...float64) float64 {
	if c.DPI == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultDPI
	}
	return c.DPI
}

// GetFont returns the text font.
func (c Chart) GetFont() *truetype.Font {
	if c.Font == nil {
		return c.defaultFont
	}
	return c.Font
}

// GetWidth returns the chart width or the default value.
func (c Chart) GetWidth() int {
	if c.Width == 0 {
		return DefaultChartWidth
	}
	return c.Width
}

// GetHeight returns the chart height or the default value.
func (c Chart) GetHeight() int {
	if c.Height == 0 {
		return DefaultChartHeight
	}
	return c.Height
}

// Render renders the chart with the given renderer to the given io.Writer.
func (c Chart) Render(rp RendererProvider, w io.Writer) error {
	if len(c.Series) == 0 {
		return errors.New("please provide at least one series")
	}
	if err := c.checkHasVisibleSeries(); err != nil {
		return err
	}

	c.YAxisSecondary.AxisType = YAxisSecondary

	r, err := rp(c.GetWidth(), c.GetHeight())
	if err != nil {
		return err
	}

	if c.Font == nil {
		defaultFont, err := GetDefaultFont()
		if err != nil {
			return err
		}
		c.defaultFont = defaultFont
	}
	r.SetDPI(c.GetDPI(DefaultDPI))

	c.drawBackground(r)

	var xt, yt, yta []Tick
	xr, yr, yra := c.getRanges()
	canvasBox := c.getDefaultCanvasBox()
	xf, yf, yfa := c.getValueFormatters()

	Debugf(c.Log, "chart; canvas box: %v", canvasBox)

	xr, yr, yra = c.setRangeDomains(canvasBox, xr, yr, yra)

	err = c.checkRanges(xr, yr, yra)
	if err != nil {
		r.Save(w)
		return err
	}

	if c.hasAxes() {
		xt, yt, yta = c.getAxesTicks(r, xr, yr, yra, xf, yf, yfa)
		canvasBox = c.getAxesAdjustedCanvasBox(r, canvasBox, xr, yr, yra, xt, yt, yta)
		xr, yr, yra = c.setRangeDomains(canvasBox, xr, yr, yra)

		Debugf(c.Log, "chart; axes adjusted canvas box: %v", canvasBox)

		// do a second pass in case things haven't settled yet.
		xt, yt, yta = c.getAxesTicks(r, xr, yr, yra, xf, yf, yfa)
		canvasBox = c.getAxesAdjustedCanvasBox(r, canvasBox, xr, yr, yra, xt, yt, yta)
		xr, yr, yra = c.setRangeDomains(canvasBox, xr, yr, yra)
	}

	if c.hasAnnotationSeries() {
		canvasBox = c.getAnnotationAdjustedCanvasBox(r, canvasBox, xr, yr, yra, xf, yf, yfa)
		xr, yr, yra = c.setRangeDomains(canvasBox, xr, yr, yra)
		xt, yt, yta = c.getAxesTicks(r, xr, yr, yra, xf, yf, yfa)

		Debugf(c.Log, "chart; annotation adjusted canvas box: %v", canvasBox)
	}

	c.drawCanvas(r, canvasBox)
	c.drawAxes(r, canvasBox, xr, yr, yra, xt, yt, yta)
	for index, series := range c.Series {
		c.drawSeries(r, canvasBox, xr, yr, yra, series, index)
	}

	c.drawTitle(r)

	for _, a := range c.Elements {
		a(r, canvasBox, c.styleDefaultsElements())
	}

	return r.Save(w)
}

func (c Chart) checkHasVisibleSeries() error {
	var style Style
	for _, s := range c.Series {
		style = s.GetStyle()
		if !style.Hidden {
			return nil
		}
	}
	return fmt.Errorf("chart render; must have (1) visible series")
}

func (c Chart) validateSeries() error {
	var err error
	for _, s := range c.Series {
		err = s.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c Chart) getRanges() (xrange, yrange, yrangeAlt Range) {
	var minx, maxx float64 = math.MaxFloat64, -math.MaxFloat64
	var miny, maxy float64 = math.MaxFloat64, -math.MaxFloat64
	var minya, maxya float64 = math.MaxFloat64, -math.MaxFloat64

	seriesMappedToSecondaryAxis := false

	// note: a possible future optimization is to not scan the series values if
	// all axis are represented by either custom ticks or custom ranges.
	for _, s := range c.Series {
		if !s.GetStyle().Hidden {
			seriesAxis := s.GetYAxis()
			if bvp, isBoundedValuesProvider := s.(BoundedValuesProvider); isBoundedValuesProvider {
				seriesLength := bvp.Len()
				for index := 0; index < seriesLength; index++ {
					vx, vy1, vy2 := bvp.GetBoundedValues(index)

					minx = math.Min(minx, vx)
					maxx = math.Max(maxx, vx)

					if seriesAxis == YAxisPrimary {
						miny = math.Min(miny, vy1)
						miny = math.Min(miny, vy2)
						maxy = math.Max(maxy, vy1)
						maxy = math.Max(maxy, vy2)
					} else if seriesAxis == YAxisSecondary {
						minya = math.Min(minya, vy1)
						minya = math.Min(minya, vy2)
						maxya = math.Max(maxya, vy1)
						maxya = math.Max(maxya, vy2)
						seriesMappedToSecondaryAxis = true
					}
				}
			} else if vp, isValuesProvider := s.(ValuesProvider); isValuesProvider {
				seriesLength := vp.Len()
				for index := 0; index < seriesLength; index++ {
					vx, vy := vp.GetValues(index)

					minx = math.Min(minx, vx)
					maxx = math.Max(maxx, vx)

					if seriesAxis == YAxisPrimary {
						miny = math.Min(miny, vy)
						maxy = math.Max(maxy, vy)
					} else if seriesAxis == YAxisSecondary {
						minya = math.Min(minya, vy)
						maxya = math.Max(maxya, vy)
						seriesMappedToSecondaryAxis = true
					}
				}
			}
		}
	}

	if c.XAxis.Range == nil {
		xrange = &ContinuousRange{}
	} else {
		xrange = c.XAxis.Range
	}

	if c.YAxis.Range == nil {
		yrange = &ContinuousRange{}
	} else {
		yrange = c.YAxis.Range
	}

	if c.YAxisSecondary.Range == nil {
		yrangeAlt = &ContinuousRange{}
	} else {
		yrangeAlt = c.YAxisSecondary.Range
	}

	if len(c.XAxis.Ticks) > 0 {
		tickMin, tickMax := math.MaxFloat64, -math.MaxFloat64
		for _, t := range c.XAxis.Ticks {
			tickMin = math.Min(tickMin, t.Value)
			tickMax = math.Max(tickMax, t.Value)
		}
		xrange.SetMin(tickMin)
		xrange.SetMax(tickMax)
	} else if xrange.IsZero() {
		xrange.SetMin(minx)
		xrange.SetMax(maxx)
	}

	if len(c.YAxis.Ticks) > 0 {
		tickMin, tickMax := math.MaxFloat64, -math.MaxFloat64
		for _, t := range c.YAxis.Ticks {
			tickMin = math.Min(tickMin, t.Value)
			tickMax = math.Max(tickMax, t.Value)
		}
		yrange.SetMin(tickMin)
		yrange.SetMax(tickMax)
	} else if yrange.IsZero() {
		yrange.SetMin(miny)
		yrange.SetMax(maxy)

		if !c.YAxis.Style.Hidden {
			delta := yrange.GetDelta()
			roundTo := GetRoundToForDelta(delta)
			rmin, rmax := RoundDown(yrange.GetMin(), roundTo), RoundUp(yrange.GetMax(), roundTo)

			yrange.SetMin(rmin)
			yrange.SetMax(rmax)
		}
	}

	if len(c.YAxisSecondary.Ticks) > 0 {
		tickMin, tickMax := math.MaxFloat64, -math.MaxFloat64
		for _, t := range c.YAxis.Ticks {
			tickMin = math.Min(tickMin, t.Value)
			tickMax = math.Max(tickMax, t.Value)
		}
		yrangeAlt.SetMin(tickMin)
		yrangeAlt.SetMax(tickMax)
	} else if seriesMappedToSecondaryAxis && yrangeAlt.IsZero() {
		yrangeAlt.SetMin(minya)
		yrangeAlt.SetMax(maxya)

		if !c.YAxisSecondary.Style.Hidden {
			delta := yrangeAlt.GetDelta()
			roundTo := GetRoundToForDelta(delta)
			rmin, rmax := RoundDown(yrangeAlt.GetMin(), roundTo), RoundUp(yrangeAlt.GetMax(), roundTo)
			yrangeAlt.SetMin(rmin)
			yrangeAlt.SetMax(rmax)
		}
	}

	return
}

func (c Chart) checkRanges(xr, yr, yra Range) error {
	Debugf(c.Log, "checking xrange: %v", xr)
	xDelta := xr.GetDelta()
	if math.IsInf(xDelta, 0) {
		return errors.New("infinite x-range delta")
	}
	if math.IsNaN(xDelta) {
		return errors.New("nan x-range delta")
	}
	if xDelta == 0 {
		return errors.New("zero x-range delta; there needs to be at least (2) values")
	}

	Debugf(c.Log, "checking yrange: %v", yr)
	yDelta := yr.GetDelta()
	if math.IsInf(yDelta, 0) {
		return errors.New("infinite y-range delta")
	}
	if math.IsNaN(yDelta) {
		return errors.New("nan y-range delta")
	}

	if c.hasSecondarySeries() {
		Debugf(c.Log, "checking secondary yrange: %v", yra)
		yraDelta := yra.GetDelta()
		if math.IsInf(yraDelta, 0) {
			return errors.New("infinite secondary y-range delta")
		}
		if math.IsNaN(yraDelta) {
			return errors.New("nan secondary y-range delta")
		}
	}

	return nil
}

func (c Chart) getDefaultCanvasBox() Box {
	return c.Box()
}

func (c Chart) getValueFormatters() (x, y, ya ValueFormatter) {
	for _, s := range c.Series {
		if vfp, isVfp := s.(ValueFormatterProvider); isVfp {
			sx, sy := vfp.GetValueFormatters()
			if s.GetYAxis() == YAxisPrimary {
				x = sx
				y = sy
			} else if s.GetYAxis() == YAxisSecondary {
				x = sx
				ya = sy
			}
		}
	}
	if c.XAxis.ValueFormatter != nil {
		x = c.XAxis.GetValueFormatter()
	}
	if c.YAxis.ValueFormatter != nil {
		y = c.YAxis.GetValueFormatter()
	}
	if c.YAxisSecondary.ValueFormatter != nil {
		ya = c.YAxisSecondary.GetValueFormatter()
	}
	return
}

func (c Chart) hasAxes() bool {
	return !c.XAxis.Style.Hidden || !c.YAxis.Style.Hidden || !c.YAxisSecondary.Style.Hidden
}

func (c Chart) getAxesTicks(r Renderer, xr, yr, yar Range, xf, yf, yfa ValueFormatter) (xticks, yticks, yticksAlt []Tick) {
	if !c.XAxis.Style.Hidden {
		xticks = c.XAxis.GetTicks(r, xr, c.styleDefaultsAxes(), xf)
	}
	if !c.YAxis.Style.Hidden {
		yticks = c.YAxis.GetTicks(r, yr, c.styleDefaultsAxes(), yf)
	}
	if !c.YAxisSecondary.Style.Hidden {
		yticksAlt = c.YAxisSecondary.GetTicks(r, yar, c.styleDefaultsAxes(), yfa)
	}
	return
}

func (c Chart) getAxesAdjustedCanvasBox(r Renderer, canvasBox Box, xr, yr, yra Range, xticks, yticks, yticksAlt []Tick) Box {
	axesOuterBox := canvasBox.Clone()
	if !c.XAxis.Style.Hidden {
		axesBounds := c.XAxis.Measure(r, canvasBox, xr, c.styleDefaultsAxes(), xticks)
		Debugf(c.Log, "chart; x-axis measured %v", axesBounds)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}
	if !c.YAxis.Style.Hidden {
		axesBounds := c.YAxis.Measure(r, canvasBox, yr, c.styleDefaultsAxes(), yticks)
		Debugf(c.Log, "chart; y-axis measured %v", axesBounds)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}
	if !c.YAxisSecondary.Style.Hidden && c.hasSecondarySeries() {
		axesBounds := c.YAxisSecondary.Measure(r, canvasBox, yra, c.styleDefaultsAxes(), yticksAlt)
		Debugf(c.Log, "chart; y-axis secondary measured %v", axesBounds)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}

	return canvasBox.OuterConstrain(c.Box(), axesOuterBox)
}

func (c Chart) setRangeDomains(canvasBox Box, xr, yr, yra Range) (Range, Range, Range) {
	xr.SetDomain(canvasBox.Width())
	yr.SetDomain(canvasBox.Height())
	yra.SetDomain(canvasBox.Height())
	return xr, yr, yra
}

func (c Chart) hasAnnotationSeries() bool {
	for _, s := range c.Series {
		if as, isAnnotationSeries := s.(AnnotationSeries); isAnnotationSeries {
			if !as.GetStyle().Hidden {
				return true
			}
		}
	}
	return false
}

func (c Chart) hasSecondarySeries() bool {
	for _, s := range c.Series {
		if s.GetYAxis() == YAxisSecondary {
			return true
		}
	}
	return false
}

func (c Chart) getAnnotationAdjustedCanvasBox(r Renderer, canvasBox Box, xr, yr, yra Range, xf, yf, yfa ValueFormatter) Box {
	annotationSeriesBox := canvasBox.Clone()
	for seriesIndex, s := range c.Series {
		if as, isAnnotationSeries := s.(AnnotationSeries); isAnnotationSeries {
			if !as.GetStyle().Hidden {
				style := c.styleDefaultsSeries(seriesIndex)
				var annotationBounds Box
				if as.YAxis == YAxisPrimary {
					annotationBounds = as.Measure(r, canvasBox, xr, yr, style)
				} else if as.YAxis == YAxisSecondary {
					annotationBounds = as.Measure(r, canvasBox, xr, yra, style)
				}

				annotationSeriesBox = annotationSeriesBox.Grow(annotationBounds)
			}
		}
	}

	return canvasBox.OuterConstrain(c.Box(), annotationSeriesBox)
}

func (c Chart) getBackgroundStyle() Style {
	return c.Background.InheritFrom(c.styleDefaultsBackground())
}

func (c Chart) drawBackground(r Renderer) {
	Draw.Box(r, Box{
		Right:  c.GetWidth(),
		Bottom: c.GetHeight(),
	}, c.getBackgroundStyle())
}

func (c Chart) getCanvasStyle() Style {
	return c.Canvas.InheritFrom(c.styleDefaultsCanvas())
}

func (c Chart) drawCanvas(r Renderer, canvasBox Box) {
	Draw.Box(r, canvasBox, c.getCanvasStyle())
}

func (c Chart) drawAxes(r Renderer, canvasBox Box, xrange, yrange, yrangeAlt Range, xticks, yticks, yticksAlt []Tick) {
	if !c.XAxis.Style.Hidden {
		c.XAxis.Render(r, canvasBox, xrange, c.styleDefaultsAxes(), xticks)
	}
	if !c.YAxis.Style.Hidden {
		c.YAxis.Render(r, canvasBox, yrange, c.styleDefaultsAxes(), yticks)
	}
	if !c.YAxisSecondary.Style.Hidden {
		c.YAxisSecondary.Render(r, canvasBox, yrangeAlt, c.styleDefaultsAxes(), yticksAlt)
	}
}

func (c Chart) drawSeries(r Renderer, canvasBox Box, xrange, yrange, yrangeAlt Range, s Series, seriesIndex int) {
	if !s.GetStyle().Hidden {
		if s.GetYAxis() == YAxisPrimary {
			s.Render(r, canvasBox, xrange, yrange, c.styleDefaultsSeries(seriesIndex))
		} else if s.GetYAxis() == YAxisSecondary {
			s.Render(r, canvasBox, xrange, yrangeAlt, c.styleDefaultsSeries(seriesIndex))
		}
	}
}

func (c Chart) drawTitle(r Renderer) {
	if len(c.Title) > 0 && !c.TitleStyle.Hidden {
		r.SetFont(c.TitleStyle.GetFont(c.GetFont()))
		r.SetFontColor(c.TitleStyle.GetFontColor(c.GetColorPalette().TextColor()))
		titleFontSize := c.TitleStyle.GetFontSize(DefaultTitleFontSize)
		r.SetFontSize(titleFontSize)

		textBox := r.MeasureText(c.Title)

		textWidth := textBox.Width()
		textHeight := textBox.Height()

		titleX := (c.GetWidth() >> 1) - (textWidth >> 1)
		titleY := c.TitleStyle.Padding.GetTop(DefaultTitleTop) + textHeight

		r.Text(c.Title, titleX, titleY)
	}
}

func (c Chart) styleDefaultsBackground() Style {
	return Style{
		FillColor:   c.GetColorPalette().BackgroundColor(),
		StrokeColor: c.GetColorPalette().BackgroundStrokeColor(),
		StrokeWidth: DefaultBackgroundStrokeWidth,
	}
}

func (c Chart) styleDefaultsCanvas() Style {
	return Style{
		FillColor:   c.GetColorPalette().CanvasColor(),
		StrokeColor: c.GetColorPalette().CanvasStrokeColor(),
		StrokeWidth: DefaultCanvasStrokeWidth,
	}
}

func (c Chart) styleDefaultsSeries(seriesIndex int) Style {
	return Style{
		DotColor:    c.GetColorPalette().GetSeriesColor(seriesIndex),
		StrokeColor: c.GetColorPalette().GetSeriesColor(seriesIndex),
		StrokeWidth: DefaultSeriesLineWidth,
		Font:        c.GetFont(),
		FontSize:    DefaultFontSize,
	}
}

func (c Chart) styleDefaultsAxes() Style {
	return Style{
		Font:        c.GetFont(),
		FontColor:   c.GetColorPalette().TextColor(),
		FontSize:    DefaultAxisFontSize,
		StrokeColor: c.GetColorPalette().AxisStrokeColor(),
		StrokeWidth: DefaultAxisLineWidth,
	}
}

func (c Chart) styleDefaultsElements() Style {
	return Style{
		Font: c.GetFont(),
	}
}

// GetColorPalette returns the color palette for the chart.
func (c Chart) GetColorPalette() ColorPalette {
	if c.ColorPalette != nil {
		return c.ColorPalette
	}
	return DefaultColorPalette
}

// Box returns the chart bounds as a box.
func (c Chart) Box() Box {
	dpr := c.Background.Padding.GetRight(DefaultBackgroundPadding.Right)
	dpb := c.Background.Padding.GetBottom(DefaultBackgroundPadding.Bottom)

	return Box{
		Top:    c.Background.Padding.GetTop(DefaultBackgroundPadding.Top),
		Left:   c.Background.Padding.GetLeft(DefaultBackgroundPadding.Left),
		Right:  c.GetWidth() - dpr,
		Bottom: c.GetHeight() - dpb,
	}
}
