package drawing

import (
	"fmt"
	"strconv"
)

var (
	// ColorTransparent is a fully transparent color.
	ColorTransparent = Color{}

	// ColorWhite is white.
	ColorWhite = Color{R: 255, G: 255, B: 255, A: 255}

	// ColorBlack is black.
	ColorBlack = Color{R: 0, G: 0, B: 0, A: 255}

	// ColorRed is red.
	ColorRed = Color{R: 255, G: 0, B: 0, A: 255}

	// ColorGreen is green.
	ColorGreen = Color{R: 0, G: 255, B: 0, A: 255}

	// ColorBlue is blue.
	ColorBlue = Color{R: 0, G: 0, B: 255, A: 255}
)

func parseHex(hex string) uint8 {
	v, _ := strconv.ParseInt(hex, 16, 16)
	return uint8(v)
}

// ColorFromHex returns a color from a css hex code.
func ColorFromHex(hex string) Color {
	var c Color
	if len(hex) == 3 {
		c.R = parseHex(string(hex[0])) * 0x11
		c.G = parseHex(string(hex[1])) * 0x11
		c.B = parseHex(string(hex[2])) * 0x11
	} else {
		c.R = parseHex(string(hex[0:2]))
		c.G = parseHex(string(hex[2:4]))
		c.B = parseHex(string(hex[4:6]))
	}
	c.A = 255
	return c
}

// ColorFromAlphaMixedRGBA returns the system alpha mixed rgba values.
func ColorFromAlphaMixedRGBA(r, g, b, a uint32) Color {
	fa := float64(a) / 255.0
	var c Color
	c.R = uint8(float64(r) / fa)
	c.G = uint8(float64(g) / fa)
	c.B = uint8(float64(b) / fa)
	c.A = uint8(a | (a >> 8))
	return c
}

// ColorChannelFromFloat returns a normalized byte from a given float value.
func ColorChannelFromFloat(v float64) uint8 {
	return uint8(v * 255)
}

// Color is our internal color type because color.Color is bullshit.
type Color struct {
	R, G, B, A uint8
}

// RGBA returns the color as a pre-alpha mixed color set.
func (c Color) RGBA() (r, g, b, a uint32) {
	fa := float64(c.A) / 255.0
	r = uint32(float64(uint32(c.R)) * fa)
	r |= r << 8
	g = uint32(float64(uint32(c.G)) * fa)
	g |= g << 8
	b = uint32(float64(uint32(c.B)) * fa)
	b |= b << 8
	a = uint32(c.A)
	a |= a << 8
	return
}

// IsZero returns if the color has been set or not.
func (c Color) IsZero() bool {
	return c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0
}

// IsTransparent returns if the colors alpha channel is zero.
func (c Color) IsTransparent() bool {
	return c.A == 0
}

// WithAlpha returns a copy of the color with a given alpha.
func (c Color) WithAlpha(a uint8) Color {
	return Color{
		R: c.R,
		G: c.G,
		B: c.B,
		A: a,
	}
}

// Equals returns true if the color equals another.
func (c Color) Equals(other Color) bool {
	return c.R == other.R &&
		c.G == other.G &&
		c.B == other.B &&
		c.A == other.A
}

// AverageWith averages two colors.
func (c Color) AverageWith(other Color) Color {
	return Color{
		R: (c.R + other.R) >> 1,
		G: (c.G + other.G) >> 1,
		B: (c.B + other.B) >> 1,
		A: c.A,
	}
}

// String returns a css string representation of the color.
func (c Color) String() string {
	fa := float64(c.A) / float64(255)
	return fmt.Sprintf("rgba(%v,%v,%v,%.1f)", c.R, c.G, c.B, fa)
}
