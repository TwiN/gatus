package chart

import (
	"errors"
	"fmt"
	"io"

	"github.com/golang/freetype/truetype"
)

// PieChart is a chart that draws sections of a circle based on percentages.
type PieChart struct {
	Title      string
	TitleStyle Style

	ColorPalette ColorPalette

	Width  int
	Height int
	DPI    float64

	Background Style
	Canvas     Style
	SliceStyle Style

	Font        *truetype.Font
	defaultFont *truetype.Font

	Values   []Value
	Elements []Renderable
}

// GetDPI returns the dpi for the chart.
func (pc PieChart) GetDPI(defaults ...float64) float64 {
	if pc.DPI == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultDPI
	}
	return pc.DPI
}

// GetFont returns the text font.
func (pc PieChart) GetFont() *truetype.Font {
	if pc.Font == nil {
		return pc.defaultFont
	}
	return pc.Font
}

// GetWidth returns the chart width or the default value.
func (pc PieChart) GetWidth() int {
	if pc.Width == 0 {
		return DefaultChartWidth
	}
	return pc.Width
}

// GetHeight returns the chart height or the default value.
func (pc PieChart) GetHeight() int {
	if pc.Height == 0 {
		return DefaultChartWidth
	}
	return pc.Height
}

// Render renders the chart with the given renderer to the given io.Writer.
func (pc PieChart) Render(rp RendererProvider, w io.Writer) error {
	if len(pc.Values) == 0 {
		return errors.New("please provide at least one value")
	}

	r, err := rp(pc.GetWidth(), pc.GetHeight())
	if err != nil {
		return err
	}

	if pc.Font == nil {
		defaultFont, err := GetDefaultFont()
		if err != nil {
			return err
		}
		pc.defaultFont = defaultFont
	}
	r.SetDPI(pc.GetDPI(DefaultDPI))

	canvasBox := pc.getDefaultCanvasBox()
	canvasBox = pc.getCircleAdjustedCanvasBox(canvasBox)

	pc.drawBackground(r)
	pc.drawCanvas(r, canvasBox)

	finalValues, err := pc.finalizeValues(pc.Values)
	if err != nil {
		return err
	}
	pc.drawSlices(r, canvasBox, finalValues)
	pc.drawTitle(r)
	for _, a := range pc.Elements {
		a(r, canvasBox, pc.styleDefaultsElements())
	}

	return r.Save(w)
}

func (pc PieChart) drawBackground(r Renderer) {
	Draw.Box(r, Box{
		Right:  pc.GetWidth(),
		Bottom: pc.GetHeight(),
	}, pc.getBackgroundStyle())
}

func (pc PieChart) drawCanvas(r Renderer, canvasBox Box) {
	Draw.Box(r, canvasBox, pc.getCanvasStyle())
}

func (pc PieChart) drawTitle(r Renderer) {
	if len(pc.Title) > 0 && !pc.TitleStyle.Hidden {
		Draw.TextWithin(r, pc.Title, pc.Box(), pc.styleDefaultsTitle())
	}
}

func (pc PieChart) drawSlices(r Renderer, canvasBox Box, values []Value) {
	cx, cy := canvasBox.Center()
	diameter := MinInt(canvasBox.Width(), canvasBox.Height())
	radius := float64(diameter >> 1)
	labelRadius := (radius * 2.0) / 3.0

	// draw the pie slices
	var rads, delta, delta2, total float64
	var lx, ly int

	if len(values) == 1 {
		pc.stylePieChartValue(0).WriteToRenderer(r)
		r.MoveTo(cx, cy)
		r.Circle(radius, cx, cy)
	} else {
		for index, v := range values {
			v.Style.InheritFrom(pc.stylePieChartValue(index)).WriteToRenderer(r)

			r.MoveTo(cx, cy)
			rads = PercentToRadians(total)
			delta = PercentToRadians(v.Value)

			r.ArcTo(cx, cy, radius, radius, rads, delta)

			r.LineTo(cx, cy)
			r.Close()
			r.FillStroke()
			total = total + v.Value
		}
	}

	// draw the labels
	total = 0
	for index, v := range values {
		v.Style.InheritFrom(pc.stylePieChartValue(index)).WriteToRenderer(r)
		if len(v.Label) > 0 {
			delta2 = PercentToRadians(total + (v.Value / 2.0))
			delta2 = RadianAdd(delta2, _pi2)
			lx, ly = CirclePoint(cx, cy, labelRadius, delta2)

			tb := r.MeasureText(v.Label)
			lx = lx - (tb.Width() >> 1)
			ly = ly + (tb.Height() >> 1)

			if lx < 0 {
				lx = 0
			}
			if ly < 0 {
				lx = 0
			}

			r.Text(v.Label, lx, ly)
		}
		total = total + v.Value
	}
}

