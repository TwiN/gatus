package drawing

// DemuxFlattener is a flattener
type DemuxFlattener struct {
	Flatteners []Flattener
}

// MoveTo implements the path builder interface.
func (dc DemuxFlattener) MoveTo(x, y float64) {
	for _, flattener := range dc.Flatteners {
		flattener.MoveTo(x, y)
	}
}

// LineTo implements the path builder interface.
func (dc DemuxFlattener) LineTo(x, y float64) {
	for _, flattener := range dc.Flatteners {
		flattener.LineTo(x, y)
	}
}

// LineJoin implements the path builder interface.
func (dc DemuxFlattener) LineJoin() {
	for _, flattener := range dc.Flatteners {
		flattener.LineJoin()
	}
}

// Close implements the path builder interface.
func (dc DemuxFlattener) Close() {
	for _, flattener := range dc.Flatteners {
		flattener.Close()
	}
}

// End implements the path builder interface.
func (dc DemuxFlattener) End() {
	for _, flattener := range dc.Flatteners {
		flattener.End()
	}
}
