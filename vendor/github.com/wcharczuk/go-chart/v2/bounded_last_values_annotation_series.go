package chart

import "fmt"

// BoundedLastValuesAnnotationSeries returns a last value annotation series for a bounded values provider.
func BoundedLastValuesAnnotationSeries(innerSeries FullBoundedValuesProvider, vfs ...ValueFormatter) AnnotationSeries {
	lvx, lvy1, lvy2 := innerSeries.GetBoundedLastValues()

	var vf ValueFormatter
	if len(vfs) > 0 {
		vf = vfs[0]
	} else if typed, isTyped := innerSeries.(ValueFormatterProvider); isTyped {
		_, vf = typed.GetValueFormatters()
	} else {
		vf = FloatValueFormatter
	}

	label1 := vf(lvy1)
	label2 := vf(lvy2)

	var seriesName string
	var seriesStyle Style
	if typed, isTyped := innerSeries.(Series); isTyped {
		seriesName = fmt.Sprintf("%s - Last Values", typed.GetName())
		seriesStyle = typed.GetStyle()
	}

	return AnnotationSeries{
		Name:  seriesName,
		Style: seriesStyle,
		Annotations: []Value2{
			{XValue: lvx, YValue: lvy1, Label: label1},
			{XValue: lvx, YValue: lvy2, Label: label2},
		},
	}
}
