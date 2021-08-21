package chart

import (
	"bytes"
	"errors"
	"image"
	"image/png"
)

// RGBACollector is a render target for a chart.
type RGBACollector interface {
	SetRGBA(i *image.RGBA)
}

// ImageWriter is a special type of io.Writer that produces a final image.
type ImageWriter struct {
	rgba     *image.RGBA
	contents *bytes.Buffer
}

func (ir *ImageWriter) Write(buffer []byte) (int, error) {
	if ir.contents == nil {
		ir.contents = bytes.NewBuffer([]byte{})
	}
	return ir.contents.Write(buffer)
}

// SetRGBA sets a raw version of the image.
func (ir *ImageWriter) SetRGBA(i *image.RGBA) {
	ir.rgba = i
}

// Image returns an *image.Image for the result.
func (ir *ImageWriter) Image() (image.Image, error) {
	if ir.rgba != nil {
		return ir.rgba, nil
	}
	if ir.contents != nil && ir.contents.Len() > 0 {
		return png.Decode(ir.contents)
	}
	return nil, errors.New("no valid sources for image data, cannot continue")
}
