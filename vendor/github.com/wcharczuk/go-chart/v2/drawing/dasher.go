package drawing

// NewDashVertexConverter creates a new dash converter.
func NewDashVertexConverter(dash []float64, dashOffset float64, flattener Flattener) *DashVertexConverter {
	var dasher DashVertexConverter
	dasher.dash = dash
	dasher.currentDash = 0
	dasher.dashOffset = dashOffset
	dasher.next = flattener
	return &dasher
}

// DashVertexConverter is a converter for dash vertexes.
type DashVertexConverter struct {
	next           Flattener
	x, y, distance float64
	dash           []float64
	currentDash    int
	dashOffset     float64
}

// LineTo implements the pathbuilder interface.
func (dasher *DashVertexConverter) LineTo(x, y float64) {
	dasher.lineTo(x, y)
}

// MoveTo implements the pathbuilder interface.
func (dasher *DashVertexConverter) MoveTo(x, y float64) {
	dasher.next.MoveTo(x, y)
	dasher.x, dasher.y = x, y
	dasher.distance = dasher.dashOffset
	dasher.currentDash = 0
}

// LineJoin implements the pathbuilder interface.
func (dasher *DashVertexConverter) LineJoin() {
	dasher.next.LineJoin()
}

// Close implements the pathbuilder interface.
func (dasher *DashVertexConverter) Close() {
	dasher.next.Close()
}

// End implements the pathbuilder interface.
func (dasher *DashVertexConverter) End() {
	dasher.next.End()
}

func (dasher *DashVertexConverter) lineTo(x, y float64) {
	rest := dasher.dash[dasher.currentDash] - dasher.distance
	for rest < 0 {
		dasher.distance = dasher.distance - dasher.dash[dasher.currentDash]
		dasher.currentDash = (dasher.currentDash + 1) % len(dasher.dash)
		rest = dasher.dash[dasher.currentDash] - dasher.distance
	}
	d := distance(dasher.x, dasher.y, x, y)
	for d >= rest {
		k := rest / d
		lx := dasher.x + k*(x-dasher.x)
		ly := dasher.y + k*(y-dasher.y)
		if dasher.currentDash%2 == 0 {
			// line
			dasher.next.LineTo(lx, ly)
		} else {
			// gap
			dasher.next.End()
			dasher.next.MoveTo(lx, ly)
		}
		d = d - rest
		dasher.x, dasher.y = lx, ly
		dasher.currentDash = (dasher.currentDash + 1) % len(dasher.dash)
		rest = dasher.dash[dasher.currentDash]
	}
	dasher.distance = d
	if dasher.currentDash%2 == 0 {
		// line
		dasher.next.LineTo(x, y)
	} else {
		// gap
		dasher.next.End()
		dasher.next.MoveTo(x, y)
	}
	if dasher.distance >= dasher.dash[dasher.currentDash] {
		dasher.distance = dasher.distance - dasher.dash[dasher.currentDash]
		dasher.currentDash = (dasher.currentDash + 1) % len(dasher.dash)
	}
	dasher.x, dasher.y = x, y
}
