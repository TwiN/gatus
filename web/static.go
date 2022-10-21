package static

import "embed"

var (
	//go:embed static
	FileSystem embed.FS
)

const (
	RootPath  = "static"
	IndexPath = RootPath + "/index.html"
)
