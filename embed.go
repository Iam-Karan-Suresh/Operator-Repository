package operator

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var rawStaticFiles embed.FS

// GetStaticFS returns the embedded React application files
func GetStaticFS() (fs.FS, error) {
	return fs.Sub(rawStaticFiles, "web/dist")
}
