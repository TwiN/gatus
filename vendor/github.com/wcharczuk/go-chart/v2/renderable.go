package chart

// Renderable is a function that can be called to render custom elements on the chart.
type Renderable func(r Renderer, canvasBox Box, defaults Style)
