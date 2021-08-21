package chart

import "github.com/wcharczuk/go-chart/v2/drawing"

// ValuesProvider is a type that produces values.
type ValuesProvider interface {
	Len() int
	GetValues(index int) (float64, float64)
}

// BoundedValuesProvider allows series to return a range.
type BoundedValuesProvider interface {
	Len() int
	GetBoundedValues(index int) (x, y1, y2 float64)
}

// FirstValuesProvider is a special type of value provider that can return it's (potentially computed) first value.
type FirstValuesProvider interface {
	GetFirstValues() (x, y float64)
}

// LastValuesProvider is a special type of value provider that can return it's (potentially computed) last value.
type LastValuesProvider interface {
	GetLastValues() (x, y float64)
}

// BoundedLastValuesProvider is a special type of value provider that can return it's (potentially computed) bounded last value.
type BoundedLastValuesProvider interface {
	GetBoundedLastValues() (x, y1, y2 float64)
}

// FullValuesProvider is an interface that combines `ValuesProvider` and `LastValuesProvider`
type FullValuesProvider interface {
	ValuesProvider
	LastValuesProvider
}

// FullBoundedValuesProvider is an interface that combines `BoundedValuesProvider` and `BoundedLastValuesProvider`
type FullBoundedValuesProvider interface {
	BoundedValuesProvider
	BoundedLastValuesProvider
}

// SizeProvider is a provider for integer size.
type SizeProvider func(xrange, yrange Range, index int, x, y float64) float64

// ColorProvider is a general provider for color ranges based on values.
type ColorProvider func(v, vmin, vmax float64) drawing.Color

// DotColorProvider is a provider for dot color.
type DotColorProvider func(xrange, yrange Range, index int, x, y float64) drawing.Color
