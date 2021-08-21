package chart

import (
	"fmt"
	"strings"

	"github.com/golang/freetype/truetype"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

const (
	// Disabled indicates if the value should be interpreted as set intentionally to zero.
	// this is because golang optionals aren't here yet.
	Disabled = -1
)

// Hidden is a prebuilt style with the `Hidden` property set to true.
func Hidden() Style {
	return Style{
		Hidden: true,
	}
}

// Shown is a prebuilt style with the `Hidden` property set to false.
// You can also think of this as the default.
func Shown() Style {
	return Style{
		Hidden: false,
	}
}

// StyleTextDefaults returns a style for drawing outside a
// chart context.
func StyleTextDefaults() Style {
	font, _ := GetDefaultFont()
	return Style{
		Hidden:    false,
		Font:      font,
		FontColor: DefaultTextColor,
		FontSize:  DefaultTitleFontSize,
	}
}

// Style is a simple style set.
type Style struct {
	Hidden  bool
	Padding Box

	ClassName string

	StrokeWidth     float64
	StrokeColor     drawing.Color
	StrokeDashArray []float64

	DotColor drawing.Color
	DotWidth float64

	DotWidthProvider SizeProvider
	DotColorProvider DotColorProvider

	FillColor drawing.Color

	FontSize  float64
	FontColor drawing.Color
	Font      *truetype.Font

	TextHorizontalAlign TextHorizontalAlign
	TextVerticalAlign   TextVerticalAlign
	TextWrap            TextWrap
	TextLineSpacing     int
	TextRotationDegrees float64 //0 is unset or normal
}

// IsZero returns if the object is set or not.
func (s Style) IsZero() bool {
	return !s.Hidden &&
		s.StrokeColor.IsZero() &&
		s.StrokeWidth == 0 &&
		s.DotColor.IsZero() &&
		s.DotWidth == 0 &&
		s.FillColor.IsZero() &&
		s.FontColor.IsZero() &&
		s.FontSize == 0 &&
		s.Font == nil &&
		s.ClassName == ""
}

// String returns a text representation of the style.
func (s Style) String() string {
	if s.IsZero() {
		return "{}"
	}

	var output []string
	if s.Hidden {
		output = []string{"\"hidden\": true"}
	} else {
		output = []string{"\"hidden\": false"}
	}

	if s.ClassName != "" {
		output = append(output, fmt.Sprintf("\"class_name\": %s", s.ClassName))
	} else {
		output = append(output, "\"class_name\": null")
	}

	if !s.Padding.IsZero() {
		output = append(output, fmt.Sprintf("\"padding\": %s", s.Padding.String()))
	} else {
		output = append(output, "\"padding\": null")
	}

	if s.StrokeWidth >= 0 {
		output = append(output, fmt.Sprintf("\"stroke_width\": %0.2f", s.StrokeWidth))
	} else {
		output = append(output, "\"stroke_width\": null")
	}

	if !s.StrokeColor.IsZero() {
		output = append(output, fmt.Sprintf("\"stroke_color\": %s", s.StrokeColor.String()))
	} else {
		output = append(output, "\"stroke_color\": null")
	}

	if len(s.StrokeDashArray) > 0 {
		var elements []string
		for _, v := range s.StrokeDashArray {
			elements = append(elements, fmt.Sprintf("%.2f", v))
		}
		dashArray := strings.Join(elements, ", ")
		output = append(output, fmt.Sprintf("\"stroke_dash_array\": [%s]", dashArray))
	} else {
		output = append(output, "\"stroke_dash_array\": null")
	}

	if s.DotWidth >= 0 {
		output = append(output, fmt.Sprintf("\"dot_width\": %0.2f", s.DotWidth))
	} else {
		output = append(output, "\"dot_width\": null")
	}

	if !s.DotColor.IsZero() {
		output = append(output, fmt.Sprintf("\"dot_color\": %s", s.DotColor.String()))
	} else {
		output = append(output, "\"dot_color\": null")
	}

	if !s.FillColor.IsZero() {
		output = append(output, fmt.Sprintf("\"fill_color\": %s", s.FillColor.String()))
	} else {
		output = append(output, "\"fill_color\": null")
	}

	if s.FontSize != 0 {
		output = append(output, fmt.Sprintf("\"font_size\": \"%0.2fpt\"", s.FontSize))
	} else {
		output = append(output, "\"font_size\": null")
	}

	if !s.FontColor.IsZero() {
		output = append(output, fmt.Sprintf("\"font_color\": %s", s.FontColor.String()))
	} else {
		output = append(output, "\"font_color\": null")
	}

	if s.Font != nil {
		output = append(output, fmt.Sprintf("\"font\": \"%s\"", s.Font.Name(truetype.NameIDFontFamily)))
	} else {
		output = append(output, "\"font_color\": null")
	}

	return "{" + strings.Join(output, ", ") + "}"
}

// GetClassName returns the class name or a default.
func (s Style) GetClassName(defaults ...string) string {
	if s.ClassName == "" {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return ""
	}
	return s.ClassName
}

// GetStrokeColor returns the stroke color.
func (s Style) GetStrokeColor(defaults ...drawing.Color) drawing.Color {
	if s.StrokeColor.IsZero() {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return drawing.ColorTransparent
	}
	return s.StrokeColor
}

// GetFillColor returns the fill color.
func (s Style) GetFillColor(defaults ...drawing.Color) drawing.Color {
	if s.FillColor.IsZero() {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return drawing.ColorTransparent
	}
	return s.FillColor
}

// GetDotColor returns the stroke color.
func (s Style) GetDotColor(defaults ...drawing.Color) drawing.Color {
	if s.DotColor.IsZero() {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return drawing.ColorTransparent
	}
	return s.DotColor
}

// GetStrokeWidth returns the stroke width.
func (s Style) GetStrokeWidth(defaults ...float64) float64 {
	if s.StrokeWidth == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultStrokeWidth
	}
	return s.StrokeWidth
}

