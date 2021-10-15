package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var folder embed.FS
var StaticFolder, _ = fs.Sub(folder, "static")
