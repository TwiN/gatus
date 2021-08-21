package chart

import (
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/golang/freetype/truetype"
)

// StackedBar is a bar within a StackedBarChart.
type StackedBar struct {
	Name   string
	Width  int
	Values []Value
}

// GetWidth returns the width of the bar.
func (sb StackedBar) GetWidth() int {
	if sb.Width == 0 {
		return 50
	}
	return sb.Width
}

// StackedBarChart is a chart that draws sections of a bar based on percentages.
type StackedBarChart struct {
	Title      string
	TitleStyle Style

	ColorPalette ColorPalette

	Width  int
	Height int
	DPI    float64

	Background Style
	Canvas     Style

	XAxis Style
	YAxis Style

	BarSpacing int

	Font        *truetype.Font
	defaultFont *truetype.Font

	IsHorizontal bool

	Bars     []StackedBar
	Elements []Renderable
}

// GetDPI returns the dpi for the chart.
func (sbc StackedBarChart) GetDPI(defaults ...float64) float64 {
	if sbc.DPI == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultDPI
	}
	return sbc.DPI
}

// GetFont returns the text font.
func (sbc StackedBarChart) GetFont() *truetype.Font {
	if sbc.Font == nil {
		return sbc.defaultFont
	}
	return sbc.Font
}

// GetWidth returns the chart width or the default value.
func (sbc StackedBarChart) GetWidth() int {
	if sbc.Width == 0 {
		return DefaultChartWidth
	}
	return sbc.Width
}

// GetHeight returns the chart height or the default value.
func (sbc StackedBarChart) GetHeight() int {
	if sbc.Height == 0 {
		return DefaultChartWidth
	}
	return sbc.Height
}

// GetBarSpacing returns the spacing between bars.
func (sbc StackedBarChart) GetBarSpacing() int {
	if sbc.BarSpacing == 0 {
		return 100
	}
	return sbc.BarSpacing
}

// Render renders the chart with the given renderer to the given io.Writer.
func (sbc StackedBarChart) Render(rp RendererProvider, w io.Writer) error {
	if len(sbc.Bars) == 0 {
		return errors.New("please provide at least one bar")
	}

	r, err := rp(sbc.GetWidth(), sbc.GetHeight())
	if err != nil {
		return err
	}

	if sbc.Font == nil {
		defaultFont, err := GetDefaultFont()
		if err != nil {
			return err
		}
		sbc.defaultFont = defaultFont
	}
	r.SetDPI(sbc.GetDPI(DefaultDPI))

	var canvasBox Box
	if sbc.IsHorizontal {
		canvasBox = sbc.getHorizontalAdjustedCanvasBox(r, sbc.getDefaultCanvasBox())
		sbc.drawCanvas(r, canvasBox)
		sbc.drawHorizontalBars(r, canvasBox)
		sbc.drawHorizontalXAxis(r, canvasBox)
		sbc.drawHorizontalYAxis(r, canvasBox)
	} else {
		canvasBox = sbc.getAdjustedCanvasBox(r, sbc.getDefaultCanvasBox())
		sbc.drawCanvas(r, canvasBox)
		sbc.drawBars(r, canvasBox)
		sbc.drawXAxis(r, canvasBox)
		sbc.drawYAxis(r, canvasBox)
	}

	sbc.drawTitle(r)
	for _, a := range sbc.Elements {
		a(r, canvasBox, sbc.styleDefaultsElements())
	}

	return r.Save(w)
}

func (sbc StackedBarChart) drawCanvas(r Renderer, canvasBox Box) {
	Draw.Box(r, canvasBox, sbc.getCanvasStyle())
}

func (sbc StackedBarChart) drawBars(r Renderer, canvasBox Box) {
	xoffset := canvasBox.Left
	for _, bar := range sbc.Bars {
		sbc.drawBar(r, canvasBox, xoffset, bar)
		xoffset += (sbc.GetBarSpacing() + bar.GetWidth())
	}
}

func (sbc StackedBarChart) drawHorizontalBars(r Renderer, canvasBox Box) {
	yOffset := canvasBox.Top
	for _, bar := range sbc.Bars {
		sbc.drawHorizontalBar(r, canvasBox, yOffset, bar)
		yOffset += sbc.GetBarSpacing() + bar.GetWidth()
	}
}

