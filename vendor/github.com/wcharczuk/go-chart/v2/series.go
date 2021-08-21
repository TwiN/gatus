package chart

// Series is an alias to Renderable.
type Series interface {
	GetName() string
	GetYAxis() YAxisType
	GetStyle() Style
	Validate() error
	Render(r Renderer, canvasBox Box, xrange, yrange Range, s Style)
}
