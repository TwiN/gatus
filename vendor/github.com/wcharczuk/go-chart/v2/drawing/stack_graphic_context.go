package drawing

import (
	"image"
	"image/color"

	"github.com/golang/freetype/truetype"
)

// StackGraphicContext is a context that does thngs.
type StackGraphicContext struct {
	current *ContextStack
}

// ContextStack is a graphic context implementation.
type ContextStack struct {
	Tr          Matrix
	Path        *Path
	LineWidth   float64
	Dash        []float64
	DashOffset  float64
	StrokeColor color.Color
	FillColor   color.Color
	FillRule    FillRule
	Cap         LineCap
	Join        LineJoin

	FontSizePoints float64
	Font           *truetype.Font

	Scale float64

	Previous *ContextStack
}

// NewStackGraphicContext Create a new Graphic context from an image
func NewStackGraphicContext() *StackGraphicContext {
	gc := &StackGraphicContext{}
	gc.current = new(ContextStack)
	gc.current.Tr = NewIdentityMatrix()
	gc.current.Path = new(Path)
	gc.current.LineWidth = 1.0
	gc.current.StrokeColor = image.Black
	gc.current.FillColor = image.White
	gc.current.Cap = RoundCap
	gc.current.FillRule = FillRuleEvenOdd
	gc.current.Join = RoundJoin
	gc.current.FontSizePoints = 10
	return gc
}

// GetMatrixTransform returns the matrix transform.
func (gc *StackGraphicContext) GetMatrixTransform() Matrix {
	return gc.current.Tr
}

// SetMatrixTransform sets the matrix transform.
func (gc *StackGraphicContext) SetMatrixTransform(tr Matrix) {
	gc.current.Tr = tr
}

// ComposeMatrixTransform composes a transform into the current transform.
func (gc *StackGraphicContext) ComposeMatrixTransform(tr Matrix) {
	gc.current.Tr.Compose(tr)
}

// Rotate rotates the matrix transform by an angle in degrees.
func (gc *StackGraphicContext) Rotate(angle float64) {
	gc.current.Tr.Rotate(angle)
}

// Translate translates a transform.
func (gc *StackGraphicContext) Translate(tx, ty float64) {
	gc.current.Tr.Translate(tx, ty)
}

// Scale scales a transform.
func (gc *StackGraphicContext) Scale(sx, sy float64) {
	gc.current.Tr.Scale(sx, sy)
}

// SetStrokeColor sets the stroke color.
func (gc *StackGraphicContext) SetStrokeColor(c color.Color) {
	gc.current.StrokeColor = c
}

// SetFillColor sets the fill color.
func (gc *StackGraphicContext) SetFillColor(c color.Color) {
	gc.current.FillColor = c
}

// SetFillRule sets the fill rule.
func (gc *StackGraphicContext) SetFillRule(f FillRule) {
	gc.current.FillRule = f
}

// SetLineWidth sets the line width.
func (gc *StackGraphicContext) SetLineWidth(lineWidth float64) {
	gc.current.LineWidth = lineWidth
}

// SetLineCap sets the line cap.
func (gc *StackGraphicContext) SetLineCap(cap LineCap) {
	gc.current.Cap = cap
}

// SetLineJoin sets the line join.
func (gc *StackGraphicContext) SetLineJoin(join LineJoin) {
	gc.current.Join = join
}

// SetLineDash sets the line dash.
func (gc *StackGraphicContext) SetLineDash(dash []float64, dashOffset float64) {
	gc.current.Dash = dash
	gc.current.DashOffset = dashOffset
}

// SetFontSize sets the font size.
func (gc *StackGraphicContext) SetFontSize(fontSizePoints float64) {
	gc.current.FontSizePoints = fontSizePoints
}

// GetFontSize gets the font size.
func (gc *StackGraphicContext) GetFontSize() float64 {
	return gc.current.FontSizePoints
}

// SetFont sets the current font.
func (gc *StackGraphicContext) SetFont(f *truetype.Font) {
	gc.current.Font = f
}

// GetFont returns the font.
func (gc *StackGraphicContext) GetFont() *truetype.Font {
	return gc.current.Font
}

// BeginPath starts a new path.
func (gc *StackGraphicContext) BeginPath() {
	gc.current.Path.Clear()
}

// IsEmpty returns if the path is empty.
func (gc *StackGraphicContext) IsEmpty() bool {
	return gc.current.Path.IsEmpty()
}

// LastPoint returns the last point on the path.
func (gc *StackGraphicContext) LastPoint() (x float64, y float64) {
	return gc.current.Path.LastPoint()
}

// MoveTo moves the cursor for a path.
func (gc *StackGraphicContext) MoveTo(x, y float64) {
	gc.current.Path.MoveTo(x, y)
}

// LineTo draws a line.
func (gc *StackGraphicContext) LineTo(x, y float64) {
	gc.current.Path.LineTo(x, y)
}

// QuadCurveTo draws a quad curve.
func (gc *StackGraphicContext) QuadCurveTo(cx, cy, x, y float64) {
	gc.current.Path.QuadCurveTo(cx, cy, x, y)
}

// CubicCurveTo draws a cubic curve.
func (gc *StackGraphicContext) CubicCurveTo(cx1, cy1, cx2, cy2, x, y float64) {
	gc.current.Path.CubicCurveTo(cx1, cy1, cx2, cy2, x, y)
}

// ArcTo draws an arc.
func (gc *StackGraphicContext) ArcTo(cx, cy, rx, ry, startAngle, delta float64) {
	gc.current.Path.ArcTo(cx, cy, rx, ry, startAngle, delta)
}

// Close closes a path.
func (gc *StackGraphicContext) Close() {
	gc.current.Path.Close()
}

// Save pushes a context onto the stack.
func (gc *StackGraphicContext) Save() {
	context := new(ContextStack)
	context.FontSizePoints = gc.current.FontSizePoints
	context.Font = gc.current.Font
	context.LineWidth = gc.current.LineWidth
	context.StrokeColor = gc.current.StrokeColor
	context.FillColor = gc.current.FillColor
	context.FillRule = gc.current.FillRule
	context.Dash = gc.current.Dash
	context.DashOffset = gc.current.DashOffset
	context.Cap = gc.current.Cap
	context.Join = gc.current.Join
	context.Path = gc.current.Path.Copy()
	context.Font = gc.current.Font
	context.Scale = gc.current.Scale
	copy(context.Tr[:], gc.current.Tr[:])
	context.Previous = gc.current
	gc.current = context
}

// Restore restores the previous context.
func (gc *StackGraphicContext) Restore() {
	if gc.current.Previous != nil {
		oldContext := gc.current
		gc.current = gc.current.Previous
		oldContext.Previous = nil
	}
}