// GetDotWidth returns the dot width for scatter plots.
func (s Style) GetDotWidth(defaults ...float64) float64 {
	if s.DotWidth == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultDotWidth
	}
	return s.DotWidth
}

// GetStrokeDashArray returns the stroke dash array.
func (s Style) GetStrokeDashArray(defaults ...[]float64) []float64 {
	if len(s.StrokeDashArray) == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return nil
	}
	return s.StrokeDashArray
}

// GetFontSize gets the font size.
func (s Style) GetFontSize(defaults ...float64) float64 {
	if s.FontSize == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultFontSize
	}
	return s.FontSize
}

// GetFontColor gets the font size.
func (s Style) GetFontColor(defaults ...drawing.Color) drawing.Color {
	if s.FontColor.IsZero() {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return drawing.ColorTransparent
	}
	return s.FontColor
}

// GetFont returns the font face.
func (s Style) GetFont(defaults ...*truetype.Font) *truetype.Font {
	if s.Font == nil {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return nil
	}
	return s.Font
}

// GetPadding returns the padding.
func (s Style) GetPadding(defaults ...Box) Box {
	if s.Padding.IsZero() {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return Box{}
	}
	return s.Padding
}

// GetTextHorizontalAlign returns the horizontal alignment.
func (s Style) GetTextHorizontalAlign(defaults ...TextHorizontalAlign) TextHorizontalAlign {
	if s.TextHorizontalAlign == TextHorizontalAlignUnset {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return TextHorizontalAlignUnset
	}
	return s.TextHorizontalAlign
}

// GetTextVerticalAlign returns the vertical alignment.
func (s Style) GetTextVerticalAlign(defaults ...TextVerticalAlign) TextVerticalAlign {
	if s.TextVerticalAlign == TextVerticalAlignUnset {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return TextVerticalAlignUnset
	}
	return s.TextVerticalAlign
}

// GetTextWrap returns the word wrap.
func (s Style) GetTextWrap(defaults ...TextWrap) TextWrap {
	if s.TextWrap == TextWrapUnset {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return TextWrapUnset
	}
	return s.TextWrap
}

// GetTextLineSpacing returns the spacing in pixels between lines of text (vertically).
func (s Style) GetTextLineSpacing(defaults ...int) int {
	if s.TextLineSpacing == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return DefaultLineSpacing
	}
	return s.TextLineSpacing
}

// GetTextRotationDegrees returns the text rotation in degrees.
func (s Style) GetTextRotationDegrees(defaults ...float64) float64 {
	if s.TextRotationDegrees == 0 {
		if len(defaults) > 0 {
			return defaults[0]
		}
	}
	return s.TextRotationDegrees
}

// WriteToRenderer passes the style's options to a renderer.
func (s Style) WriteToRenderer(r Renderer) {
	r.SetClassName(s.GetClassName())
	r.SetStrokeColor(s.GetStrokeColor())
	r.SetStrokeWidth(s.GetStrokeWidth())
	r.SetStrokeDashArray(s.GetStrokeDashArray())
	r.SetFillColor(s.GetFillColor())
	r.SetFont(s.GetFont())
	r.SetFontColor(s.GetFontColor())
	r.SetFontSize(s.GetFontSize())

	r.ClearTextRotation()
	if s.GetTextRotationDegrees() != 0 {
		r.SetTextRotation(DegreesToRadians(s.GetTextRotationDegrees()))
	}
}

