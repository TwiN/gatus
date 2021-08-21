package chart

import "github.com/wcharczuk/go-chart/v2/drawing"

var (
	// ColorWhite is white.
	ColorWhite = drawing.Color{R: 255, G: 255, B: 255, A: 255}
	// ColorBlue is the basic theme blue color.
	ColorBlue = drawing.Color{R: 0, G: 116, B: 217, A: 255}
	// ColorCyan is the basic theme cyan color.
	ColorCyan = drawing.Color{R: 0, G: 217, B: 210, A: 255}
	// ColorGreen is the basic theme green color.
	ColorGreen = drawing.Color{R: 0, G: 217, B: 101, A: 255}
	// ColorRed is the basic theme red color.
	ColorRed = drawing.Color{R: 217, G: 0, B: 116, A: 255}
	// ColorOrange is the basic theme orange color.
	ColorOrange = drawing.Color{R: 217, G: 101, B: 0, A: 255}
	// ColorYellow is the basic theme yellow color.
	ColorYellow = drawing.Color{R: 217, G: 210, B: 0, A: 255}
	// ColorBlack is the basic theme black color.
	ColorBlack = drawing.Color{R: 51, G: 51, B: 51, A: 255}
	// ColorLightGray is the basic theme light gray color.
	ColorLightGray = drawing.Color{R: 239, G: 239, B: 239, A: 255}

	// ColorAlternateBlue is a alternate theme color.
	ColorAlternateBlue = drawing.Color{R: 106, G: 195, B: 203, A: 255}
	// ColorAlternateGreen is a alternate theme color.
	ColorAlternateGreen = drawing.Color{R: 42, G: 190, B: 137, A: 255}
	// ColorAlternateGray is a alternate theme color.
	ColorAlternateGray = drawing.Color{R: 110, G: 128, B: 139, A: 255}
	// ColorAlternateYellow is a alternate theme color.
	ColorAlternateYellow = drawing.Color{R: 240, G: 174, B: 90, A: 255}
	// ColorAlternateLightGray is a alternate theme color.
	ColorAlternateLightGray = drawing.Color{R: 187, G: 190, B: 191, A: 255}

	// ColorTransparent is a transparent (alpha zero) color.
	ColorTransparent = drawing.Color{R: 1, G: 1, B: 1, A: 0}
)

var (
	// DefaultBackgroundColor is the default chart background color.
	// It is equivalent to css color:white.
	DefaultBackgroundColor = ColorWhite
	// DefaultBackgroundStrokeColor is the default chart border color.
	// It is equivalent to color:white.
	DefaultBackgroundStrokeColor = ColorWhite
	// DefaultCanvasColor is the default chart canvas color.
	// It is equivalent to css color:white.
	DefaultCanvasColor = ColorWhite
	// DefaultCanvasStrokeColor is the default chart canvas stroke color.
	// It is equivalent to css color:white.
	DefaultCanvasStrokeColor = ColorWhite
	// DefaultTextColor is the default chart text color.
	// It is equivalent to #333333.
	DefaultTextColor = ColorBlack
	// DefaultAxisColor is the default chart axis line color.
	// It is equivalent to #333333.
	DefaultAxisColor = ColorBlack
	// DefaultStrokeColor is the default chart border color.
	// It is equivalent to #efefef.
	DefaultStrokeColor = ColorLightGray
	// DefaultFillColor is the default fill color.
	// It is equivalent to #0074d9.
	DefaultFillColor = ColorBlue
	// DefaultAnnotationFillColor is the default annotation background color.
	DefaultAnnotationFillColor = ColorWhite
	// DefaultGridLineColor is the default grid line color.
	DefaultGridLineColor = ColorLightGray
)

var (
	// DefaultColors are a couple default series colors.
	DefaultColors = []drawing.Color{
		ColorBlue,
		ColorGreen,
		ColorRed,
		ColorCyan,
		ColorOrange,
	}

	// DefaultAlternateColors are a couple alternate colors.
	DefaultAlternateColors = []drawing.Color{
		ColorAlternateBlue,
		ColorAlternateGreen,
		ColorAlternateGray,
		ColorAlternateYellow,
		ColorBlue,
		ColorGreen,
		ColorRed,
		ColorCyan,
		ColorOrange,
	}
)

// GetDefaultColor returns a color from the default list by index.
// NOTE: the index will wrap around (using a modulo).
func GetDefaultColor(index int) drawing.Color {
	finalIndex := index % len(DefaultColors)
	return DefaultColors[finalIndex]
}

// GetAlternateColor returns a color from the default list by index.
// NOTE: the index will wrap around (using a modulo).
func GetAlternateColor(index int) drawing.Color {
	finalIndex := index % len(DefaultAlternateColors)
	return DefaultAlternateColors[finalIndex]
}

// ColorPalette is a set of colors that.
type ColorPalette interface {
	BackgroundColor() drawing.Color
	BackgroundStrokeColor() drawing.Color
	CanvasColor() drawing.Color
	CanvasStrokeColor() drawing.Color
	AxisStrokeColor() drawing.Color
	TextColor() drawing.Color
	GetSeriesColor(index int) drawing.Color
}

// DefaultColorPalette represents the default palatte.
var DefaultColorPalette defaultColorPalette

type defaultColorPalette struct{}

func (dp defaultColorPalette) BackgroundColor() drawing.Color {
	return DefaultBackgroundColor
}

func (dp defaultColorPalette) BackgroundStrokeColor() drawing.Color {
	return DefaultBackgroundStrokeColor
}

func (dp defaultColorPalette) CanvasColor() drawing.Color {
	return DefaultCanvasColor
}

func (dp defaultColorPalette) CanvasStrokeColor() drawing.Color {
	return DefaultCanvasStrokeColor
}

func (dp defaultColorPalette) AxisStrokeColor() drawing.Color {
	return DefaultAxisColor
}

func (dp defaultColorPalette) TextColor() drawing.Color {
	return DefaultTextColor
}

func (dp defaultColorPalette) GetSeriesColor(index int) drawing.Color {
	return GetDefaultColor(index)
}

// AlternateColorPalette represents the default palatte.
var AlternateColorPalette alternateColorPalette

type alternateColorPalette struct{}

func (ap alternateColorPalette) BackgroundColor() drawing.Color {
	return DefaultBackgroundColor
}

func (ap alternateColorPalette) BackgroundStrokeColor() drawing.Color {
	return DefaultBackgroundStrokeColor
}

func (ap alternateColorPalette) CanvasColor() drawing.Color {
	return DefaultCanvasColor
}

func (ap alternateColorPalette) CanvasStrokeColor() drawing.Color {
	return DefaultCanvasStrokeColor
}

func (ap alternateColorPalette) AxisStrokeColor() drawing.Color {
	return DefaultAxisColor
}

func (ap alternateColorPalette) TextColor() drawing.Color {
	return DefaultTextColor
}

func (ap alternateColorPalette) GetSeriesColor(index int) drawing.Color {
	return GetAlternateColor(index)
}
