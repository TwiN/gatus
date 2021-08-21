package chart

// NameProvider is a type that returns a name.
type NameProvider interface {
	GetName() string
}

// StyleProvider is a type that returns a style.
type StyleProvider interface {
	GetStyle() Style
}

// IsZeroable is a type that returns if it's been set or not.
type IsZeroable interface {
	IsZero() bool
}

// Stringable is a type that has a string representation.
type Stringable interface {
	String() string
}

// Range is a common interface for a range of values.
type Range interface {
	Stringable
	IsZeroable

	GetMin() float64
	SetMin(min float64)

	GetMax() float64
	SetMax(max float64)

	GetDelta() float64

	GetDomain() int
	SetDomain(domain int)

	IsDescending() bool

	// Translate the range to the domain.
	Translate(value float64) int
}