func (sbc StackedBarChart) drawBar(r Renderer, canvasBox Box, xoffset int, bar StackedBar) int {
	barSpacing2 := sbc.GetBarSpacing() >> 1
	bxl := xoffset + barSpacing2
	bxr := bxl + bar.GetWidth()

	normalizedBarComponents := Values(bar.Values).Normalize()
	yoffset := canvasBox.Top
	for index, bv := range normalizedBarComponents {
		barHeight := int(math.Ceil(bv.Value * float64(canvasBox.Height())))
		barBox := Box{
			Top:    yoffset,
			Left:   bxl,
			Right:  bxr,
			Bottom: MinInt(yoffset+barHeight, canvasBox.Bottom-DefaultStrokeWidth),
		}
		Draw.Box(r, barBox, bv.Style.InheritFrom(sbc.styleDefaultsStackedBarValue(index)))
		yoffset += barHeight
	}

	// draw the labels
	yoffset = canvasBox.Top
	var lx, ly int
	for index, bv := range normalizedBarComponents {
		barHeight := int(math.Ceil(bv.Value * float64(canvasBox.Height())))

		if len(bv.Label) > 0 {
			lx = bxl + ((bxr - bxl) / 2)
			ly = yoffset + (barHeight / 2)

			bv.Style.InheritFrom(sbc.styleDefaultsStackedBarValue(index)).WriteToRenderer(r)
			tb := r.MeasureText(bv.Label)
			lx = lx - (tb.Width() >> 1)
			ly = ly + (tb.Height() >> 1)

			if lx < 0 {
				lx = 0
			}
			if ly < 0 {
				lx = 0
			}

			r.Text(bv.Label, lx, ly)
		}
		yoffset += barHeight
	}

	return bxr
}

func (sbc StackedBarChart) drawHorizontalBar(r Renderer, canvasBox Box, yoffset int, bar StackedBar) {
	halfBarSpacing := sbc.GetBarSpacing() >> 1

	boxTop := yoffset + halfBarSpacing
	boxBottom := boxTop + bar.GetWidth()

	normalizedBarComponents := Values(bar.Values).Normalize()

	xOffset := canvasBox.Right
	for index, bv := range normalizedBarComponents {
		barHeight := int(math.Ceil(bv.Value * float64(canvasBox.Width())))
		barBox := Box{
			Top:    boxTop,
			Left:   MinInt(xOffset-barHeight, canvasBox.Left+DefaultStrokeWidth),
			Right:  xOffset,
			Bottom: boxBottom,
		}
		Draw.Box(r, barBox, bv.Style.InheritFrom(sbc.styleDefaultsStackedBarValue(index)))
		xOffset -= barHeight
	}

	// draw the labels
	xOffset = canvasBox.Right
	var lx, ly int
	for index, bv := range normalizedBarComponents {
		barHeight := int(math.Ceil(bv.Value * float64(canvasBox.Width())))

		if len(bv.Label) > 0 {
			lx = xOffset - (barHeight / 2)
			ly = boxTop + ((boxBottom - boxTop) / 2)

			bv.Style.InheritFrom(sbc.styleDefaultsStackedBarValue(index)).WriteToRenderer(r)
			tb := r.MeasureText(bv.Label)
			lx = lx - (tb.Width() >> 1)
			ly = ly + (tb.Height() >> 1)

			if lx < 0 {
				lx = 0
			}
			if ly < 0 {
				lx = 0
			}

			r.Text(bv.Label, lx, ly)
		}
		xOffset -= barHeight
	}
}

