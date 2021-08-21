package chart

// ValueFormatterProvider is a series that has custom formatters.
type ValueFormatterProvider interface {
	GetValueFormatters() (x, y ValueFormatter)
}
