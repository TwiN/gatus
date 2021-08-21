package drawing

import (
	"github.com/golang/freetype/raster"
	"golang.org/x/image/math/fixed"
)

// FtLineBuilder is a builder for freetype raster glyphs.
type FtLineBuilder struct {
	Adder raster.Adder
}

// MoveTo implements the path builder interface.
func (liner FtLineBuilder) MoveTo(x, y float64) {
	liner.Adder.Start(fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)})
}

// LineTo implements the path builder interface.
func (liner FtLineBuilder) LineTo(x, y float64) {
	liner.Adder.Add1(fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)})
}

// LineJoin implements the path builder interface.
func (liner FtLineBuilder) LineJoin() {}

// Close implements the path builder interface.
func (liner FtLineBuilder) Close() {}

// End implements the path builder interface.
func (liner FtLineBuilder) End() {}
