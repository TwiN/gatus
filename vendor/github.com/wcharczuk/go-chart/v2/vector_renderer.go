package chart

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"

	"golang.org/x/image/font"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// SVG returns a new png/raster renderer.
func SVG(width, height int) (Renderer, error) {
	buffer := bytes.NewBuffer([]byte{})
	canvas := newCanvas(buffer)
	canvas.Start(width, height)
	return &vectorRenderer{
		b:   buffer,
		c:   canvas,
		s:   &Style{},
		p:   []string{},
		dpi: DefaultDPI,
	}, nil
}

// SVGWithCSS returns a new png/raster renderer with attached custom CSS
// The optional nonce argument sets a CSP nonce.
func SVGWithCSS(css string, nonce string) func(width, height int) (Renderer, error) {
	return func(width, height int) (Renderer, error) {
		buffer := bytes.NewBuffer([]byte{})
		canvas := newCanvas(buffer)
		canvas.css = css
		canvas.nonce = nonce
		canvas.Start(width, height)
		return &vectorRenderer{
			b:   buffer,
			c:   canvas,
			s:   &Style{},
			p:   []string{},
			dpi: DefaultDPI,
		}, nil
	}
}

// vectorRenderer renders chart commands to a bitmap.
type vectorRenderer struct {
	dpi float64
	b   *bytes.Buffer
	c   *canvas
	s   *Style
	p   []string
	fc  *font.Drawer
}

func (vr *vectorRenderer) ResetStyle() {
	vr.s = &Style{Font: vr.s.Font}
	vr.fc = nil
}

// GetDPI returns the dpi.
func (vr *vectorRenderer) GetDPI() float64 {
	return vr.dpi
}

// SetDPI implements the interface method.
func (vr *vectorRenderer) SetDPI(dpi float64) {
	vr.dpi = dpi
	vr.c.dpi = dpi
}

// SetClassName implements the interface method.
func (vr *vectorRenderer) SetClassName(classname string) {
	vr.s.ClassName = classname
}

// SetStrokeColor implements the interface method.
func (vr *vectorRenderer) SetStrokeColor(c drawing.Color) {
	vr.s.StrokeColor = c
}

// SetFillColor implements the interface method.
func (vr *vectorRenderer) SetFillColor(c drawing.Color) {
	vr.s.FillColor = c
}

// SetLineWidth implements the interface method.
func (vr *vectorRenderer) SetStrokeWidth(width float64) {
	vr.s.StrokeWidth = width
}

// StrokeDashArray sets the stroke dash array.
func (vr *vectorRenderer) SetStrokeDashArray(dashArray []float64) {
	vr.s.StrokeDashArray = dashArray
}

// MoveTo implements the interface method.
func (vr *vectorRenderer) MoveTo(x, y int) {
	vr.p = append(vr.p, fmt.Sprintf("M %d %d", x, y))
}

// LineTo implements the interface method.
func (vr *vectorRenderer) LineTo(x, y int) {
	vr.p = append(vr.p, fmt.Sprintf("L %d %d", x, y))
}

// QuadCurveTo draws a quad curve.
func (vr *vectorRenderer) QuadCurveTo(cx, cy, x, y int) {
	vr.p = append(vr.p, fmt.Sprintf("Q%d,%d %d,%d", cx, cy, x, y))
}

func (vr *vectorRenderer) ArcTo(cx, cy int, rx, ry, startAngle, delta float64) {
	startAngle = RadianAdd(startAngle, _pi2)
	endAngle := RadianAdd(startAngle, delta)

	startx := cx + int(rx*math.Sin(startAngle))
	starty := cy - int(ry*math.Cos(startAngle))

	if len(vr.p) > 0 {
		vr.p = append(vr.p, fmt.Sprintf("L %d %d", startx, starty))
	} else {
		vr.p = append(vr.p, fmt.Sprintf("M %d %d", startx, starty))
	}

	endx := cx + int(rx*math.Sin(endAngle))
	endy := cy - int(ry*math.Cos(endAngle))

	dd := RadiansToDegrees(delta)

	largeArcFlag := 0
	if delta > _pi {
		largeArcFlag = 1
	}

	vr.p = append(vr.p, fmt.Sprintf("A %d %d %0.2f %d 1 %d %d", int(rx), int(ry), dd, largeArcFlag, endx, endy))
}

