package chart

import "github.com/wcharczuk/go-chart/v2/drawing"

// Jet is a color map provider based on matlab's jet color map.
func Jet(v, vmin, vmax float64) drawing.Color {
	c := drawing.Color{R: 0xff, G: 0xff, B: 0xff, A: 0xff} // white
	var dv float64

	if v < vmin {
		v = vmin
	}
	if v > vmax {
		v = vmax
	}
	dv = vmax - vmin

	if v < (vmin + 0.25*dv) {
		c.R = 0
		c.G = drawing.ColorChannelFromFloat(4 * (v - vmin) / dv)
	} else if v < (vmin + 0.5*dv) {
		c.R = 0
		c.B = drawing.ColorChannelFromFloat(1 + 4*(vmin+0.25*dv-v)/dv)
	} else if v < (vmin + 0.75*dv) {
		c.R = drawing.ColorChannelFromFloat(4 * (v - vmin - 0.5*dv) / dv)
		c.B = 0
	} else {
		c.G = drawing.ColorChannelFromFloat(1 + 4*(vmin+0.75*dv-v)/dv)
		c.B = 0
	}

	return c
}
