package chart

import "fmt"

// FirstValueAnnotation returns an annotation series of just the first value of a value provider as an annotation.
func FirstValueAnnotation(innerSeries ValuesProvider, vfs ...ValueFormatter) AnnotationSeries {
	var vf ValueFormatter
	if len(vfs) > 0 {
		vf = vfs[0]
	} else if typed, isTyped := innerSeries.(ValueFormatterProvider); isTyped {
		_, vf = typed.GetValueFormatters()
	} else {
		vf = FloatValueFormatter
	}

	var firstValue Value2
	if typed, isTyped := innerSeries.(FirstValuesProvider); isTyped {
		firstValue.XValue, firstValue.YValue = typed.GetFirstValues()
		firstValue.Label = vf(firstValue.YValue)
	} else {
		firstValue.XValue, firstValue.YValue = innerSeries.GetValues(0)
		firstValue.Label = vf(firstValue.YValue)
	}

	var seriesName string
	var seriesStyle Style
	if typed, isTyped := innerSeries.(Series); isTyped {
		seriesName = fmt.Sprintf("%s - First Value", typed.GetName())
		seriesStyle = typed.GetStyle()
	}

	return AnnotationSeries{
		Name:        seriesName,
		Style:       seriesStyle,
		Annotations: []Value2{firstValue},
	}
}