// WriteDrawingOptionsToRenderer passes just the drawing style options to a renderer.
func (s Style) WriteDrawingOptionsToRenderer(r Renderer) {
	r.SetClassName(s.GetClassName())
	r.SetStrokeColor(s.GetStrokeColor())
	r.SetStrokeWidth(s.GetStrokeWidth())
	r.SetStrokeDashArray(s.GetStrokeDashArray())
	r.SetFillColor(s.GetFillColor())
}

// WriteTextOptionsToRenderer passes just the text style options to a renderer.
func (s Style) WriteTextOptionsToRenderer(r Renderer) {
	r.SetClassName(s.GetClassName())
	r.SetFont(s.GetFont())
	r.SetFontColor(s.GetFontColor())
	r.SetFontSize(s.GetFontSize())
}

// InheritFrom coalesces two styles into a new style.
func (s Style) InheritFrom(defaults Style) (final Style) {
	final.ClassName = s.GetClassName(defaults.ClassName)

	final.StrokeColor = s.GetStrokeColor(defaults.StrokeColor)
	final.StrokeWidth = s.GetStrokeWidth(defaults.StrokeWidth)
	final.StrokeDashArray = s.GetStrokeDashArray(defaults.StrokeDashArray)

	final.DotColor = s.GetDotColor(defaults.DotColor)
	final.DotWidth = s.GetDotWidth(defaults.DotWidth)

	final.DotWidthProvider = s.DotWidthProvider
	final.DotColorProvider = s.DotColorProvider

	final.FillColor = s.GetFillColor(defaults.FillColor)
	final.FontColor = s.GetFontColor(defaults.FontColor)
	final.FontSize = s.GetFontSize(defaults.FontSize)
	final.Font = s.GetFont(defaults.Font)
	final.Padding = s.GetPadding(defaults.Padding)
	final.TextHorizontalAlign = s.GetTextHorizontalAlign(defaults.TextHorizontalAlign)
	final.TextVerticalAlign = s.GetTextVerticalAlign(defaults.TextVerticalAlign)
	final.TextWrap = s.GetTextWrap(defaults.TextWrap)
	final.TextLineSpacing = s.GetTextLineSpacing(defaults.TextLineSpacing)
	final.TextRotationDegrees = s.GetTextRotationDegrees(defaults.TextRotationDegrees)

	return
}

// GetStrokeOptions returns the stroke components.
func (s Style) GetStrokeOptions() Style {
	return Style{
		ClassName:       s.ClassName,
		StrokeDashArray: s.StrokeDashArray,
		StrokeColor:     s.StrokeColor,
		StrokeWidth:     s.StrokeWidth,
	}
}

// GetFillOptions returns the fill components.
func (s Style) GetFillOptions() Style {
	return Style{
		ClassName: s.ClassName,
		FillColor: s.FillColor,
	}
}

// GetDotOptions returns the dot components.
func (s Style) GetDotOptions() Style {
	return Style{
		ClassName:       s.ClassName,
		StrokeDashArray: nil,
		FillColor:       s.DotColor,
		StrokeColor:     s.DotColor,
		StrokeWidth:     1.0,
	}
}

// GetFillAndStrokeOptions returns the fill and stroke components.
func (s Style) GetFillAndStrokeOptions() Style {
	return Style{
		ClassName:       s.ClassName,
		StrokeDashArray: s.StrokeDashArray,
		FillColor:       s.FillColor,
		StrokeColor:     s.StrokeColor,
		StrokeWidth:     s.StrokeWidth,
	}
}

// GetTextOptions returns just the text components of the style.
func (s Style) GetTextOptions() Style {
	return Style{
		ClassName:           s.ClassName,
		FontColor:           s.FontColor,
		FontSize:            s.FontSize,
		Font:                s.Font,
		TextHorizontalAlign: s.TextHorizontalAlign,
		TextVerticalAlign:   s.TextVerticalAlign,
		TextWrap:            s.TextWrap,
		TextLineSpacing:     s.TextLineSpacing,
		TextRotationDegrees: s.TextRotationDegrees,
	}
}

// ShouldDrawStroke tells drawing functions if they should draw the stroke.
func (s Style) ShouldDrawStroke() bool {
	return !s.StrokeColor.IsZero() && s.StrokeWidth > 0
}

// ShouldDrawDot tells drawing functions if they should draw the dot.
func (s Style) ShouldDrawDot() bool {
	return (!s.DotColor.IsZero() && s.DotWidth > 0) || s.DotColorProvider != nil || s.DotWidthProvider != nil
}

// ShouldDrawFill tells drawing functions if they should draw the stroke.
func (s Style) ShouldDrawFill() bool {
	return !s.FillColor.IsZero()
}
