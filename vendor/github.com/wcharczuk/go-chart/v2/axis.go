package chart

// TickPosition is an enumeration of possible tick drawing positions.
type TickPosition int

const (
	// TickPositionUnset means to use the default tick position.
	TickPositionUnset TickPosition = 0
	// TickPositionBetweenTicks draws the labels for a tick between the previous and current tick.
	TickPositionBetweenTicks TickPosition = 1
	// TickPositionUnderTick draws the tick below the tick.
	TickPositionUnderTick TickPosition = 2
)

// YAxisType is a type of y-axis; it can either be primary or secondary.
type YAxisType int

const (
	// YAxisPrimary is the primary axis.
	YAxisPrimary YAxisType = 0
	// YAxisSecondary is the secondary axis.
	YAxisSecondary YAxisType = 1
)

// Axis is a chart feature detailing what values happen where.
type Axis interface {
	GetName() string
	SetName(name string)

	GetStyle() Style
	SetStyle(style Style)

	GetTicks() []Tick
	GenerateTicks(r Renderer, ra Range, vf ValueFormatter) []Tick

	// GenerateGridLines returns the gridlines for the axis.
	GetGridLines(ticks []Tick) []GridLine

	// Measure should return an absolute box for the axis.
	// This is used when auto-fitting the canvas to the background.
	Measure(r Renderer, canvasBox Box, ra Range, style Style, ticks []Tick) Box

	// Render renders the axis.
	Render(r Renderer, canvasBox Box, ra Range, style Style, ticks []Tick)
}
