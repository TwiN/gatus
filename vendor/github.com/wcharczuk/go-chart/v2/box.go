package chart

import (
	"fmt"
	"math"
)

var (
	// BoxZero is a preset box that represents an intentional zero value.
	BoxZero = Box{IsSet: true}
)

// NewBox returns a new (set) box.
func NewBox(top, left, right, bottom int) Box {
	return Box{
		IsSet:  true,
		Top:    top,
		Left:   left,
		Right:  right,
		Bottom: bottom,
	}
}

// Box represents the main 4 dimensions of a box.
type Box struct {
	Top    int
	Left   int
	Right  int
	Bottom int
	IsSet  bool
}

// IsZero returns if the box is set or not.
func (b Box) IsZero() bool {
	if b.IsSet {
		return false
	}
	return b.Top == 0 && b.Left == 0 && b.Right == 0 && b.Bottom == 0
}

// String returns a string representation of the box.
func (b Box) String() string {
	return fmt.Sprintf("box(%d,%d,%d,%d)", b.Top, b.Left, b.Right, b.Bottom)
}

// GetTop returns a coalesced value with a default.
func (b Box) GetTop(defaults ...int) int {
	if !b.IsSet && b.Top == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return 0
	}
	return b.Top
}

// GetLeft returns a coalesced value with a default.
func (b Box) GetLeft(defaults ...int) int {
	if !b.IsSet && b.Left == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return 0
	}
	return b.Left
}

// GetRight returns a coalesced value with a default.
func (b Box) GetRight(defaults ...int) int {
	if !b.IsSet && b.Right == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return 0
	}
	return b.Right
}

// GetBottom returns a coalesced value with a default.
func (b Box) GetBottom(defaults ...int) int {
	if !b.IsSet && b.Bottom == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return 0
	}
	return b.Bottom
}

// Width returns the width
func (b Box) Width() int {
	return AbsInt(b.Right - b.Left)
}

// Height returns the height
func (b Box) Height() int {
	return AbsInt(b.Bottom - b.Top)
}

// Center returns the center of the box
func (b Box) Center() (x, y int) {
	w2, h2 := b.Width()>>1, b.Height()>>1
	return b.Left + w2, b.Top + h2
}

// Aspect returns the aspect ratio of the box.
func (b Box) Aspect() float64 {
	return float64(b.Width()) / float64(b.Height())
}

// Clone returns a new copy of the box.
func (b Box) Clone() Box {
	return Box{
		IsSet:  b.IsSet,
		Top:    b.Top,
		Left:   b.Left,
		Right:  b.Right,
		Bottom: b.Bottom,
	}
}

// IsBiggerThan returns if a box is bigger than another box.
func (b Box) IsBiggerThan(other Box) bool {
	return b.Top < other.Top ||
		b.Bottom > other.Bottom ||
		b.Left < other.Left ||
		b.Right > other.Right
}

// IsSmallerThan returns if a box is smaller than another box.
func (b Box) IsSmallerThan(other Box) bool {
	return b.Top > other.Top &&
		b.Bottom < other.Bottom &&
		b.Left > other.Left &&
		b.Right < other.Right
}

// Equals returns if the box equals another box.
func (b Box) Equals(other Box) bool {
	return b.Top == other.Top &&
		b.Left == other.Left &&
		b.Right == other.Right &&
		b.Bottom == other.Bottom
}

// Grow grows a box based on another box.
func (b Box) Grow(other Box) Box {
	return Box{
		Top:    MinInt(b.Top, other.Top),
		Left:   MinInt(b.Left, other.Left),
		Right:  MaxInt(b.Right, other.Right),
		Bottom: MaxInt(b.Bottom, other.Bottom),
	}
}

// Shift pushes a box by x,y.
func (b Box) Shift(x, y int) Box {
	return Box{
		Top:    b.Top + y,
		Left:   b.Left + x,
		Right:  b.Right + x,
		Bottom: b.Bottom + y,
	}
}

// Corners returns the box as a set of corners.
func (b Box) Corners() BoxCorners {
	return BoxCorners{
		TopLeft:     Point{b.Left, b.Top},
		TopRight:    Point{b.Right, b.Top},
		BottomRight: Point{b.Right, b.Bottom},
		BottomLeft:  Point{b.Left, b.Bottom},
	}
}

// Fit is functionally the inverse of grow.
// Fit maintains the original aspect ratio of the `other` box,
// but constrains it to the bounds of the target box.
func (b Box) Fit(other Box) Box {
	ba := b.Aspect()
	oa := other.Aspect()

	if oa == ba {
		return b.Clone()
	}

	bw, bh := float64(b.Width()), float64(b.Height())
	bw2 := int(bw) >> 1
	bh2 := int(bh) >> 1
	if oa > ba { // ex. 16:9 vs. 4:3
		var noh2 int
		if oa > 1.0 {
			noh2 = int(bw/oa) >> 1
		} else {
			noh2 = int(bh*oa) >> 1
		}
		return Box{
			Top:    (b.Top + bh2) - noh2,
			Left:   b.Left,
			Right:  b.Right,
			Bottom: (b.Top + bh2) + noh2,
		}
	}
	var now2 int
	if oa > 1.0 {
		now2 = int(bh/oa) >> 1
	} else {
		now2 = int(bw*oa) >> 1
	}
	return Box{
		Top:    b.Top,
		Left:   (b.Left + bw2) - now2,
		Right:  (b.Left + bw2) + now2,
		Bottom: b.Bottom,
	}
}

