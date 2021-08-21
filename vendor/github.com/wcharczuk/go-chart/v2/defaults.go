package chart

const (
	// DefaultChartHeight is the default chart height.
	DefaultChartHeight = 400
	// DefaultChartWidth is the default chart width.
	DefaultChartWidth = 1024
	// DefaultStrokeWidth is the default chart stroke width.
	DefaultStrokeWidth = 0.0
	// DefaultDotWidth is the default chart dot width.
	DefaultDotWidth = 0.0
	// DefaultSeriesLineWidth is the default line width.
	DefaultSeriesLineWidth = 1.0
	// DefaultAxisLineWidth is the line width of the axis lines.
	DefaultAxisLineWidth = 1.0
	//DefaultDPI is the default dots per inch for the chart.
	DefaultDPI = 92.0
	// DefaultMinimumFontSize is the default minimum font size.
	DefaultMinimumFontSize = 8.0
	// DefaultFontSize is the default font size.
	DefaultFontSize = 10.0
	// DefaultTitleFontSize is the default title font size.
	DefaultTitleFontSize = 18.0
	// DefaultAnnotationDeltaWidth is the width of the left triangle out of annotations.
	DefaultAnnotationDeltaWidth = 10
	// DefaultAnnotationFontSize is the font size of annotations.
	DefaultAnnotationFontSize = 10.0
	// DefaultAxisFontSize is the font size of the axis labels.
	DefaultAxisFontSize = 10.0
	// DefaultTitleTop is the default distance from the top of the chart to put the title.
	DefaultTitleTop = 10

	// DefaultBackgroundStrokeWidth is the default stroke on the chart background.
	DefaultBackgroundStrokeWidth = 0.0
	// DefaultCanvasStrokeWidth is the default stroke on the chart canvas.
	DefaultCanvasStrokeWidth = 0.0

	// DefaultLineSpacing is the default vertical distance between lines of text.
	DefaultLineSpacing = 5

	// DefaultYAxisMargin is the default distance from the right of the canvas to the y axis labels.
	DefaultYAxisMargin = 10
	// DefaultXAxisMargin is the default distance from bottom of the canvas to the x axis labels.
	DefaultXAxisMargin = 10

	//DefaultVerticalTickHeight is half the margin.
	DefaultVerticalTickHeight = DefaultXAxisMargin >> 1
	//DefaultHorizontalTickWidth is half the margin.
	DefaultHorizontalTickWidth = DefaultYAxisMargin >> 1

	// DefaultTickCount is the default number of ticks to show
	DefaultTickCount = 10
	// DefaultTickCountSanityCheck is a hard limit on number of ticks to prevent infinite loops.
	DefaultTickCountSanityCheck = 1 << 10 //1024

	// DefaultMinimumTickHorizontalSpacing is the minimum distance between horizontal ticks.
	DefaultMinimumTickHorizontalSpacing = 20
	// DefaultMinimumTickVerticalSpacing is the minimum distance between vertical ticks.
	DefaultMinimumTickVerticalSpacing = 20

	// DefaultDateFormat is the default date format.
	DefaultDateFormat = "2006-01-02"
	// DefaultDateHourFormat is the date format for hour timestamp formats.
	DefaultDateHourFormat = "01-02 3PM"
	// DefaultDateMinuteFormat is the date format for minute range timestamp formats.
	DefaultDateMinuteFormat = "01-02 3:04PM"
	// DefaultFloatFormat is the default float format.
	DefaultFloatFormat = "%.2f"
	// DefaultPercentValueFormat is the default percent format.
	DefaultPercentValueFormat = "%0.2f%%"

	// DefaultBarSpacing is the default pixel spacing between bars.
	DefaultBarSpacing = 100
	// DefaultBarWidth is the default pixel width of bars in a bar chart.
	DefaultBarWidth = 50
)

var (
	// DashArrayDots is a dash array that represents '....' style stroke dashes.
	DashArrayDots = []int{1, 1}
	// DashArrayDashesSmall is a dash array that represents '- - -' style stroke dashes.
	DashArrayDashesSmall = []int{3, 3}
	// DashArrayDashesMedium is a dash array that represents '-- -- --' style stroke dashes.
	DashArrayDashesMedium = []int{5, 5}
	// DashArrayDashesLarge is a dash array that represents '----- ----- -----' style stroke dashes.
	DashArrayDashesLarge = []int{10, 10}
)

var (
	// DefaultAnnotationPadding is the padding around an annotation.
	DefaultAnnotationPadding = Box{Top: 5, Left: 5, Right: 5, Bottom: 5}

	// DefaultBackgroundPadding is the default canvas padding config.
	DefaultBackgroundPadding = Box{Top: 5, Left: 5, Right: 5, Bottom: 5}
)

const (
	// ContentTypePNG is the png mime type.
	ContentTypePNG = "image/png"

	// ContentTypeSVG is the svg mime type.
	ContentTypeSVG = "image/svg+xml"
)
