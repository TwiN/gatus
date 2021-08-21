package chart

import (
	"math"
)

// HideYAxis hides a y-axis.
func HideYAxis() YAxis {
	return YAxis{
		Style: Hidden(),
	}
}

// YAxis is a veritcal rule of the range.
// There can be (2) y-axes; a primary and secondary.
type YAxis struct {
	Name      string
	NameStyle Style

	Style Style

	Zero GridLine

	AxisType  YAxisType
	Ascending bool

	ValueFormatter ValueFormatter
	Range          Range

	TickStyle Style
	Ticks     []Tick

	GridLines      []GridLine
	GridMajorStyle Style
	GridMinorStyle Style
}

// GetName returns the name.
func (ya YAxis) GetName() string {
	return ya.Name
}

// GetNameStyle returns the name style.
func (ya YAxis) GetNameStyle() Style {
	return ya.NameStyle
}

// GetStyle returns the style.
func (ya YAxis) GetStyle() Style {
	return ya.Style
}

// GetValueFormatter returns the value formatter for the axis.
func (ya YAxis) GetValueFormatter() ValueFormatter {
	if ya.ValueFormatter != nil {
		return ya.ValueFormatter
	}
	return FloatValueFormatter
}

// GetTickStyle returns the tick style.
func (ya YAxis) GetTickStyle() Style {
	return ya.TickStyle
}

// GetTicks returns the ticks for a series.
// The coalesce priority is:
// 	- User Supplied Ticks (i.e. Ticks array on the axis itself).
// 	- Range ticks (i.e. if the range provides ticks).
//	- Generating continuous ticks based on minimum spacing and canvas width.
func (ya YAxis) GetTicks(r Renderer, ra Range, defaults Style, vf ValueFormatter) []Tick {
	if len(ya.Ticks) > 0 {
		return ya.Ticks
	}
	if tp, isTickProvider := ra.(TicksProvider); isTickProvider {
		return tp.GetTicks(r, defaults, vf)
	}
	tickStyle := ya.Style.InheritFrom(defaults)
	return GenerateContinuousTicks(r, ra, true, tickStyle, vf)
}

// GetGridLines returns the gridlines for the axis.
func (ya YAxis) GetGridLines(ticks []Tick) []GridLine {
	if len(ya.GridLines) > 0 {
		return ya.GridLines
	}
	return GenerateGridLines(ticks, ya.GridMajorStyle, ya.GridMinorStyle)
}

// Measure returns the bounds of the axis.
func (ya YAxis) Measure(r Renderer, canvasBox Box, ra Range, defaults Style, ticks []Tick) Box {
	var tx int
	if ya.AxisType == YAxisPrimary {
		tx = canvasBox.Right + DefaultYAxisMargin
	} else if ya.AxisType == YAxisSecondary {
		tx = canvasBox.Left - DefaultYAxisMargin
	}

	ya.TickStyle.InheritFrom(ya.Style.InheritFrom(defaults)).WriteToRenderer(r)
	var minx, maxx, miny, maxy = math.MaxInt32, 0, math.MaxInt32, 0
	var maxTextHeight int
	for _, t := range ticks {
		v := t.Value
		ly := canvasBox.Bottom - ra.Translate(v)

		tb := r.MeasureText(t.Label)
		tbh2 := tb.Height() >> 1
		finalTextX := tx
		if ya.AxisType == YAxisSecondary {
			finalTextX = tx - tb.Width()
		}

		maxTextHeight = MaxInt(tb.Height(), maxTextHeight)

		if ya.AxisType == YAxisPrimary {
			minx = canvasBox.Right
			maxx = MaxInt(maxx, tx+tb.Width())
		} else if ya.AxisType == YAxisSecondary {
			minx = MinInt(minx, finalTextX)
			maxx = MaxInt(maxx, tx)
		}

		miny = MinInt(miny, ly-tbh2)
		maxy = MaxInt(maxy, ly+tbh2)
	}

	if !ya.NameStyle.Hidden && len(ya.Name) > 0 {
		maxx += (DefaultYAxisMargin + maxTextHeight)
	}

	return Box{
		Top:    miny,
		Left:   minx,
		Right:  maxx,
		Bottom: maxy,
	}
}