func (sbc StackedBarChart) drawXAxis(r Renderer, canvasBox Box) {
	if !sbc.XAxis.Hidden {
		axisStyle := sbc.XAxis.InheritFrom(sbc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Right, canvasBox.Bottom)
		r.Stroke()

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Left, canvasBox.Bottom+DefaultVerticalTickHeight)
		r.Stroke()

		cursor := canvasBox.Left
		for _, bar := range sbc.Bars {

			barLabelBox := Box{
				Top:    canvasBox.Bottom + DefaultXAxisMargin,
				Left:   cursor,
				Right:  cursor + bar.GetWidth() + sbc.GetBarSpacing(),
				Bottom: sbc.GetHeight(),
			}
			if len(bar.Name) > 0 {
				Draw.TextWithin(r, bar.Name, barLabelBox, axisStyle)
			}
			axisStyle.WriteToRenderer(r)
			r.MoveTo(barLabelBox.Right, canvasBox.Bottom)
			r.LineTo(barLabelBox.Right, canvasBox.Bottom+DefaultVerticalTickHeight)
			r.Stroke()
			cursor += bar.GetWidth() + sbc.GetBarSpacing()
		}
	}
}

func (sbc StackedBarChart) drawHorizontalXAxis(r Renderer, canvasBox Box) {
	if !sbc.XAxis.Hidden {
		axisStyle := sbc.XAxis.InheritFrom(sbc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)
		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Right, canvasBox.Bottom)
		r.Stroke()

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Left, canvasBox.Bottom+DefaultVerticalTickHeight)
		r.Stroke()

		ticks := LinearRangeWithStep(0.0, 1.0, 0.2)
		for _, t := range ticks {
			axisStyle.GetStrokeOptions().WriteToRenderer(r)
			tx := canvasBox.Left + int(t*float64(canvasBox.Width()))
			r.MoveTo(tx, canvasBox.Bottom)
			r.LineTo(tx, canvasBox.Bottom+DefaultVerticalTickHeight)
			r.Stroke()

			axisStyle.GetTextOptions().WriteToRenderer(r)
			text := fmt.Sprintf("%0.0f%%", t*100)

			textBox := r.MeasureText(text)
			textX := tx - (textBox.Width() >> 1)
			textY := canvasBox.Bottom + DefaultXAxisMargin + 10

			if t == 1 {
				textX = canvasBox.Right - textBox.Width()
			}

			Draw.Text(r, text, textX, textY, axisStyle)
		}
	}
}

func (sbc StackedBarChart) drawYAxis(r Renderer, canvasBox Box) {
	if !sbc.YAxis.Hidden {
		axisStyle := sbc.YAxis.InheritFrom(sbc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)
		r.MoveTo(canvasBox.Right, canvasBox.Top)
		r.LineTo(canvasBox.Right, canvasBox.Bottom)
		r.Stroke()

		r.MoveTo(canvasBox.Right, canvasBox.Bottom)
		r.LineTo(canvasBox.Right+DefaultHorizontalTickWidth, canvasBox.Bottom)
		r.Stroke()

		ticks := LinearRangeWithStep(0.0, 1.0, 0.2)
		for _, t := range ticks {
			axisStyle.GetStrokeOptions().WriteToRenderer(r)
			ty := canvasBox.Bottom - int(t*float64(canvasBox.Height()))
			r.MoveTo(canvasBox.Right, ty)
			r.LineTo(canvasBox.Right+DefaultHorizontalTickWidth, ty)
			r.Stroke()

			axisStyle.GetTextOptions().WriteToRenderer(r)
			text := fmt.Sprintf("%0.0f%%", t*100)

			tb := r.MeasureText(text)
			Draw.Text(r, text, canvasBox.Right+DefaultYAxisMargin+5, ty+(tb.Height()>>1), axisStyle)
		}
	}
}

func (sbc StackedBarChart) drawHorizontalYAxis(r Renderer, canvasBox Box) {
	if !sbc.YAxis.Hidden {
		axisStyle := sbc.YAxis.InheritFrom(sbc.styleDefaultsHorizontalAxes())
		axisStyle.WriteToRenderer(r)

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Left, canvasBox.Top)
		r.Stroke()

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Left-DefaultHorizontalTickWidth, canvasBox.Bottom)
		r.Stroke()

		cursor := canvasBox.Top
		for _, bar := range sbc.Bars {
			barLabelBox := Box{
				Top:    cursor,
				Left:   0,
				Right:  canvasBox.Left - DefaultYAxisMargin,
				Bottom: cursor + bar.GetWidth() + sbc.GetBarSpacing(),
			}
			if len(bar.Name) > 0 {
				Draw.TextWithin(r, bar.Name, barLabelBox, axisStyle)
			}
			axisStyle.WriteToRenderer(r)
			r.MoveTo(canvasBox.Left, barLabelBox.Bottom)
			r.LineTo(canvasBox.Left-DefaultHorizontalTickWidth, barLabelBox.Bottom)
			r.Stroke()
			cursor += bar.GetWidth() + sbc.GetBarSpacing()
		}
	}
}

