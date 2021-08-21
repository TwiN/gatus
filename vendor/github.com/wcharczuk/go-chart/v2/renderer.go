package chart

import (
	"io"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// Renderer represents the basic methods required to draw a chart.
type Renderer interface {
	// ResetStyle should reset any style related settings on the renderer.
	ResetStyle()

	// GetDPI gets the DPI for the renderer.
	GetDPI() float64

	// SetDPI sets the DPI for the renderer.
	SetDPI(dpi float64)

	// SetClassName sets the current class name.
	SetClassName(string)

	// SetStrokeColor sets the current stroke color.
	SetStrokeColor(drawing.Color)

	// SetFillColor sets the current fill color.
	SetFillColor(drawing.Color)

	// SetStrokeWidth sets the stroke width.
	SetStrokeWidth(width float64)

	// SetStrokeDashArray sets the stroke dash array.
	SetStrokeDashArray(dashArray []float64)

	// MoveTo moves the cursor to a given point.
	MoveTo(x, y int)

	// LineTo both starts a shape and draws a line to a given point
	// from the previous point.
	LineTo(x, y int)

	// QuadCurveTo draws a quad curve.
	// cx and cy represent the bezier "control points".
	QuadCurveTo(cx, cy, x, y int)

	// ArcTo draws an arc with a given center (cx,cy)
	// a given set of radii (rx,ry), a startAngle and delta (in radians).
	ArcTo(cx, cy int, rx, ry, startAngle, delta float64)

	// Close finalizes a shape as drawn by LineTo.
	Close()

	// Stroke strokes the path.
	Stroke()

	// Fill fills the path, but does not stroke.
	Fill()

	// FillStroke fills and strokes a path.
	FillStroke()

	// Circle draws a circle at the given coords with a given radius.
	Circle(radius float64, x, y int)

	// SetFont sets a font for a text field.
	SetFont(*truetype.Font)

	// SetFontColor sets a font's color
	SetFontColor(drawing.Color)

	// SetFontSize sets the font size for a text field.
	SetFontSize(size float64)

	// Text draws a text blob.
	Text(body string, x, y int)

	// MeasureText measures text.
	MeasureText(body string) Box

	// SetTextRotatation sets a rotation for drawing elements.
	SetTextRotation(radians float64)

	// ClearTextRotation clears rotation.
	ClearTextRotation()

	// Save writes the image to the given writer.
	Save(w io.Writer) error
}
