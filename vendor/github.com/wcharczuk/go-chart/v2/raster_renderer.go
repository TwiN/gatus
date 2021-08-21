package chart

import (
	"image"
	"image/png"
	"io"
	"math"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// PNG returns a new png/raster renderer.
func PNG(width, height int) (Renderer, error) {
	i := image.NewRGBA(image.Rect(0, 0, width, height))
	gc, err := drawing.NewRasterGraphicContext(i)
	if err == nil {
		return &rasterRenderer{
			i:  i,
			gc: gc,
		}, nil
	}
	return nil, err
}

// rasterRenderer renders chart commands to a bitmap.
type rasterRenderer struct {
	i  *image.RGBA
	gc *drawing.RasterGraphicContext

	rotateRadians *float64

	s Style
}

func (rr *rasterRenderer) ResetStyle() {
	rr.s = Style{Font: rr.s.Font}
	rr.ClearTextRotation()
}

// GetDPI returns the dpi.
func (rr *rasterRenderer) GetDPI() float64 {
	return rr.gc.GetDPI()
}

// SetDPI implements the interface method.
func (rr *rasterRenderer) SetDPI(dpi float64) {
	rr.gc.SetDPI(dpi)
}

// SetClassName implements the interface method. However, PNGs have no classes.
func (rr *rasterRenderer) SetClassName(_ string) {}

// SetStrokeColor implements the interface method.
func (rr *rasterRenderer) SetStrokeColor(c drawing.Color) {
	rr.s.StrokeColor = c
}

// SetLineWidth implements the interface method.
func (rr *rasterRenderer) SetStrokeWidth(width float64) {
	rr.s.StrokeWidth = width
}

// StrokeDashArray sets the stroke dash array.
func (rr *rasterRenderer) SetStrokeDashArray(dashArray []float64) {
	rr.s.StrokeDashArray = dashArray
}

// SetFillColor implements the interface method.
func (rr *rasterRenderer) SetFillColor(c drawing.Color) {
	rr.s.FillColor = c
}

// MoveTo implements the interface method.
func (rr *rasterRenderer) MoveTo(x, y int) {
	rr.gc.MoveTo(float64(x), float64(y))
}

// LineTo implements the interface method.
func (rr *rasterRenderer) LineTo(x, y int) {
	rr.gc.LineTo(float64(x), float64(y))
}

// QuadCurveTo implements the interface method.
func (rr *rasterRenderer) QuadCurveTo(cx, cy, x, y int) {
	rr.gc.QuadCurveTo(float64(cx), float64(cy), float64(x), float64(y))
}

// ArcTo implements the interface method.
func (rr *rasterRenderer) ArcTo(cx, cy int, rx, ry, startAngle, delta float64) {
	rr.gc.ArcTo(float64(cx), float64(cy), rx, ry, startAngle, delta)
}

// Close implements the interface method.
func (rr *rasterRenderer) Close() {
	rr.gc.Close()
}

// Stroke implements the interface method.
func (rr *rasterRenderer) Stroke() {
	rr.gc.SetStrokeColor(rr.s.StrokeColor)
	rr.gc.SetLineWidth(rr.s.StrokeWidth)
	rr.gc.SetLineDash(rr.s.StrokeDashArray, 0)
	rr.gc.Stroke()
}

// Fill implements the interface method.
func (rr *rasterRenderer) Fill() {
	rr.gc.SetFillColor(rr.s.FillColor)
	rr.gc.Fill()
}

// FillStroke implements the interface method.
func (rr *rasterRenderer) FillStroke() {
	rr.gc.SetFillColor(rr.s.FillColor)
	rr.gc.SetStrokeColor(rr.s.StrokeColor)
	rr.gc.SetLineWidth(rr.s.StrokeWidth)
	rr.gc.SetLineDash(rr.s.StrokeDashArray, 0)
	rr.gc.FillStroke()
}

// Circle fully draws a circle at a given point but does not apply the fill or stroke.
func (rr *rasterRenderer) Circle(radius float64, x, y int) {
	xf := float64(x)
	yf := float64(y)

	rr.gc.MoveTo(xf-radius, yf)                            //9
	rr.gc.QuadCurveTo(xf-radius, yf-radius, xf, yf-radius) //12
	rr.gc.QuadCurveTo(xf+radius, yf-radius, xf+radius, yf) //3
	rr.gc.QuadCurveTo(xf+radius, yf+radius, xf, yf+radius) //6
	rr.gc.QuadCurveTo(xf-radius, yf+radius, xf-radius, yf) //9
}

// SetFont implements the interface method.
func (rr *rasterRenderer) SetFont(f *truetype.Font) {
	rr.s.Font = f
}

// SetFontSize implements the interface method.
func (rr *rasterRenderer) SetFontSize(size float64) {
	rr.s.FontSize = size
}

// SetFontColor implements the interface method.
func (rr *rasterRenderer) SetFontColor(c drawing.Color) {
	rr.s.FontColor = c
}

// Text implements the interface method.
func (rr *rasterRenderer) Text(body string, x, y int) {
	xf, yf := rr.getCoords(x, y)
	rr.gc.SetFont(rr.s.Font)
	rr.gc.SetFontSize(rr.s.FontSize)
	rr.gc.SetFillColor(rr.s.FontColor)
	rr.gc.CreateStringPath(body, float64(xf), float64(yf))
	rr.gc.Fill()
}

// MeasureText returns the height and width in pixels of a string.
func (rr *rasterRenderer) MeasureText(body string) Box {
	rr.gc.SetFont(rr.s.Font)
	rr.gc.SetFontSize(rr.s.FontSize)
	rr.gc.SetFillColor(rr.s.FontColor)
	l, t, r, b, err := rr.gc.GetStringBounds(body)
	if err != nil {
		return Box{}
	}
	if l < 0 {
		r = r - l // equivalent to r+(-1*l)
		l = 0
	}
	if t < 0 {
		b = b - t
		t = 0
	}

	if l > 0 {
		r = r + l
		l = 0
	}

	if t > 0 {
		b = b + t
		t = 0
	}

	textBox := Box{
		Top:    int(math.Ceil(t)),
		Left:   int(math.Ceil(l)),
		Right:  int(math.Ceil(r)),
		Bottom: int(math.Ceil(b)),
	}
	if rr.rotateRadians == nil {
		return textBox
	}

	return textBox.Corners().Rotate(RadiansToDegrees(*rr.rotateRadians)).Box()
}

// SetTextRotation sets a text rotation.
func (rr *rasterRenderer) SetTextRotation(radians float64) {
	rr.rotateRadians = &radians
}

func (rr *rasterRenderer) getCoords(x, y int) (xf, yf int) {
	if rr.rotateRadians == nil {
		xf = x
		yf = y
		return
	}

	rr.gc.Translate(float64(x), float64(y))
	rr.gc.Rotate(*rr.rotateRadians)
	return
}

// ClearTextRotation clears text rotation.
func (rr *rasterRenderer) ClearTextRotation() {
	rr.gc.SetMatrixTransform(drawing.NewIdentityMatrix())
	rr.rotateRadians = nil
}

// Save implements the interface method.
func (rr *rasterRenderer) Save(w io.Writer) error {
	if typed, isTyped := w.(RGBACollector); isTyped {
		typed.SetRGBA(rr.i)
		return nil
	}
	return png.Encode(w, rr.i)
}