func (sbc StackedBarChart) drawTitle(r Renderer) {
	if len(sbc.Title) > 0 && !sbc.TitleStyle.Hidden {
		r.SetFont(sbc.TitleStyle.GetFont(sbc.GetFont()))
		r.SetFontColor(sbc.TitleStyle.GetFontColor(sbc.GetColorPalette().TextColor()))
		titleFontSize := sbc.TitleStyle.GetFontSize(DefaultTitleFontSize)
		r.SetFontSize(titleFontSize)

		textBox := r.MeasureText(sbc.Title)

		textWidth := textBox.Width()
		textHeight := textBox.Height()

		titleX := (sbc.GetWidth() >> 1) - (textWidth >> 1)
		titleY := sbc.TitleStyle.Padding.GetTop(DefaultTitleTop) + textHeight

		r.Text(sbc.Title, titleX, titleY)
	}
}

func (sbc StackedBarChart) getCanvasStyle() Style {
	return sbc.Canvas.InheritFrom(sbc.styleDefaultsCanvas())
}

func (sbc StackedBarChart) styleDefaultsCanvas() Style {
	return Style{
		FillColor:   sbc.GetColorPalette().CanvasColor(),
		StrokeColor: sbc.GetColorPalette().CanvasStrokeColor(),
		StrokeWidth: DefaultCanvasStrokeWidth,
	}
}

// GetColorPalette returns the color palette for the chart.
func (sbc StackedBarChart) GetColorPalette() ColorPalette {
	if sbc.ColorPalette != nil {
		return sbc.ColorPalette
	}
	return AlternateColorPalette
}

func (sbc StackedBarChart) getDefaultCanvasBox() Box {
	return sbc.Box()
}