// Close closes a shape.
func (vr *vectorRenderer) Close() {
	vr.p = append(vr.p, fmt.Sprintf("Z"))
}

// Stroke draws the path with no fill.
func (vr *vectorRenderer) Stroke() {
	vr.drawPath(vr.s.GetStrokeOptions())
}

// Fill draws the path with no stroke.
func (vr *vectorRenderer) Fill() {
	vr.drawPath(vr.s.GetFillOptions())
}

// FillStroke draws the path with both fill and stroke.
func (vr *vectorRenderer) FillStroke() {
	vr.drawPath(vr.s.GetFillAndStrokeOptions())
}

// drawPath draws a path.
func (vr *vectorRenderer) drawPath(s Style) {
	vr.c.Path(strings.Join(vr.p, "\n"), vr.s.GetFillAndStrokeOptions())
	vr.p = []string{} // clear the path
}

// Circle implements the interface method.
func (vr *vectorRenderer) Circle(radius float64, x, y int) {
	vr.c.Circle(x, y, int(radius), vr.s.GetFillAndStrokeOptions())
}

// SetFont implements the interface method.
func (vr *vectorRenderer) SetFont(f *truetype.Font) {
	vr.s.Font = f
}

// SetFontColor implements the interface method.
func (vr *vectorRenderer) SetFontColor(c drawing.Color) {
	vr.s.FontColor = c
}

// SetFontSize implements the interface method.
func (vr *vectorRenderer) SetFontSize(size float64) {
	vr.s.FontSize = size
}

// Text draws a text blob.
func (vr *vectorRenderer) Text(body string, x, y int) {
	vr.c.Text(x, y, body, vr.s.GetTextOptions())
}

// MeasureText uses the truetype font drawer to measure the width of text.
func (vr *vectorRenderer) MeasureText(body string) (box Box) {
	if vr.s.GetFont() != nil {
		vr.fc = &font.Drawer{
			Face: truetype.NewFace(vr.s.GetFont(), &truetype.Options{
				DPI:  vr.dpi,
				Size: vr.s.FontSize,
			}),
		}
		w := vr.fc.MeasureString(body).Ceil()

		box.Right = w
		box.Bottom = int(drawing.PointsToPixels(vr.dpi, vr.s.FontSize))
		if vr.c.textTheta == nil {
			return
		}
		box = box.Corners().Rotate(RadiansToDegrees(*vr.c.textTheta)).Box()
	}
	return
}

// SetTextRotation sets the text rotation.
func (vr *vectorRenderer) SetTextRotation(radians float64) {
	vr.c.textTheta = &radians
}

// ClearTextRotation clears the text rotation.
func (vr *vectorRenderer) ClearTextRotation() {
	vr.c.textTheta = nil
}

// Save saves the renderer's contents to a writer.
func (vr *vectorRenderer) Save(w io.Writer) error {
	vr.c.End()
	_, err := w.Write(vr.b.Bytes())
	return err
}

func newCanvas(w io.Writer) *canvas {
	return &canvas{
		w:   w,
		dpi: DefaultDPI,
	}
}

type canvas struct {
	w         io.Writer
	dpi       float64
	textTheta *float64
	width     int
	height    int
	css       string
	nonce     string
}

