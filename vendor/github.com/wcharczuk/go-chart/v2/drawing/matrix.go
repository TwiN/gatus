package drawing

import (
	"math"
)

// Matrix represents an affine transformation
type Matrix [6]float64

const (
	epsilon = 1e-6
)

// Determinant compute the determinant of the matrix
func (tr Matrix) Determinant() float64 {
	return tr[0]*tr[3] - tr[1]*tr[2]
}

// Transform applies the transformation matrix to points. It modify the points passed in parameter.
func (tr Matrix) Transform(points []float64) {
	for i, j := 0, 1; j < len(points); i, j = i+2, j+2 {
		x := points[i]
		y := points[j]
		points[i] = x*tr[0] + y*tr[2] + tr[4]
		points[j] = x*tr[1] + y*tr[3] + tr[5]
	}
}

// TransformPoint applies the transformation matrix to point. It returns the point the transformed point.
func (tr Matrix) TransformPoint(x, y float64) (xres, yres float64) {
	xres = x*tr[0] + y*tr[2] + tr[4]
	yres = x*tr[1] + y*tr[3] + tr[5]
	return xres, yres
}

func minMax(x, y float64) (min, max float64) {
	if x > y {
		return y, x
	}
	return x, y
}

// TransformRectangle applies the transformation matrix to the rectangle represented by the min and the max point of the rectangle
func (tr Matrix) TransformRectangle(x0, y0, x2, y2 float64) (nx0, ny0, nx2, ny2 float64) {
	points := []float64{x0, y0, x2, y0, x2, y2, x0, y2}
	tr.Transform(points)
	points[0], points[2] = minMax(points[0], points[2])
	points[4], points[6] = minMax(points[4], points[6])
	points[1], points[3] = minMax(points[1], points[3])
	points[5], points[7] = minMax(points[5], points[7])

	nx0 = math.Min(points[0], points[4])
	ny0 = math.Min(points[1], points[5])
	nx2 = math.Max(points[2], points[6])
	ny2 = math.Max(points[3], points[7])
	return nx0, ny0, nx2, ny2
}

// InverseTransform applies the transformation inverse matrix to the rectangle represented by the min and the max point of the rectangle
func (tr Matrix) InverseTransform(points []float64) {
	d := tr.Determinant() // matrix determinant
	for i, j := 0, 1; j < len(points); i, j = i+2, j+2 {
		x := points[i]
		y := points[j]
		points[i] = ((x-tr[4])*tr[3] - (y-tr[5])*tr[2]) / d
		points[j] = ((y-tr[5])*tr[0] - (x-tr[4])*tr[1]) / d
	}
}

// InverseTransformPoint applies the transformation inverse matrix to point. It returns the point the transformed point.
func (tr Matrix) InverseTransformPoint(x, y float64) (xres, yres float64) {
	d := tr.Determinant() // matrix determinant
	xres = ((x-tr[4])*tr[3] - (y-tr[5])*tr[2]) / d
	yres = ((y-tr[5])*tr[0] - (x-tr[4])*tr[1]) / d
	return xres, yres
}

// VectorTransform applies the transformation matrix to points without using the translation parameter of the affine matrix.
// It modify the points passed in parameter.
func (tr Matrix) VectorTransform(points []float64) {
	for i, j := 0, 1; j < len(points); i, j = i+2, j+2 {
		x := points[i]
		y := points[j]
		points[i] = x*tr[0] + y*tr[2]
		points[j] = x*tr[1] + y*tr[3]
	}
}

// NewIdentityMatrix creates an identity transformation matrix.
func NewIdentityMatrix() Matrix {
	return Matrix{1, 0, 0, 1, 0, 0}
}

// NewTranslationMatrix creates a transformation matrix with a translation tx and ty translation parameter
func NewTranslationMatrix(tx, ty float64) Matrix {
	return Matrix{1, 0, 0, 1, tx, ty}
}

// NewScaleMatrix creates a transformation matrix with a sx, sy scale factor
func NewScaleMatrix(sx, sy float64) Matrix {
	return Matrix{sx, 0, 0, sy, 0, 0}
}

// NewRotationMatrix creates a rotation transformation matrix. angle is in radian
func NewRotationMatrix(angle float64) Matrix {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return Matrix{c, s, -s, c, 0, 0}
}

