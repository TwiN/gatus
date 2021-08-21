package chart

import (
	"math"
)

var (
	// Draw contains helpers for drawing common objects.
	Draw = &draw{}
)

type draw struct{}

// LineSeries draws a line series with a renderer.
func (d draw) LineSeries(r Renderer, canvasBox Box, xrange, yrange Range, style Style, vs ValuesProvider) {
	if vs.Len() == 0 {
		return
	}

	cb := canvasBox.Bottom
	cl := canvasBox.Left

	v0x, v0y := vs.GetValues(0)
	x0 := cl + xrange.Translate(v0x)
	y0 := cb - yrange.Translate(v0y)

	yv0 := yrange.Translate(0)

	var vx, vy float64
	var x, y int

	if style.ShouldDrawStroke() && style.ShouldDrawFill() {
		style.GetFillOptions().WriteDrawingOptionsToRenderer(r)
		r.MoveTo(x0, y0)
		for i := 1; i < vs.Len(); i++ {
			vx, vy = vs.GetValues(i)
			x = cl + xrange.Translate(vx)
			y = cb - yrange.Translate(vy)
			r.LineTo(x, y)
		}
		r.LineTo(x, MinInt(cb, cb-yv0))
		r.LineTo(x0, MinInt(cb, cb-yv0))
		r.LineTo(x0, y0)
		r.Fill()
	}

	if style.ShouldDrawStroke() {
		style.GetStrokeOptions().WriteDrawingOptionsToRenderer(r)

		r.MoveTo(x0, y0)
		for i := 1; i < vs.Len(); i++ {
			vx, vy = vs.GetValues(i)
			x = cl + xrange.Translate(vx)
			y = cb - yrange.Translate(vy)
			r.LineTo(x, y)
		}
		r.Stroke()
	}

	if style.ShouldDrawDot() {
		defaultDotWidth := style.GetDotWidth()

		style.GetDotOptions().WriteDrawingOptionsToRenderer(r)
		for i := 0; i < vs.Len(); i++ {
			vx, vy = vs.GetValues(i)
			x = cl + xrange.Translate(vx)
			y = cb - yrange.Translate(vy)

			dotWidth := defaultDotWidth
			if style.DotWidthProvider != nil {
				dotWidth = style.DotWidthProvider(xrange, yrange, i, vx, vy)
			}

			if style.DotColorProvider != nil {
				dotColor := style.DotColorProvider(xrange, yrange, i, vx, vy)

				r.SetFillColor(dotColor)
				r.SetStrokeColor(dotColor)
			}

			r.Circle(dotWidth, x, y)
			r.FillStroke()
		}
	}
}

// BoundedSeries draws a series that implements BoundedValuesProvider.
func (d draw) BoundedSeries(r Renderer, canvasBox Box, xrange, yrange Range, style Style, bbs BoundedValuesProvider, drawOffsetIndexes ...int) {
	drawOffsetIndex := 0
	if len(drawOffsetIndexes) > 0 {
		drawOffsetIndex = drawOffsetIndexes[0]
	}

	cb := canvasBox.Bottom
	cl := canvasBox.Left

	v0x, v0y1, v0y2 := bbs.GetBoundedValues(0)
	x0 := cl + xrange.Translate(v0x)
	y0 := cb - yrange.Translate(v0y1)

	var vx, vy1, vy2 float64
	var x, y int

	xvalues := make([]float64, bbs.Len())
	xvalues[0] = v0x
	y2values := make([]float64, bbs.Len())
	y2values[0] = v0y2

	style.GetFillAndStrokeOptions().WriteToRenderer(r)
	r.MoveTo(x0, y0)
	for i := 1; i < bbs.Len(); i++ {
		vx, vy1, vy2 = bbs.GetBoundedValues(i)

		xvalues[i] = vx
		y2values[i] = vy2

		x = cl + xrange.Translate(vx)
		y = cb - yrange.Translate(vy1)
		if i > drawOffsetIndex {
			r.LineTo(x, y)
		} else {
			r.MoveTo(x, y)
		}
	}
	y = cb - yrange.Translate(vy2)
	r.LineTo(x, y)
	for i := bbs.Len() - 1; i >= drawOffsetIndex; i-- {
		vx, vy2 = xvalues[i], y2values[i]
		x = cl + xrange.Translate(vx)
		y = cb - yrange.Translate(vy2)
		r.LineTo(x, y)
	}
	r.Close()
	r.FillStroke()
}

// HistogramSeries draws a value provider as boxes from 0.
func (d draw) HistogramSeries(r Renderer, canvasBox Box, xrange, yrange Range, style Style, vs ValuesProvider, barWidths ...int) {
	if vs.Len() == 0 {
		return
	}

	//calculate bar width?
	seriesLength := vs.Len()
	barWidth := int(math.Floor(float64(xrange.GetDomain()) / float64(seriesLength)))
	if len(barWidths) > 0 {
		barWidth = barWidths[0]
	}

	cb := canvasBox.Bottom
	cl := canvasBox.Left

	//foreach datapoint, draw a box.
	for index := 0; index < seriesLength; index++ {
		vx, vy := vs.GetValues(index)
		y0 := yrange.Translate(0)
		x := cl + xrange.Translate(vx)
		y := yrange.Translate(vy)

		d.Box(r, Box{
			Top:    cb - y0,
			Left:   x - (barWidth >> 1),
			Right:  x + (barWidth >> 1),
			Bottom: cb - y,
		}, style)
	}
}