func (c *canvas) Start(width, height int) {
	c.width = width
	c.height = height
	c.w.Write([]byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d">\n`, c.width, c.height)))
	if c.css != "" {
		c.w.Write([]byte(`<style type="text/css"`))
		if c.nonce != "" {
			// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy
			c.w.Write([]byte(fmt.Sprintf(` nonce="%s"`, c.nonce)))
		}
		// To avoid compatibility issues between XML and CSS (f.e. with child selectors) we should encapsulate the CSS with CDATA.
		c.w.Write([]byte(fmt.Sprintf(`><![CDATA[%s]]></style>`, c.css)))
	}
}

func (c *canvas) Path(d string, style Style) {
	var strokeDashArrayProperty string
	if len(style.StrokeDashArray) > 0 {
		strokeDashArrayProperty = c.getStrokeDashArray(style)
	}
	c.w.Write([]byte(fmt.Sprintf(`<path %s d="%s" %s/>`, strokeDashArrayProperty, d, c.styleAsSVG(style))))
}

func (c *canvas) Text(x, y int, body string, style Style) {
	if c.textTheta == nil {
		c.w.Write([]byte(fmt.Sprintf(`<text x="%d" y="%d" %s>%s</text>`, x, y, c.styleAsSVG(style), body)))
	} else {
		transform := fmt.Sprintf(` transform="rotate(%0.2f,%d,%d)"`, RadiansToDegrees(*c.textTheta), x, y)
		c.w.Write([]byte(fmt.Sprintf(`<text x="%d" y="%d" %s%s>%s</text>`, x, y, c.styleAsSVG(style), transform, body)))
	}
}

func (c *canvas) Circle(x, y, r int, style Style) {
	c.w.Write([]byte(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" %s/>`, x, y, r, c.styleAsSVG(style))))
}

func (c *canvas) End() {
	c.w.Write([]byte("</svg>"))
}

// getStrokeDashArray returns the stroke-dasharray property of a style.
func (c *canvas) getStrokeDashArray(s Style) string {
	if len(s.StrokeDashArray) > 0 {
		var values []string
		for _, v := range s.StrokeDashArray {
			values = append(values, fmt.Sprintf("%0.1f", v))
		}
		return "stroke-dasharray=\"" + strings.Join(values, ", ") + "\""
	}
	return ""
}

// GetFontFace returns the font face for the style.
func (c *canvas) getFontFace(s Style) string {
	family := "sans-serif"
	if s.GetFont() != nil {
		name := s.GetFont().Name(truetype.NameIDFontFamily)
		if len(name) != 0 {
			family = fmt.Sprintf(`'%s',%s`, name, family)
		}
	}
	return fmt.Sprintf("font-family:%s", family)
}

// styleAsSVG returns the style as a svg style or class string.
func (c *canvas) styleAsSVG(s Style) string {
	sw := s.StrokeWidth
	sc := s.StrokeColor
	fc := s.FillColor
	fs := s.FontSize
	fnc := s.FontColor

	if s.ClassName != "" {
		var classes []string
		classes = append(classes, s.ClassName)
		if !sc.IsZero() {
			classes = append(classes, "stroke")
		}
		if !fc.IsZero() {
			classes = append(classes, "fill")
		}
		if fs != 0 || s.Font != nil {
			classes = append(classes, "text")
		}

		return fmt.Sprintf("class=\"%s\"", strings.Join(classes, " "))
	}

	var pieces []string

	if sw != 0 {
		pieces = append(pieces, "stroke-width:"+fmt.Sprintf("%d", int(sw)))
	} else {
		pieces = append(pieces, "stroke-width:0")
	}

	if !sc.IsZero() {
		pieces = append(pieces, "stroke:"+sc.String())
	} else {
		pieces = append(pieces, "stroke:none")
	}

	if !fnc.IsZero() {
		pieces = append(pieces, "fill:"+fnc.String())
	} else if !fc.IsZero() {
		pieces = append(pieces, "fill:"+fc.String())
	} else {
		pieces = append(pieces, "fill:none")
	}

	if fs != 0 {
		pieces = append(pieces, "font-size:"+fmt.Sprintf("%.1fpx", drawing.PointsToPixels(c.dpi, fs)))
	}

	if s.Font != nil {
		pieces = append(pieces, c.getFontFace(s))
	}
	return fmt.Sprintf("style=\"%s\"", strings.Join(pieces, ";"))
}
