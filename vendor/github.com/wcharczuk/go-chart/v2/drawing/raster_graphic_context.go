package drawing

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/golang/freetype/raster"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// NewRasterGraphicContext creates a new Graphic context from an image.
func NewRasterGraphicContext(img draw.Image) (*RasterGraphicContext, error) {
	var painter Painter
	switch selectImage := img.(type) {
	case *image.RGBA:
		painter = raster.NewRGBAPainter(selectImage)
	default:
		return nil, errors.New("NewRasterGraphicContext() :: invalid image type")
	}
	return NewRasterGraphicContextWithPainter(img, painter), nil
}

// NewRasterGraphicContextWithPainter creates a new Graphic context from an image and a Painter (see Freetype-go)
func NewRasterGraphicContextWithPainter(img draw.Image, painter Painter) *RasterGraphicContext {
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	return &RasterGraphicContext{
		NewStackGraphicContext(),
		img,
		painter,
		raster.NewRasterizer(width, height),
		raster.NewRasterizer(width, height),
		&truetype.GlyphBuf{},
		DefaultDPI,
	}
}

// RasterGraphicContext is the implementation of GraphicContext for a raster image
type RasterGraphicContext struct {
	*StackGraphicContext
	img              draw.Image
	painter          Painter
	fillRasterizer   *raster.Rasterizer
	strokeRasterizer *raster.Rasterizer
	glyphBuf         *truetype.GlyphBuf
	DPI              float64
}

// SetDPI sets the screen resolution in dots per inch.
func (rgc *RasterGraphicContext) SetDPI(dpi float64) {
	rgc.DPI = dpi
	rgc.recalc()
}

// GetDPI returns the resolution of the Image GraphicContext
func (rgc *RasterGraphicContext) GetDPI() float64 {
	return rgc.DPI
}

// Clear fills the current canvas with a default transparent color
func (rgc *RasterGraphicContext) Clear() {
	width, height := rgc.img.Bounds().Dx(), rgc.img.Bounds().Dy()
	rgc.ClearRect(0, 0, width, height)
}

// ClearRect fills the current canvas with a default transparent color at the specified rectangle
func (rgc *RasterGraphicContext) ClearRect(x1, y1, x2, y2 int) {
	imageColor := image.NewUniform(rgc.current.FillColor)
	draw.Draw(rgc.img, image.Rect(x1, y1, x2, y2), imageColor, image.ZP, draw.Over)
}

// DrawImage draws the raster image in the current canvas
func (rgc *RasterGraphicContext) DrawImage(img image.Image) {
	DrawImage(img, rgc.img, rgc.current.Tr, draw.Over, BilinearFilter)
}

// FillString draws the text at point (0, 0)
func (rgc *RasterGraphicContext) FillString(text string) (cursor float64, err error) {
	cursor, err = rgc.FillStringAt(text, 0, 0)
	return
}

// FillStringAt draws the text at the specified point (x, y)
func (rgc *RasterGraphicContext) FillStringAt(text string, x, y float64) (cursor float64, err error) {
	cursor, err = rgc.CreateStringPath(text, x, y)
	rgc.Fill()
	return
}

// StrokeString draws the contour of the text at point (0, 0)
func (rgc *RasterGraphicContext) StrokeString(text string) (cursor float64, err error) {
	cursor, err = rgc.StrokeStringAt(text, 0, 0)
	return
}

// StrokeStringAt draws the contour of the text at point (x, y)
func (rgc *RasterGraphicContext) StrokeStringAt(text string, x, y float64) (cursor float64, err error) {
	cursor, err = rgc.CreateStringPath(text, x, y)
	rgc.Stroke()
	return
}

func (rgc *RasterGraphicContext) drawGlyph(glyph truetype.Index, dx, dy float64) error {
	if err := rgc.glyphBuf.Load(rgc.current.Font, fixed.Int26_6(rgc.current.Scale), glyph, font.HintingNone); err != nil {
		return err
	}
	e0 := 0
	for _, e1 := range rgc.glyphBuf.Ends {
		DrawContour(rgc, rgc.glyphBuf.Points[e0:e1], dx, dy)
		e0 = e1
	}
	return nil
}

// CreateStringPath creates a path from the string s at x, y, and returns the string width.
// The text is placed so that the left edge of the em square of the first character of s
// and the baseline intersect at x, y. The majority of the affected pixels will be
// above and to the right of the point, but some may be below or to the left.
// For example, drawing a string that starts with a 'J' in an italic font may
// affect pixels below and left of the point.
func (rgc *RasterGraphicContext) CreateStringPath(s string, x, y float64) (cursor float64, err error) {
	f := rgc.GetFont()
	if f == nil {
		err = errors.New("No font loaded, cannot continue")
		return
	}
	rgc.recalc()

	startx := x
	prev, hasPrev := truetype.Index(0), false
	for _, rc := range s {
		index := f.Index(rc)
		if hasPrev {
			x += fUnitsToFloat64(f.Kern(fixed.Int26_6(rgc.current.Scale), prev, index))
		}
		err = rgc.drawGlyph(index, x, y)
		if err != nil {
			cursor = x - startx
			return
		}
		x += fUnitsToFloat64(f.HMetric(fixed.Int26_6(rgc.current.Scale), index).AdvanceWidth)
		prev, hasPrev = index, true
	}
	cursor = x - startx
	return
}

