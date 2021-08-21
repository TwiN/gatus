package chart

import (
	"strings"
)

// TextHorizontalAlign is an enum for the horizontal alignment options.
type TextHorizontalAlign int

const (
	// TextHorizontalAlignUnset is the unset state for text horizontal alignment.
	TextHorizontalAlignUnset TextHorizontalAlign = 0
	// TextHorizontalAlignLeft aligns a string horizontally so that it's left ligature starts at horizontal pixel 0.
	TextHorizontalAlignLeft TextHorizontalAlign = 1
	// TextHorizontalAlignCenter left aligns a string horizontally so that there are equal pixels
	// to the left and to the right of a string within a box.
	TextHorizontalAlignCenter TextHorizontalAlign = 2
	// TextHorizontalAlignRight right aligns a string horizontally so that the right ligature ends at the right-most pixel
	// of a box.
	TextHorizontalAlignRight TextHorizontalAlign = 3
)

// TextWrap is an enum for the word wrap options.
type TextWrap int

const (
	// TextWrapUnset is the unset state for text wrap options.
	TextWrapUnset TextWrap = 0
	// TextWrapNone will spill text past horizontal boundaries.
	TextWrapNone TextWrap = 1
	// TextWrapWord will split a string on words (i.e. spaces) to fit within a horizontal boundary.
	TextWrapWord TextWrap = 2
	// TextWrapRune will split a string on a rune (i.e. utf-8 codepage) to fit within a horizontal boundary.
	TextWrapRune TextWrap = 3
)

// TextVerticalAlign is an enum for the vertical alignment options.
type TextVerticalAlign int

const (
	// TextVerticalAlignUnset is the unset state for vertical alignment options.
	TextVerticalAlignUnset TextVerticalAlign = 0
	// TextVerticalAlignBaseline aligns text according to the "baseline" of the string, or where a normal ascender begins.
	TextVerticalAlignBaseline TextVerticalAlign = 1
	// TextVerticalAlignBottom aligns the text according to the lowers pixel of any of the ligatures (ex. g or q both extend below the baseline).
	TextVerticalAlignBottom TextVerticalAlign = 2
	// TextVerticalAlignMiddle aligns the text so that there is an equal amount of space above and below the top and bottom of the ligatures.
	TextVerticalAlignMiddle TextVerticalAlign = 3
	// TextVerticalAlignMiddleBaseline aligns the text vertically so that there is an equal number of pixels above and below the baseline of the string.
	TextVerticalAlignMiddleBaseline TextVerticalAlign = 4
	// TextVerticalAlignTop alignts the text so that the top of the ligatures are at y-pixel 0 in the container.
	TextVerticalAlignTop TextVerticalAlign = 5
)

var (
	// Text contains utilities for text.
	Text = &text{}
)

// TextStyle encapsulates text style options.
type TextStyle struct {
	HorizontalAlign TextHorizontalAlign
	VerticalAlign   TextVerticalAlign
	Wrap            TextWrap
}

type text struct{}

func (t text) WrapFit(r Renderer, value string, width int, style Style) []string {
	switch style.TextWrap {
	case TextWrapRune:
		return t.WrapFitRune(r, value, width, style)
	case TextWrapWord:
		return t.WrapFitWord(r, value, width, style)
	}
	return []string{value}
}

func (t text) WrapFitWord(r Renderer, value string, width int, style Style) []string {
	style.WriteToRenderer(r)

	var output []string
	var line string
	var word string

	var textBox Box

	for _, c := range value {
		if c == rune('\n') { // commit the line to output
			output = append(output, t.Trim(line+word))
			line = ""
			word = ""
			continue
		}

		textBox = r.MeasureText(line + word + string(c))

		if textBox.Width() >= width {
			output = append(output, t.Trim(line))
			line = word
			word = string(c)
			continue
		}

		if c == rune(' ') || c == rune('\t') {
			line = line + word + string(c)
			word = ""
			continue
		}
		word = word + string(c)
	}

	return append(output, t.Trim(line+word))
}

func (t text) WrapFitRune(r Renderer, value string, width int, style Style) []string {
	style.WriteToRenderer(r)

	var output []string
	var line string
	var textBox Box
	for _, c := range value {
		if c == rune('\n') {
			output = append(output, line)
			line = ""
			continue
		}

		textBox = r.MeasureText(line + string(c))

		if textBox.Width() >= width {
			output = append(output, line)
			line = string(c)
			continue
		}
		line = line + string(c)
	}
	return t.appendLast(output, line)
}

func (t text) Trim(value string) string {
	return strings.Trim(value, " \t\n\r")
}

func (t text) MeasureLines(r Renderer, lines []string, style Style) Box {
	style.WriteTextOptionsToRenderer(r)
	var output Box
	for index, line := range lines {
		lineBox := r.MeasureText(line)
		output.Right = MaxInt(lineBox.Right, output.Right)
		output.Bottom += lineBox.Height()
		if index < len(lines)-1 {
			output.Bottom += +style.GetTextLineSpacing()
		}
	}
	return output
}

func (t text) appendLast(lines []string, text string) []string {
	if len(lines) == 0 {
		return []string{text}
	}
	lastLine := lines[len(lines)-1]
	lines[len(lines)-1] = lastLine + text
	return lines
}
