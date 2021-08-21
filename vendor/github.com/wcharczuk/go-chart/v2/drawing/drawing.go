package drawing

import (
	"image/color"

	"github.com/golang/freetype/truetype"
)

// FillRule defines the type for fill rules
type FillRule int

const (
	// FillRuleEvenOdd determines the "insideness" of a point in the shape
	// by drawing a ray from that point to infinity in any direction
	// and counting the number of path segments from the given shape that the ray crosses.
	// If this number is odd, the point is inside; if even, the point is outside.
	FillRuleEvenOdd FillRule = iota
	// FillRuleWinding determines the "insideness" of a point in the shape
	// by drawing a ray from that point to infinity in any direction
	// and then examining the places where a segment of the shape crosses the ray.
	// Starting with a count of zero, add one each time a path segment crosses
	// the ray from left to right and subtract one each time
	// a path segment crosses the ray from right to left. After counting the crossings,
	// if the result is zero then the point is outside the path. Otherwise, it is inside.
	FillRuleWinding
)

// LineCap is the style of line extremities
type LineCap int

const (
	// RoundCap defines a rounded shape at the end of the line
	RoundCap LineCap = iota
	// ButtCap defines a squared shape exactly at the end of the line
	ButtCap
	// SquareCap defines a squared shape at the end of the line
	SquareCap
)

// LineJoin is the style of segments joint
type LineJoin int

const (
	// BevelJoin represents cut segments joint
	BevelJoin LineJoin = iota
	// RoundJoin represents rounded segments joint
	RoundJoin
	// MiterJoin represents peaker segments joint
	MiterJoin
)

// StrokeStyle keeps stroke style attributes
// that is used by the Stroke method of a Drawer
type StrokeStyle struct {
	// Color defines the color of stroke
	Color color.Color
	// Line width
	Width float64
	// Line cap style rounded, butt or square
	LineCap LineCap
	// Line join style bevel, round or miter
	LineJoin LineJoin
	// offset of the first dash
	DashOffset float64
	// array represented dash length pair values are plain dash and impair are space between dash
	// if empty display plain line
	Dash []float64
}

// SolidFillStyle define style attributes for a solid fill style
type SolidFillStyle struct {
	// Color defines the line color
	Color color.Color
	// FillRule defines the file rule to used
	FillRule FillRule
}

// Valign Vertical Alignment of the text
type Valign int

const (
	// ValignTop top align text
	ValignTop Valign = iota
	// ValignCenter centered text
	ValignCenter
	// ValignBottom bottom aligned text
	ValignBottom
	// ValignBaseline align text with the baseline of the font
	ValignBaseline
)

// Halign Horizontal Alignment of the text
type Halign int

const (
	// HalignLeft Horizontally align to left
	HalignLeft = iota
	// HalignCenter Horizontally align to center
	HalignCenter
	// HalignRight Horizontally align to right
	HalignRight
)

// TextStyle describe text property
type TextStyle struct {
	// Color defines the color of text
	Color color.Color
	// Size font size
	Size float64
	// The font to use
	Font *truetype.Font
	// Horizontal Alignment of the text
	Halign Halign
	// Vertical Alignment of the text
	Valign Valign
}

// ScalingPolicy is a constant to define how to scale an image
type ScalingPolicy int

const (
	// ScalingNone no scaling applied
	ScalingNone ScalingPolicy = iota
	// ScalingStretch the image is stretched so that its width and height are exactly the given width and height
	ScalingStretch
	// ScalingWidth the image is scaled so that its width is exactly the given width
	ScalingWidth
	// ScalingHeight the image is scaled so that its height is exactly the given height
	ScalingHeight
	// ScalingFit the image is scaled to the largest scale that allow the image to fit within a rectangle width x height
	ScalingFit
	// ScalingSameArea the image is scaled so that its area is exactly the area of the given rectangle width x height
	ScalingSameArea
	// ScalingFill the image is scaled to the smallest scale that allow the image to fully cover a rectangle width x height
	ScalingFill
)

// ImageScaling style attributes used to display the image
type ImageScaling struct {
	// Horizontal Alignment of the image
	Halign Halign
	// Vertical Alignment of the image
	Valign Valign
	// Width Height used by scaling policy
	Width, Height float64
	// ScalingPolicy defines the scaling policy to applied to the image
	ScalingPolicy ScalingPolicy
}