// Constrain is similar to `Fit` except that it will work
// more literally like the opposite of grow.
func (b Box) Constrain(other Box) Box {
	newBox := b.Clone()

	newBox.Top = MaxInt(newBox.Top, other.Top)
	newBox.Left = MaxInt(newBox.Left, other.Left)
	newBox.Right = MinInt(newBox.Right, other.Right)
	newBox.Bottom = MinInt(newBox.Bottom, other.Bottom)

	return newBox
}

// OuterConstrain is similar to `Constraint` with the difference
// that it applies corrections
func (b Box) OuterConstrain(bounds, other Box) Box {
	newBox := b.Clone()
	if other.Top < bounds.Top {
		delta := bounds.Top - other.Top
		newBox.Top = b.Top + delta
	}

	if other.Left < bounds.Left {
		delta := bounds.Left - other.Left
		newBox.Left = b.Left + delta
	}

	if other.Right > bounds.Right {
		delta := other.Right - bounds.Right
		newBox.Right = b.Right - delta
	}

	if other.Bottom > bounds.Bottom {
		delta := other.Bottom - bounds.Bottom
		newBox.Bottom = b.Bottom - delta
	}
	return newBox
}

// BoxCorners is a box with independent corners.
type BoxCorners struct {
	TopLeft, TopRight, BottomRight, BottomLeft Point
}

// Box return the BoxCorners as a regular box.
func (bc BoxCorners) Box() Box {
	return Box{
		Top:    MinInt(bc.TopLeft.Y, bc.TopRight.Y),
		Left:   MinInt(bc.TopLeft.X, bc.BottomLeft.X),
		Right:  MaxInt(bc.TopRight.X, bc.BottomRight.X),
		Bottom: MaxInt(bc.BottomLeft.Y, bc.BottomRight.Y),
	}
}

// Width returns the width
func (bc BoxCorners) Width() int {
	minLeft := MinInt(bc.TopLeft.X, bc.BottomLeft.X)
	maxRight := MaxInt(bc.TopRight.X, bc.BottomRight.X)
	return maxRight - minLeft
}

// Height returns the height
func (bc BoxCorners) Height() int {
	minTop := MinInt(bc.TopLeft.Y, bc.TopRight.Y)
	maxBottom := MaxInt(bc.BottomLeft.Y, bc.BottomRight.Y)
	return maxBottom - minTop
}

// Center returns the center of the box
func (bc BoxCorners) Center() (x, y int) {

	left := MeanInt(bc.TopLeft.X, bc.BottomLeft.X)
	right := MeanInt(bc.TopRight.X, bc.BottomRight.X)
	x = ((right - left) >> 1) + left

	top := MeanInt(bc.TopLeft.Y, bc.TopRight.Y)
	bottom := MeanInt(bc.BottomLeft.Y, bc.BottomRight.Y)
	y = ((bottom - top) >> 1) + top

	return
}

// Rotate rotates the box.
func (bc BoxCorners) Rotate(thetaDegrees float64) BoxCorners {
	cx, cy := bc.Center()

	thetaRadians := DegreesToRadians(thetaDegrees)

	tlx, tly := RotateCoordinate(cx, cy, bc.TopLeft.X, bc.TopLeft.Y, thetaRadians)
	trx, try := RotateCoordinate(cx, cy, bc.TopRight.X, bc.TopRight.Y, thetaRadians)
	brx, bry := RotateCoordinate(cx, cy, bc.BottomRight.X, bc.BottomRight.Y, thetaRadians)
	blx, bly := RotateCoordinate(cx, cy, bc.BottomLeft.X, bc.BottomLeft.Y, thetaRadians)

	return BoxCorners{
		TopLeft:     Point{tlx, tly},
		TopRight:    Point{trx, try},
		BottomRight: Point{brx, bry},
		BottomLeft:  Point{blx, bly},
	}
}

// Equals returns if the box equals another box.
func (bc BoxCorners) Equals(other BoxCorners) bool {
	return bc.TopLeft.Equals(other.TopLeft) &&
		bc.TopRight.Equals(other.TopRight) &&
		bc.BottomRight.Equals(other.BottomRight) &&
		bc.BottomLeft.Equals(other.BottomLeft)
}

func (bc BoxCorners) String() string {
	return fmt.Sprintf("BoxC{%s,%s,%s,%s}", bc.TopLeft.String(), bc.TopRight.String(), bc.BottomRight.String(), bc.BottomLeft.String())
}

// Point is an X,Y pair
type Point struct {
	X, Y int
}

// DistanceTo calculates the distance to another point.
func (p Point) DistanceTo(other Point) float64 {
	dx := math.Pow(float64(p.X-other.X), 2)
	dy := math.Pow(float64(p.Y-other.Y), 2)
	return math.Pow(dx+dy, 0.5)
}

// Equals returns if a point equals another point.
func (p Point) Equals(other Point) bool {
	return p.X == other.X && p.Y == other.Y
}

// String returns a string representation of the point.
func (p Point) String() string {
	return fmt.Sprintf("P{%d,%d}", p.X, p.Y)
}
