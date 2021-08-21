package drawing

// ImageFilter defines the type of filter to use
type ImageFilter int

const (
	// LinearFilter defines a linear filter
	LinearFilter ImageFilter = iota
	// BilinearFilter defines a bilinear filter
	BilinearFilter
	// BicubicFilter defines a bicubic filter
	BicubicFilter
)