// Render renders the axis.
func (ya YAxis) Render(r Renderer, canvasBox Box, ra Range, defaults Style, ticks []Tick) {
	tickStyle := ya.TickStyle.InheritFrom(ya.Style.InheritFrom(defaults))
	tickStyle.WriteToRenderer(r)

	sw := tickStyle.GetStrokeWidth(defaults.StrokeWidth)

	var lx int
	var tx int
	if ya.AxisType == YAxisPrimary {
		lx = canvasBox.Right + int(sw)
		tx = lx + DefaultYAxisMargin
	} else if ya.AxisType == YAxisSecondary {
		lx = canvasBox.Left - int(sw)
		tx = lx - DefaultYAxisMargin
	}

	r.MoveTo(lx, canvasBox.Bottom)
	r.LineTo(lx, canvasBox.Top)
	r.Stroke()

	var maxTextWidth int
	var finalTextX, finalTextY int
	for _, t := range ticks {
		v := t.Value
		ly := canvasBox.Bottom - ra.Translate(v)

		tb := Draw.MeasureText(r, t.Label, tickStyle)

		if tb.Width() > maxTextWidth {
			maxTextWidth = tb.Width()
		}

		if ya.AxisType == YAxisSecondary {
			finalTextX = tx - tb.Width()
		} else {
			finalTextX = tx
		}

		if tickStyle.TextRotationDegrees == 0 {
			finalTextY = ly + tb.Height()>>1
		} else {
			finalTextY = ly
		}

		tickStyle.WriteToRenderer(r)

		r.MoveTo(lx, ly)
		if ya.AxisType == YAxisPrimary {
			r.LineTo(lx+DefaultHorizontalTickWidth, ly)
		} else if ya.AxisType == YAxisSecondary {
			r.LineTo(lx-DefaultHorizontalTickWidth, ly)
		}
		r.Stroke()

		Draw.Text(r, t.Label, finalTextX, finalTextY, tickStyle)
	}

	nameStyle := ya.NameStyle.InheritFrom(defaults.InheritFrom(Style{TextRotationDegrees: 90}))
	if !ya.NameStyle.Hidden && len(ya.Name) > 0 {
		nameStyle.GetTextOptions().WriteToRenderer(r)
		tb := Draw.MeasureText(r, ya.Name, nameStyle)

		var tx int
		if ya.AxisType == YAxisPrimary {
			tx = canvasBox.Right + int(sw) + DefaultYAxisMargin + maxTextWidth + DefaultYAxisMargin
		} else if ya.AxisType == YAxisSecondary {
			tx = canvasBox.Left - (DefaultYAxisMargin + int(sw) + maxTextWidth + DefaultYAxisMargin)
		}

		var ty int
		if nameStyle.TextRotationDegrees == 0 {
			ty = canvasBox.Top + (canvasBox.Height()>>1 - tb.Width()>>1)
		} else {
			ty = canvasBox.Top + (canvasBox.Height()>>1 - tb.Height()>>1)
		}

		Draw.Text(r, ya.Name, tx, ty, nameStyle)
	}

	if !ya.Zero.Style.Hidden {
		ya.Zero.Render(r, canvasBox, ra, false, Style{})
	}

	if !ya.GridMajorStyle.Hidden || !ya.GridMinorStyle.Hidden {
		for _, gl := range ya.GetGridLines(ticks) {
			if (gl.IsMinor && !ya.GridMinorStyle.Hidden) || (!gl.IsMinor && !ya.GridMajorStyle.Hidden) {
				defaults := ya.GridMajorStyle
				if gl.IsMinor {
					defaults = ya.GridMinorStyle
				}
				gl.Render(r, canvasBox, ra, false, gl.Style.InheritFrom(defaults))
			}
		}
	}
}