func (pc PieChart) finalizeValues(values []Value) ([]Value, error) {
	finalValues := Values(values).Normalize()
	if len(finalValues) == 0 {
		return nil, fmt.Errorf("pie chart must contain at least (1) non-zero value")
	}
	return finalValues, nil
}

func (pc PieChart) getDefaultCanvasBox() Box {
	return pc.Box()
}

func (pc PieChart) getCircleAdjustedCanvasBox(canvasBox Box) Box {
	circleDiameter := MinInt(canvasBox.Width(), canvasBox.Height())

	square := Box{
		Right:  circleDiameter,
		Bottom: circleDiameter,
	}

	return canvasBox.Fit(square)
}

func (pc PieChart) getBackgroundStyle() Style {
	return pc.Background.InheritFrom(pc.styleDefaultsBackground())
}

func (pc PieChart) getCanvasStyle() Style {
	return pc.Canvas.InheritFrom(pc.styleDefaultsCanvas())
}

func (pc PieChart) styleDefaultsCanvas() Style {
	return Style{
		FillColor:   pc.GetColorPalette().CanvasColor(),
		StrokeColor: pc.GetColorPalette().CanvasStrokeColor(),
		StrokeWidth: DefaultStrokeWidth,
	}
}

func (pc PieChart) styleDefaultsPieChartValue() Style {
	return Style{
		StrokeColor: pc.GetColorPalette().TextColor(),
		StrokeWidth: 5.0,
		FillColor:   pc.GetColorPalette().TextColor(),
	}
}

func (pc PieChart) stylePieChartValue(index int) Style {
	return pc.SliceStyle.InheritFrom(Style{
		StrokeColor: ColorWhite,
		StrokeWidth: 5.0,
		FillColor:   pc.GetColorPalette().GetSeriesColor(index),
		FontSize:    pc.getScaledFontSize(),
		FontColor:   pc.GetColorPalette().TextColor(),
		Font:        pc.GetFont(),
	})
}

func (pc PieChart) getScaledFontSize() float64 {
	effectiveDimension := MinInt(pc.GetWidth(), pc.GetHeight())
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

func (pc PieChart) styleDefaultsBackground() Style {
	return Style{
		FillColor:   pc.GetColorPalette().BackgroundColor(),
		StrokeColor: pc.GetColorPalette().BackgroundStrokeColor(),
		StrokeWidth: DefaultStrokeWidth,
	}
}

func (pc PieChart) styleDefaultsElements() Style {
	return Style{
		Font: pc.GetFont(),
	}
}

func (pc PieChart) styleDefaultsTitle() Style {
	return pc.TitleStyle.InheritFrom(Style{
		FontColor:           pc.GetColorPalette().TextColor(),
		Font:                pc.GetFont(),
		FontSize:            pc.getTitleFontSize(),
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignTop,
		TextWrap:            TextWrapWord,
	})
}

func (pc PieChart) getTitleFontSize() float64 {
	effectiveDimension := MinInt(pc.GetWidth(), pc.GetHeight())
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

// GetColorPalette returns the color palette for the chart.
func (pc PieChart) GetColorPalette() ColorPalette {
	if pc.ColorPalette != nil {
		return pc.ColorPalette
	}
	return AlternateColorPalette
}

// Box returns the chart bounds as a box.
func (pc PieChart) Box() Box {
	dpr := pc.Background.Padding.GetRight(DefaultBackgroundPadding.Right)
	dpb := pc.Background.Padding.GetBottom(DefaultBackgroundPadding.Bottom)

	return Box{
		Top:    pc.Background.Padding.GetTop(DefaultBackgroundPadding.Top),
		Left:   pc.Background.Padding.GetLeft(DefaultBackgroundPadding.Left),
		Right:  pc.GetWidth() - dpr,
		Bottom: pc.GetHeight() - dpb,
	}
}