// NewMatrixFromRects creates a transformation matrix, combining a scale and a translation, that transform rectangle1 into rectangle2.
func NewMatrixFromRects(rectangle1, rectangle2 [4]float64) Matrix {
	xScale := (rectangle2[2] - rectangle2[0]) / (rectangle1[2] - rectangle1[0])
	yScale := (rectangle2[3] - rectangle2[1]) / (rectangle1[3] - rectangle1[1])
	xOffset := rectangle2[0] - (rectangle1[0] * xScale)
	yOffset := rectangle2[1] - (rectangle1[1] * yScale)
	return Matrix{xScale, 0, 0, yScale, xOffset, yOffset}
}

// Inverse computes the inverse matrix
func (tr *Matrix) Inverse() {
	d := tr.Determinant() // matrix determinant
	tr0, tr1, tr2, tr3, tr4, tr5 := tr[0], tr[1], tr[2], tr[3], tr[4], tr[5]
	tr[0] = tr3 / d
	tr[1] = -tr1 / d
	tr[2] = -tr2 / d
	tr[3] = tr0 / d
	tr[4] = (tr2*tr5 - tr3*tr4) / d
	tr[5] = (tr1*tr4 - tr0*tr5) / d
}

// Copy copies the matrix.
func (tr Matrix) Copy() Matrix {
	var result Matrix
	copy(result[:], tr[:])
	return result
}

// Compose multiplies trToConcat x tr
func (tr *Matrix) Compose(trToCompose Matrix) {
	tr0, tr1, tr2, tr3, tr4, tr5 := tr[0], tr[1], tr[2], tr[3], tr[4], tr[5]
	tr[0] = trToCompose[0]*tr0 + trToCompose[1]*tr2
	tr[1] = trToCompose[1]*tr3 + trToCompose[0]*tr1
	tr[2] = trToCompose[2]*tr0 + trToCompose[3]*tr2
	tr[3] = trToCompose[3]*tr3 + trToCompose[2]*tr1
	tr[4] = trToCompose[4]*tr0 + trToCompose[5]*tr2 + tr4
	tr[5] = trToCompose[5]*tr3 + trToCompose[4]*tr1 + tr5
}

// Scale adds a scale to the matrix
func (tr *Matrix) Scale(sx, sy float64) {
	tr[0] = sx * tr[0]
	tr[1] = sx * tr[1]
	tr[2] = sy * tr[2]
	tr[3] = sy * tr[3]
}

// Translate adds a translation to the matrix
func (tr *Matrix) Translate(tx, ty float64) {
	tr[4] = tx*tr[0] + ty*tr[2] + tr[4]
	tr[5] = ty*tr[3] + tx*tr[1] + tr[5]
}

// Rotate adds a rotation to the matrix.
func (tr *Matrix) Rotate(radians float64) {
	c := math.Cos(radians)
	s := math.Sin(radians)
	t0 := c*tr[0] + s*tr[2]
	t1 := s*tr[3] + c*tr[1]
	t2 := c*tr[2] - s*tr[0]
	t3 := c*tr[3] - s*tr[1]
	tr[0] = t0
	tr[1] = t1
	tr[2] = t2
	tr[3] = t3
}

// GetTranslation gets the matrix traslation.
func (tr Matrix) GetTranslation() (x, y float64) {
	return tr[4], tr[5]
}

// GetScaling gets the matrix scaling.
func (tr Matrix) GetScaling() (x, y float64) {
	return tr[0], tr[3]
}

// GetScale computes a scale for the matrix
func (tr Matrix) GetScale() float64 {
	x := 0.707106781*tr[0] + 0.707106781*tr[1]
	y := 0.707106781*tr[2] + 0.707106781*tr[3]
	return math.Sqrt(x*x + y*y)
}

// ******************** Testing ********************

// Equals tests if a two transformation are equal. A tolerance is applied when comparing matrix elements.
func (tr Matrix) Equals(tr2 Matrix) bool {
	for i := 0; i < 6; i = i + 1 {
		if !fequals(tr[i], tr2[i]) {
			return false
		}
	}
	return true
}

// IsIdentity tests if a transformation is the identity transformation. A tolerance is applied when comparing matrix elements.
func (tr Matrix) IsIdentity() bool {
	return fequals(tr[4], 0) && fequals(tr[5], 0) && tr.IsTranslation()
}

// IsTranslation tests if a transformation is is a pure translation. A tolerance is applied when comparing matrix elements.
func (tr Matrix) IsTranslation() bool {
	return fequals(tr[0], 1) && fequals(tr[1], 0) && fequals(tr[2], 0) && fequals(tr[3], 1)
}

// fequals compares two floats. return true if the distance between the two floats is less than epsilon, false otherwise
func fequals(float1, float2 float64) bool {
	return math.Abs(float1-float2) <= epsilon
}