// MeasureAnnotation measures how big an annotation would be.
func (d draw) MeasureAnnotation(r Renderer, canvasBox Box, style Style, lx, ly int, label string) Box {
	style.WriteToRenderer(r)
	defer r.ResetStyle()

	textBox := r.MeasureText(label)
	textWidth := textBox.Width()
	textHeight := textBox.Height()
	halfTextHeight := textHeight >> 1

	pt := style.Padding.GetTop(DefaultAnnotationPadding.Top)
	pl := style.Padding.GetLeft(DefaultAnnotationPadding.Left)
	pr := style.Padding.GetRight(DefaultAnnotationPadding.Right)
	pb := style.Padding.GetBottom(DefaultAnnotationPadding.Bottom)

	strokeWidth := style.GetStrokeWidth()

	top := ly - (pt + halfTextHeight)
	right := lx + pl + pr + textWidth + DefaultAnnotationDeltaWidth + int(strokeWidth)
	bottom := ly + (pb + halfTextHeight)

	return Box{
		Top:    top,
		Left:   lx,
		Right:  right,
		Bottom: bottom,
	}
}

// Annotation draws an anotation with a renderer.
func (d draw) Annotation(r Renderer, canvasBox Box, style Style, lx, ly int, label string) {
	style.GetTextOptions().WriteToRenderer(r)
	defer r.ResetStyle()

	textBox := r.MeasureText(label)
	textWidth := textBox.Width()
	halfTextHeight := textBox.Height() >> 1

	style.GetFillAndStrokeOptions().WriteToRenderer(r)

	pt := style.Padding.GetTop(DefaultAnnotationPadding.Top)
	pl := style.Padding.GetLeft(DefaultAnnotationPadding.Left)
	pr := style.Padding.GetRight(DefaultAnnotationPadding.Right)
	pb := style.Padding.GetBottom(DefaultAnnotationPadding.Bottom)

	textX := lx + pl + DefaultAnnotationDeltaWidth
	textY := ly + halfTextHeight

	ltx := lx + DefaultAnnotationDeltaWidth
	lty := ly - (pt + halfTextHeight)

	rtx := lx + pl + pr + textWidth + DefaultAnnotationDeltaWidth
	rty := ly - (pt + halfTextHeight)

	rbx := lx + pl + pr + textWidth + DefaultAnnotationDeltaWidth
	rby := ly + (pb + halfTextHeight)

	lbx := lx + DefaultAnnotationDeltaWidth
	lby := ly + (pb + halfTextHeight)

	r.MoveTo(lx, ly)
	r.LineTo(ltx, lty)
	r.LineTo(rtx, rty)
	r.LineTo(rbx, rby)
	r.LineTo(lbx, lby)
	r.LineTo(lx, ly)
	r.Close()
	r.FillStroke()

	style.GetTextOptions().WriteToRenderer(r)
	r.Text(label, textX, textY)
}

// Box draws a box with a given style.
func (d draw) Box(r Renderer, b Box, s Style) {
	s.GetFillAndStrokeOptions().WriteToRenderer(r)
	defer r.ResetStyle()

	r.MoveTo(b.Left, b.Top)
	r.LineTo(b.Right, b.Top)
	r.LineTo(b.Right, b.Bottom)
	r.LineTo(b.Left, b.Bottom)
	r.LineTo(b.Left, b.Top)
	r.FillStroke()
}

func (d draw) BoxRotated(r Renderer, b Box, thetaDegrees float64, s Style) {
	d.BoxCorners(r, b.Corners().Rotate(thetaDegrees), s)
}

func (d draw) BoxCorners(r Renderer, bc BoxCorners, s Style) {
	s.GetFillAndStrokeOptions().WriteToRenderer(r)
	defer r.ResetStyle()

	r.MoveTo(bc.TopLeft.X, bc.TopLeft.Y)
	r.LineTo(bc.TopRight.X, bc.TopRight.Y)
	r.LineTo(bc.BottomRight.X, bc.BottomRight.Y)
	r.LineTo(bc.BottomLeft.X, bc.BottomLeft.Y)
	r.Close()
	r.FillStroke()
}

// DrawText draws text with a given style.
func (d draw) Text(r Renderer, text string, x, y int, style Style) {
	style.GetTextOptions().WriteToRenderer(r)
	defer r.ResetStyle()

	r.Text(text, x, y)
}

func (d draw) MeasureText(r Renderer, text string, style Style) Box {
	style.GetTextOptions().WriteToRenderer(r)
	defer r.ResetStyle()

	return r.MeasureText(text)
}

// TextWithin draws the text within a given box.
func (d draw) TextWithin(r Renderer, text string, box Box, style Style) {
	style.GetTextOptions().WriteToRenderer(r)
	defer r.ResetStyle()

	lines := Text.WrapFit(r, text, box.Width(), style)
	linesBox := Text.MeasureLines(r, lines, style)

	y := box.Top

	switch style.GetTextVerticalAlign() {
	case TextVerticalAlignBottom, TextVerticalAlignBaseline: // i have to build better baseline handling into measure text
		y = y - linesBox.Height()
	case TextVerticalAlignMiddle:
		y = y + (box.Height() >> 1) - (linesBox.Height() >> 1)
	case TextVerticalAlignMiddleBaseline:
		y = y + (box.Height() >> 1) - linesBox.Height()
	}

	var tx, ty int
	for _, line := range lines {
		lineBox := r.MeasureText(line)
		switch style.GetTextHorizontalAlign() {
		case TextHorizontalAlignCenter:
			tx = box.Left + ((box.Width() - lineBox.Width()) >> 1)
		case TextHorizontalAlignRight:
			tx = box.Right - lineBox.Width()
		default:
			tx = box.Left
		}
		if style.TextRotationDegrees == 0 {
			ty = y + lineBox.Height()
		} else {
			ty = y
		}

		r.Text(line, tx, ty)
		y += lineBox.Height() + style.GetTextLineSpacing()
	}
}