// GetStringBounds returns the approximate pixel bounds of a string.
func (rgc *RasterGraphicContext) GetStringBounds(s string) (left, top, right, bottom float64, err error) {
	f := rgc.GetFont()
	if f == nil {
		err = errors.New("No font loaded, cannot continue")
		return
	}
	rgc.recalc()

	left = math.MaxFloat64
	top = math.MaxFloat64

	cursor := 0.0
	prev, hasPrev := truetype.Index(0), false
	for _, rc := range s {
		index := f.Index(rc)
		if hasPrev {
			cursor += fUnitsToFloat64(f.Kern(fixed.Int26_6(rgc.current.Scale), prev, index))
		}

		if err = rgc.glyphBuf.Load(rgc.current.Font, fixed.Int26_6(rgc.current.Scale), index, font.HintingNone); err != nil {
			return
		}
		e0 := 0
		for _, e1 := range rgc.glyphBuf.Ends {
			ps := rgc.glyphBuf.Points[e0:e1]
			for _, p := range ps {
				x, y := pointToF64Point(p)
				top = math.Min(top, y)
				bottom = math.Max(bottom, y)
				left = math.Min(left, x+cursor)
				right = math.Max(right, x+cursor)
			}
			e0 = e1
		}
		cursor += fUnitsToFloat64(f.HMetric(fixed.Int26_6(rgc.current.Scale), index).AdvanceWidth)
		prev, hasPrev = index, true
	}
	return
}

// recalc recalculates scale and bounds values from the font size, screen
// resolution and font metrics, and invalidates the glyph cache.
func (rgc *RasterGraphicContext) recalc() {
	rgc.current.Scale = rgc.current.FontSizePoints * float64(rgc.DPI)
}

// SetFont sets the font used to draw text.
func (rgc *RasterGraphicContext) SetFont(font *truetype.Font) {
	rgc.current.Font = font
}

// GetFont returns the font used to draw text.
func (rgc *RasterGraphicContext) GetFont() *truetype.Font {
	return rgc.current.Font
}

// SetFontSize sets the font size in points (as in ``a 12 point font'').
func (rgc *RasterGraphicContext) SetFontSize(fontSizePoints float64) {
	rgc.current.FontSizePoints = fontSizePoints
	rgc.recalc()
}

func (rgc *RasterGraphicContext) paint(rasterizer *raster.Rasterizer, color color.Color) {
	rgc.painter.SetColor(color)
	rasterizer.Rasterize(rgc.painter)
	rasterizer.Clear()
	rgc.current.Path.Clear()
}

// Stroke strokes the paths with the color specified by SetStrokeColor
func (rgc *RasterGraphicContext) Stroke(paths ...*Path) {
	paths = append(paths, rgc.current.Path)
	rgc.strokeRasterizer.UseNonZeroWinding = true

	stroker := NewLineStroker(rgc.current.Cap, rgc.current.Join, Transformer{Tr: rgc.current.Tr, Flattener: FtLineBuilder{Adder: rgc.strokeRasterizer}})
	stroker.HalfLineWidth = rgc.current.LineWidth / 2

	var liner Flattener
	if rgc.current.Dash != nil && len(rgc.current.Dash) > 0 {
		liner = NewDashVertexConverter(rgc.current.Dash, rgc.current.DashOffset, stroker)
	} else {
		liner = stroker
	}
	for _, p := range paths {
		Flatten(p, liner, rgc.current.Tr.GetScale())
	}

	rgc.paint(rgc.strokeRasterizer, rgc.current.StrokeColor)
}

// Fill fills the paths with the color specified by SetFillColor
func (rgc *RasterGraphicContext) Fill(paths ...*Path) {
	paths = append(paths, rgc.current.Path)
	rgc.fillRasterizer.UseNonZeroWinding = rgc.current.FillRule == FillRuleWinding

	flattener := Transformer{Tr: rgc.current.Tr, Flattener: FtLineBuilder{Adder: rgc.fillRasterizer}}
	for _, p := range paths {
		Flatten(p, flattener, rgc.current.Tr.GetScale())
	}

	rgc.paint(rgc.fillRasterizer, rgc.current.FillColor)
}

// FillStroke first fills the paths and than strokes them
func (rgc *RasterGraphicContext) FillStroke(paths ...*Path) {
	paths = append(paths, rgc.current.Path)
	rgc.fillRasterizer.UseNonZeroWinding = rgc.current.FillRule == FillRuleWinding
	rgc.strokeRasterizer.UseNonZeroWinding = true

	flattener := Transformer{Tr: rgc.current.Tr, Flattener: FtLineBuilder{Adder: rgc.fillRasterizer}}

	stroker := NewLineStroker(rgc.current.Cap, rgc.current.Join, Transformer{Tr: rgc.current.Tr, Flattener: FtLineBuilder{Adder: rgc.strokeRasterizer}})
	stroker.HalfLineWidth = rgc.current.LineWidth / 2

	var liner Flattener
	if rgc.current.Dash != nil && len(rgc.current.Dash) > 0 {
		liner = NewDashVertexConverter(rgc.current.Dash, rgc.current.DashOffset, stroker)
	} else {
		liner = stroker
	}

	demux := DemuxFlattener{Flatteners: []Flattener{flattener, liner}}
	for _, p := range paths {
		Flatten(p, demux, rgc.current.Tr.GetScale())
	}

	// Fill
	rgc.paint(rgc.fillRasterizer, rgc.current.FillColor)
	// Stroke
	rgc.paint(rgc.strokeRasterizer, rgc.current.StrokeColor)
}
