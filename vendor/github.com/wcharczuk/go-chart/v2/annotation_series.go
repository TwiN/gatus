package chart

import (
	"fmt"
	"math"
)

// Interface Assertions.
var (
	_ Series = (*AnnotationSeries)(nil)
)

// AnnotationSeries is a series of labels on the chart.
type AnnotationSeries struct {
	Name        string
	Style       Style
	YAxis       YAxisType
	Annotations []Value2
}

// GetName returns the name of the time series.
func (as AnnotationSeries) GetName() string {
	return as.Name
}

// GetStyle returns the line style.
func (as AnnotationSeries) GetStyle() Style {
	return as.Style
}

// GetYAxis returns which YAxis the series draws on.
func (as AnnotationSeries) GetYAxis() YAxisType {
	return as.YAxis
}

func (as AnnotationSeries) annotationStyleDefaults(defaults Style) Style {
	return Style{
		FontColor:   DefaultTextColor,
		Font:        defaults.Font,
		FillColor:   DefaultAnnotationFillColor,
		FontSize:    DefaultAnnotationFontSize,
		StrokeColor: defaults.StrokeColor,
		StrokeWidth: defaults.StrokeWidth,
		Padding:     DefaultAnnotationPadding,
	}
}

// Measure returns a bounds box of the series.
func (as AnnotationSeries) Measure(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) Box {
	box := Box{
		Top:    math.MaxInt32,
		Left:   math.MaxInt32,
		Right:  0,
		Bottom: 0,
	}
	if !as.Style.Hidden {
		seriesStyle := as.Style.InheritFrom(as.annotationStyleDefaults(defaults))
		for _, a := range as.Annotations {
			style := a.Style.InheritFrom(seriesStyle)
			lx := canvasBox.Left + xrange.Translate(a.XValue)
			ly := canvasBox.Bottom - yrange.Translate(a.YValue)
			ab := Draw.MeasureAnnotation(r, canvasBox, style, lx, ly, a.Label)
			box.Top = MinInt(box.Top, ab.Top)
			box.Left = MinInt(box.Left, ab.Left)
			box.Right = MaxInt(box.Right, ab.Right)
			box.Bottom = MaxInt(box.Bottom, ab.Bottom)
		}
	}
	return box
}

// Render draws the series.
func (as AnnotationSeries) Render(r Renderer, canvasBox Box, xrange, yrange Range, defaults Style) {
	if !as.Style.Hidden {
		seriesStyle := as.Style.InheritFrom(as.annotationStyleDefaults(defaults))
		for _, a := range as.Annotations {
			style := a.Style.InheritFrom(seriesStyle)
			lx := canvasBox.Left + xrange.Translate(a.XValue)
			ly := canvasBox.Bottom - yrange.Translate(a.YValue)
			Draw.Annotation(r, canvasBox, style, lx, ly, a.Label)
		}
	}
}

// Validate validates the series.
func (as AnnotationSeries) Validate() error {
	if len(as.Annotations) == 0 {
		return fmt.Errorf("annotation series requires annotations to be set and not empty")
	}
	return nil
}