func (sbc StackedBarChart) getAdjustedCanvasBox(r Renderer, canvasBox Box) Box {
	var totalWidth int
	for _, bar := range sbc.Bars {
		totalWidth += bar.GetWidth() + sbc.GetBarSpacing()
	}

	if !sbc.XAxis.Hidden {
		xaxisHeight := DefaultVerticalTickHeight

		axisStyle := sbc.XAxis.InheritFrom(sbc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		cursor := canvasBox.Left
		for _, bar := range sbc.Bars {
			if len(bar.Name) > 0 {
				barLabelBox := Box{
					Top:    canvasBox.Bottom + DefaultXAxisMargin,
					Left:   cursor,
					Right:  cursor + bar.GetWidth() + sbc.GetBarSpacing(),
					Bottom: sbc.GetHeight(),
				}
				lines := Text.WrapFit(r, bar.Name, barLabelBox.Width(), axisStyle)
				linesBox := Text.MeasureLines(r, lines, axisStyle)

				xaxisHeight = MaxInt(linesBox.Height()+(2*DefaultXAxisMargin), xaxisHeight)
			}
		}
		return Box{
			Top:    canvasBox.Top,
			Left:   canvasBox.Left,
			Right:  canvasBox.Left + totalWidth,
			Bottom: sbc.GetHeight() - xaxisHeight,
		}
	}
	return Box{
		Top:    canvasBox.Top,
		Left:   canvasBox.Left,
		Right:  canvasBox.Left + totalWidth,
		Bottom: canvasBox.Bottom,
	}

}

func (sbc StackedBarChart) getHorizontalAdjustedCanvasBox(r Renderer, canvasBox Box) Box {
	var totalHeight int
	for _, bar := range sbc.Bars {
		totalHeight += bar.GetWidth() + sbc.GetBarSpacing()
	}

	if !sbc.YAxis.Hidden {
		yAxisWidth := DefaultHorizontalTickWidth

		axisStyle := sbc.YAxis.InheritFrom(sbc.styleDefaultsHorizontalAxes())
		axisStyle.WriteToRenderer(r)

		cursor := canvasBox.Top
		for _, bar := range sbc.Bars {
			if len(bar.Name) > 0 {
				barLabelBox := Box{
					Top:    cursor,
					Left:   0,
					Right:  canvasBox.Left + DefaultYAxisMargin,
					Bottom: cursor + bar.GetWidth() + sbc.GetBarSpacing(),
				}
				lines := Text.WrapFit(r, bar.Name, barLabelBox.Width(), axisStyle)
				linesBox := Text.MeasureLines(r, lines, axisStyle)

				yAxisWidth = MaxInt(linesBox.Height()+(2*DefaultXAxisMargin), yAxisWidth)
			}
		}
		return Box{
			Top:    canvasBox.Top,
			Left:   canvasBox.Left + yAxisWidth,
			Right:  canvasBox.Right,
			Bottom: canvasBox.Top + totalHeight,
		}
	}
	return Box{
		Top:    canvasBox.Top,
		Left:   canvasBox.Left,
		Right:  canvasBox.Right,
		Bottom: canvasBox.Top + totalHeight,
	}
}

// Box returns the chart bounds as a box.
func (sbc StackedBarChart) Box() Box {
	dpr := sbc.Background.Padding.GetRight(10)
	dpb := sbc.Background.Padding.GetBottom(50)

	return Box{
		Top:    sbc.Background.Padding.GetTop(20),
		Left:   sbc.Background.Padding.GetLeft(20),
		Right:  sbc.GetWidth() - dpr,
		Bottom: sbc.GetHeight() - dpb,
	}
}

func (sbc StackedBarChart) styleDefaultsStackedBarValue(index int) Style {
	return Style{
		StrokeColor: sbc.GetColorPalette().GetSeriesColor(index),
		StrokeWidth: 3.0,
		FillColor:   sbc.GetColorPalette().GetSeriesColor(index),
		FontSize:    sbc.getScaledFontSize(),
		FontColor:   sbc.GetColorPalette().TextColor(),
		Font:        sbc.GetFont(),
	}
}

func (sbc StackedBarChart) styleDefaultsTitle() Style {
	return sbc.TitleStyle.InheritFrom(Style{
		FontColor:           DefaultTextColor,
		Font:                sbc.GetFont(),
		FontSize:            sbc.getTitleFontSize(),
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignTop,
		TextWrap:            TextWrapWord,
	})
}

func (sbc StackedBarChart) getScaledFontSize() float64 {
	effectiveDimension := MinInt(sbc.GetWidth(), sbc.GetHeight())
	if effectiveDimension >= 2048 {
		return 48.0
	} else if effectiveDimension >= 1024 {
		return 24.0
	} else if effectiveDimension > 512 {
		return 18.0
	} else if effectiveDimension > 256 {
		return 12.0
	}
	return 10.0
}

func (sbc StackedBarChart) getTitleFontSize() float64 {
	effectiveDimension := MinInt(sbc.GetWidth(), sbc.GetHeight())
	if effectiveDimension >= 2048 {
		return 48
	} else if effectiveDimension >= 1024 {
		return 24
	} else if effectiveDimension >= 512 {
		return 18
	} else if effectiveDimension >= 256 {
		return 12
	}
	return 10
}

func (sbc StackedBarChart) styleDefaultsAxes() Style {
	return Style{
		StrokeColor:         DefaultAxisColor,
		Font:                sbc.GetFont(),
		FontSize:            DefaultAxisFontSize,
		FontColor:           DefaultAxisColor,
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignTop,
		TextWrap:            TextWrapWord,
	}
}

func (sbc StackedBarChart) styleDefaultsHorizontalAxes() Style {
	return Style{
		StrokeColor:         DefaultAxisColor,
		Font:                sbc.GetFont(),
		FontSize:            DefaultAxisFontSize,
		FontColor:           DefaultAxisColor,
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignMiddle,
		TextWrap:            TextWrapWord,
	}
}

func (sbc StackedBarChart) styleDefaultsElements() Style {
	return Style{
		Font: sbc.GetFont(),
	}
}
